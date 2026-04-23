package security

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
)

const (
	defaultLoginMaxFailures   = 5
	defaultLoginFailureWindow = 15 * time.Minute
	defaultLoginLockDuration  = 15 * time.Minute
)

type LoginAttemptLimiterOptions struct {
	MaxFailures   int
	FailureWindow time.Duration
	LockDuration  time.Duration
	Now           func() time.Time
}

type LoginAttemptLimiter struct {
	mu            sync.Mutex
	maxFailures   int
	failureWindow time.Duration
	lockDuration  time.Duration
	now           func() time.Time
	states        map[string]*loginAttemptState
}

type loginAttemptState struct {
	failures    []time.Time
	lockedUntil time.Time
}

func NewLoginAttemptLimiter(options LoginAttemptLimiterOptions) *LoginAttemptLimiter {
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

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &LoginAttemptLimiter{
		maxFailures:   maxFailures,
		failureWindow: failureWindow,
		lockDuration:  lockDuration,
		now:           now,
		states:        make(map[string]*loginAttemptState),
	}
}

func BuildLoginAttemptKey(username string, ip string) string {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	if normalizedUsername == "" {
		normalizedUsername = "unknown"
	}

	normalizedIP := strings.TrimSpace(ip)
	if normalizedIP == "" {
		normalizedIP = "unknown"
	}

	return normalizedUsername + "|" + normalizedIP
}

func (l *LoginAttemptLimiter) Allow(key string) error {
	if key == "" {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.states[key]
	if !ok {
		return nil
	}

	now := l.now().UTC()
	l.pruneFailures(state, now)

	if state.lockedUntil.After(now) {
		remaining := state.lockedUntil.Sub(now).Round(time.Second)
		if remaining < time.Second {
			remaining = time.Second
		}
		return apperror.Wrap(
			apperror.ErrLoginRateLimited.Code,
			apperror.ErrLoginRateLimited.HTTPStatus,
			apperror.ErrLoginRateLimited.Message,
			fmt.Errorf("retry after %s", remaining),
		)
	}

	if len(state.failures) == 0 {
		delete(l.states, key)
	}
	return nil
}

func (l *LoginAttemptLimiter) RecordFailure(key string) {
	if key == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	state, ok := l.states[key]
	if !ok {
		state = &loginAttemptState{}
		l.states[key] = state
	}

	l.pruneFailures(state, now)
	state.failures = append(state.failures, now)
	if len(state.failures) >= l.maxFailures {
		state.failures = nil
		state.lockedUntil = now.Add(l.lockDuration)
	}
}

func (l *LoginAttemptLimiter) RecordSuccess(key string) {
	if key == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.states, key)
}

func (l *LoginAttemptLimiter) pruneFailures(state *loginAttemptState, now time.Time) {
	if state == nil {
		return
	}

	threshold := now.Add(-l.failureWindow)
	filtered := state.failures[:0]
	for _, item := range state.failures {
		if item.After(threshold) {
			filtered = append(filtered, item)
		}
	}
	state.failures = filtered
}
