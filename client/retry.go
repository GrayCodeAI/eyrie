package client

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/GrayCodeAI/eyrie/types"
)

// RetryConfig controls retry behavior for HTTP clients.
// It embeds types.RetryConfig for the core fields and adds RetryOn for
// HTTP-status-code–driven retry decisions.
type RetryConfig struct {
	types.RetryConfig
	RetryOn []int // HTTP status codes to retry on
}

// NewRetryConfig constructs a RetryConfig from core fields and optional
// HTTP status codes to retry on.
func NewRetryConfig(maxRetries int, baseDelay, maxDelay time.Duration, retryOn ...int) RetryConfig {
	return RetryConfig{
		RetryConfig: types.RetryConfig{MaxRetries: maxRetries, BaseDelay: baseDelay, MaxDelay: maxDelay},
		RetryOn:     retryOn,
	}
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return NewRetryConfig(3, 500*time.Millisecond, 30*time.Second, 429, 500, 502, 503, 529)
}

// shouldRetry checks if a status code is retryable.
func (rc RetryConfig) shouldRetry(statusCode int) bool {
	for _, code := range rc.RetryOn {
		if code == statusCode {
			return true
		}
	}
	return false
}

// backoffDelay calculates delay with exponential backoff + jitter.
// Respects Retry-After headers from HTTP responses when present.
func (rc RetryConfig) backoffDelay(attempt int, resp *http.Response) time.Duration {
	// Respect Retry-After header if present
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				d := time.Duration(secs) * time.Second
				if d > rc.MaxDelay {
					d = rc.MaxDelay
				}
				return d
			}
			if t, err := http.ParseTime(ra); err == nil {
				d := time.Until(t)
				if d > rc.MaxDelay {
					d = rc.MaxDelay
				}
				if d > 0 {
					return d
				}
			}
		}
	}

	// Use shared exponential backoff with full jitter
	return types.BackoffDelay(attempt, rc.BaseDelay, rc.MaxDelay)
}

var retryDelayRe = regexp.MustCompile(`(?i)(?:retry|try again)\s+(?:in|after)\s+(\d+(?:\.\d+)?)\s*(ms|milliseconds?|s|seconds?)`)

// parseRetryDelay extracts a delay hint from an error message.
func parseRetryDelay(errMsg string) time.Duration {
	m := retryDelayRe.FindStringSubmatch(errMsg)
	if m == nil {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	switch {
	case len(m[2]) > 0 && m[2][0] == 'm':
		return time.Duration(val * float64(time.Millisecond))
	default:
		return time.Duration(val * float64(time.Second))
	}
}

// doWithRetry executes an HTTP request with retry logic.
//
// Note: doWithRetry operates at the transport layer, before
// formatAPIError constructs *EyrieError. Structured-error awareness
// lives in the fallback chain (fallback.go:240-244) where
// *EyrieError.IsRetriable() / IsAuthError() drive provider
// rotation. doWithRetry only needs the raw transport status code
// and the underlying network error to decide whether to retry the
// same request.
func doWithRetry(ctx context.Context, httpClient *http.Client, req *http.Request, rc RetryConfig, logger *slog.Logger) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := rc.backoffDelay(attempt-1, lastResp)
			if lastErr != nil {
				if parsed := parseRetryDelay(lastErr.Error()); parsed > delay {
					delay = parsed
				}
			}
			logger.Debug(
				"retrying request",
				"attempt", attempt, "max", rc.MaxRetries,
				"delay", delay, "url", req.URL.String(),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Clone request body for retry (body may have been consumed)
		retryReq := req.Clone(ctx)
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to clone request body: %w", err)
			}
			retryReq.Body = body
		}

		resp, err := httpClient.Do(retryReq)
		if err != nil {
			lastErr = err
			logger.Warn("request failed", "attempt", attempt, "error", err)
			continue
		}

		if !rc.shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp
		lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, req.URL.String())
		logger.Warn("retryable status", "attempt", attempt, "status", resp.StatusCode)
		_ = resp.Body.Close()
	}

	return nil, fmt.Errorf("eyrie: %s: max retries (%d) exceeded for %s: %w", req.URL.Host, rc.MaxRetries, req.URL.Path, lastErr)
}
