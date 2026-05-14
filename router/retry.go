package router

import (
	"math/rand/v2"
	"strings"
	"time"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	OnRetry    func(RetryEvent)
}

type RetryEvent struct {
	Err        error
	Attempt    int
	MaxRetries int
	Delay      time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, code := range []string{"429", "500", "502", "503", "529"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	for _, pattern := range []string{"timeout", "deadline exceeded", "connection refused", "EOF", "reset by peer", "broken pipe"} {
		if strings.Contains(strings.ToLower(msg), pattern) {
			return true
		}
	}
	return false
}

func BackoffDelay(attempt int, cfg RetryConfig) time.Duration {
	base := cfg.BaseDelay
	for i := 0; i < attempt; i++ {
		base *= 2
	}
	if base > cfg.MaxDelay {
		base = cfg.MaxDelay
	}
	jitter := 0.5 + rand.Float64()
	return time.Duration(float64(base) * jitter)
}

var afterFunc = func(d time.Duration) <-chan time.Time {
	return time.After(d)
}
