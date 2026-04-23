package security

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
)

const (
	defaultLoginAttemptRedisPrefix    = "snowpanel:auth:attempt"
	defaultLoginAttemptCommandTimeout = 500 * time.Millisecond
)

var recordFailureScript = redis.NewScript(`
local lockKey = KEYS[1]
local failuresKey = KEYS[2]
local maxFailures = tonumber(ARGV[1])
local failureWindowMs = tonumber(ARGV[2])
local lockDurationMs = tonumber(ARGV[3])

if redis.call("EXISTS", lockKey) == 1 then
	return 1
end

local count = redis.call("INCR", failuresKey)
if count == 1 then
	redis.call("PEXPIRE", failuresKey, failureWindowMs)
end

if count >= maxFailures then
	redis.call("SET", lockKey, "1", "PX", lockDurationMs)
	redis.call("DEL", failuresKey)
	return 1
end

return 0
`)

type RedisLoginAttemptLimiterOptions struct {
	MaxFailures    int
	FailureWindow  time.Duration
	LockDuration   time.Duration
	KeyPrefix      string
	CommandTimeout time.Duration
	Now            func() time.Time
}

type RedisLoginAttemptLimiter struct {
	client         redis.UniversalClient
	maxFailures    int
	failureWindow  time.Duration
	lockDuration   time.Duration
	keyPrefix      string
	commandTimeout time.Duration
	fallback       *LoginAttemptLimiter
}

func NewRedisLoginAttemptLimiter(
	client redis.UniversalClient,
	options RedisLoginAttemptLimiterOptions,
) *RedisLoginAttemptLimiter {
	maxFailures := options.MaxFailures
	if maxFailures <= 0 {
		maxFailures = defaultLoginMaxFailures
	}

	failureWindow := options.FailureWindow
	if failureWindow <= 0 {
		failureWindow = defaultLoginFailureWindow
	}

	lockDuration := options.LockDuration
	if lockDuration <= 0 {
		lockDuration = defaultLoginLockDuration
	}

	commandTimeout := options.CommandTimeout
	if commandTimeout <= 0 {
		commandTimeout = defaultLoginAttemptCommandTimeout
	}

	keyPrefix := strings.TrimSpace(options.KeyPrefix)
	if keyPrefix == "" {
		keyPrefix = defaultLoginAttemptRedisPrefix
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &RedisLoginAttemptLimiter{
		client:         client,
		maxFailures:    maxFailures,
		failureWindow:  failureWindow,
		lockDuration:   lockDuration,
		keyPrefix:      keyPrefix,
		commandTimeout: commandTimeout,
		fallback: NewLoginAttemptLimiter(LoginAttemptLimiterOptions{
			MaxFailures:   maxFailures,
			FailureWindow: failureWindow,
			LockDuration:  lockDuration,
			Now:           now,
		}),
	}
}

func (l *RedisLoginAttemptLimiter) Allow(key string) error {
	if key == "" {
		return nil
	}
	if l.client == nil {
		return l.fallback.Allow(key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.commandTimeout)
	defer cancel()

	ttl, err := l.client.PTTL(ctx, l.lockKey(key)).Result()
	if err != nil && err != redis.Nil {
		return l.fallback.Allow(key)
	}
	if ttl > 0 || ttl == -1 {
		if ttl < time.Second {
			ttl = time.Second
		}
		if ttl == -1 {
			ttl = l.lockDuration
		}
		return apperror.Wrap(
			apperror.ErrLoginRateLimited.Code,
			apperror.ErrLoginRateLimited.HTTPStatus,
			apperror.ErrLoginRateLimited.Message,
			fmt.Errorf("retry after %s", ttl.Round(time.Second)),
		)
	}
	return nil
}

func (l *RedisLoginAttemptLimiter) RecordFailure(key string) {
	if key == "" {
		return
	}
	l.fallback.RecordFailure(key)
	if l.client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.commandTimeout)
	defer cancel()

	_, err := recordFailureScript.Run(
		ctx,
		l.client,
		[]string{l.lockKey(key), l.failuresKey(key)},
		l.maxFailures,
		l.failureWindow.Milliseconds(),
		l.lockDuration.Milliseconds(),
	).Result()
	if err != nil {
		return
	}
}

func (l *RedisLoginAttemptLimiter) RecordSuccess(key string) {
	if key == "" {
		return
	}
	l.fallback.RecordSuccess(key)
	if l.client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.commandTimeout)
	defer cancel()
	_ = l.client.Del(ctx, l.lockKey(key), l.failuresKey(key)).Err()
}

func (l *RedisLoginAttemptLimiter) lockKey(attemptKey string) string {
	return l.keyPrefix + ":lock:" + attemptKey
}

func (l *RedisLoginAttemptLimiter) failuresKey(attemptKey string) string {
	return l.keyPrefix + ":failures:" + attemptKey
}
