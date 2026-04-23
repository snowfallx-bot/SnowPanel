package security

import (
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
)

func TestBuildLoginAttemptKey(t *testing.T) {
	key := BuildLoginAttemptKey(" Admin ", " 127.0.0.1 ")
	if key != "admin|127.0.0.1" {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestLoginAttemptLimiterLocksAfterMaxFailures(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := now
	limiter := NewLoginAttemptLimiter(LoginAttemptLimiterOptions{
		MaxFailures:   3,
		FailureWindow: 10 * time.Minute,
		LockDuration:  5 * time.Minute,
		Now: func() time.Time {
			return clock
		},
	})

	key := BuildLoginAttemptKey("admin", "127.0.0.1")
	limiter.RecordFailure(key)
	limiter.RecordFailure(key)
	limiter.RecordFailure(key)

	err := limiter.Allow(key)
	if err == nil {
		t.Fatalf("expected limiter to block")
	}

	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrLoginRateLimited.Code {
		t.Fatalf("unexpected code: %d", appErr.Code)
	}

	clock = clock.Add(6 * time.Minute)
	if err := limiter.Allow(key); err != nil {
		t.Fatalf("expected limiter to allow after lock duration, got %v", err)
	}
}

func TestLoginAttemptLimiterResetOnSuccess(t *testing.T) {
	limiter := NewLoginAttemptLimiter(LoginAttemptLimiterOptions{
		MaxFailures:   2,
		FailureWindow: 10 * time.Minute,
		LockDuration:  10 * time.Minute,
		Now:           time.Now,
	})

	key := BuildLoginAttemptKey("admin", "127.0.0.1")
	limiter.RecordFailure(key)
	limiter.RecordSuccess(key)
	if err := limiter.Allow(key); err != nil {
		t.Fatalf("expected limiter allow after success reset, got %v", err)
	}
}

func TestLoginAttemptLimiterWindowExpiry(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := base
	limiter := NewLoginAttemptLimiter(LoginAttemptLimiterOptions{
		MaxFailures:   2,
		FailureWindow: 2 * time.Minute,
		LockDuration:  10 * time.Minute,
		Now: func() time.Time {
			return clock
		},
	})

	key := BuildLoginAttemptKey("admin", "127.0.0.1")
	limiter.RecordFailure(key)

	clock = base.Add(3 * time.Minute)
	limiter.RecordFailure(key)
	if err := limiter.Allow(key); err != nil {
		t.Fatalf("expected limiter allow because first failure expired, got %v", err)
	}
}
