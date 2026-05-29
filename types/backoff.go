package types

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

// BackoffDelay calculates an exponential backoff delay with full jitter.
// This is the shared backoff implementation used by both client and router retry logic.
//
// Parameters:
//   - attempt: the current retry attempt (0-based)
//   - baseDelay: the base delay duration
//   - maxDelay: the maximum delay duration cap
func BackoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	exp := math.Pow(2, float64(attempt))
	delay := time.Duration(float64(baseDelay) * exp)
	if delay > maxDelay {
		delay = maxDelay
	}
	return CryptoRandDuration(delay)
}

// CryptoRandDuration returns a cryptographically random duration in [0, max).
func CryptoRandDuration(max time.Duration) time.Duration {
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
