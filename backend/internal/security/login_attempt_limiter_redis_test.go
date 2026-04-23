package security

import (
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
)

func TestRedisLoginAttemptLimiterLocksAfterMaxFailures(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	limiter := NewRedisLoginAttemptLimiter(client, RedisLoginAttemptLimiterOptions{
		MaxFailures:   3,
		FailureWindow: 10 * time.Minute,
		LockDuration:  5 * time.Minute,
	})
	key := BuildLoginAttemptKey("admin", "127.0.0.1")

	limiter.RecordFailure(key)
	limiter.RecordFailure(key)
	limiter.RecordFailure(key)

	err := limiter.Allow(key)
	if err == nil {
		t.Fatalf("expected limiter block after max failures")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.ErrLoginRateLimited.Code {
		t.Fatalf("expected rate limit app error, got %v", err)
	}

	server.FastForward(6 * time.Minute)
	if err := limiter.Allow(key); err != nil {
		t.Fatalf("expected limiter allow after lock expiry, got %v", err)
	}
}

func TestRedisLoginAttemptLimiterRecordSuccessClearsLock(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	limiter := NewRedisLoginAttemptLimiter(client, RedisLoginAttemptLimiterOptions{
		MaxFailures:   2,
		FailureWindow: 10 * time.Minute,
		LockDuration:  10 * time.Minute,
	})
	key := BuildLoginAttemptKey("admin", "127.0.0.1")

	limiter.RecordFailure(key)
	limiter.RecordFailure(key)
	if err := limiter.Allow(key); err == nil {
		t.Fatalf("expected lock before success reset")
	}

	limiter.RecordSuccess(key)
	if err := limiter.Allow(key); err != nil {
		t.Fatalf("expected allow after success reset, got %v", err)
	}
}

func TestRedisLoginAttemptLimiterFallsBackToMemoryOnRedisError(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1",
	})
	t.Cleanup(func() {
		_ = client.Close()
	})

	limiter := NewRedisLoginAttemptLimiter(client, RedisLoginAttemptLimiterOptions{
		MaxFailures:    2,
		FailureWindow:  10 * time.Minute,
		LockDuration:   5 * time.Minute,
		CommandTimeout: 20 * time.Millisecond,
	})
	key := BuildLoginAttemptKey("admin", "127.0.0.1")

	limiter.RecordFailure(key)
	limiter.RecordFailure(key)

	err := limiter.Allow(key)
	if err == nil {
		t.Fatalf("expected fallback in-memory limiter to block")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.ErrLoginRateLimited.Code {
		t.Fatalf("expected rate limit app error, got %v", err)
	}
}
