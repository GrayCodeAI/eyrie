package router

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/types"
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
	return types.IsTransient(err)
}

// ShouldTryNextDeployment reports billing/credit errors where another deployment may succeed.
func ShouldTryNextDeployment(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	for _, pattern := range []string{
		"requires more credits", "can only afford", "insufficient credits",
		"insufficient balance", "payment required", "out of credits", "402",
	} {
		if strings.Contains(low, pattern) {
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
	return cryptoRandDuration(base)
}

func cryptoRandDuration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	bigN := big.NewInt(int64(max))
	result, err := rand.Int(rand.Reader, bigN)
	if err != nil {
		return 0
	}
	return time.Duration(result.Int64())
}

var afterFunc = time.After
