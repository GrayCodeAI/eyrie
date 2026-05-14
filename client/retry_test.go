package client

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.BaseDelay != 500*time.Millisecond {
		t.Errorf("BaseDelay = %v, want 500ms", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", cfg.MaxDelay)
	}
	if len(cfg.RetryOn) != 5 {
		t.Fatalf("RetryOn has %d codes, want 5", len(cfg.RetryOn))
	}
	expected := []int{429, 500, 502, 503, 529}
	for i, code := range expected {
		if cfg.RetryOn[i] != code {
			t.Errorf("RetryOn[%d] = %d, want %d", i, cfg.RetryOn[i], code)
		}
	}
}

func TestRetryShouldRetryTrue(t *testing.T) {
	cfg := DefaultRetryConfig()
	codes := []int{429, 500, 502, 503, 529}
	for _, code := range codes {
		if !cfg.shouldRetry(code) {
			t.Errorf("shouldRetry(%d) = false, want true", code)
		}
	}
}

func TestRetryShouldRetryFalse(t *testing.T) {
	cfg := DefaultRetryConfig()
	codes := []int{400, 401, 403, 404}
	for _, code := range codes {
		if cfg.shouldRetry(code) {
			t.Errorf("shouldRetry(%d) = true, want false", code)
		}
	}
}

func TestRetryDelayIncreasesExponentially(t *testing.T) {
	cfg := RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  10 * time.Second,
	}

	// Run multiple samples to confirm the trend despite jitter
	samples := 100
	var avg0, avg1, avg2 time.Duration
	for i := 0; i < samples; i++ {
		avg0 += cfg.backoffDelay(0, nil)
		avg1 += cfg.backoffDelay(1, nil)
		avg2 += cfg.backoffDelay(2, nil)
	}
	avg0 /= time.Duration(samples)
	avg1 /= time.Duration(samples)
	avg2 /= time.Duration(samples)

	// With full jitter, average should be half the exponential cap:
	// attempt 0: base*2^0 = 100ms, avg jitter ~50ms
	// attempt 1: base*2^1 = 200ms, avg jitter ~100ms
	// attempt 2: base*2^2 = 400ms, avg jitter ~200ms
	if avg1 <= avg0 {
		t.Errorf("average delay for attempt 1 (%v) should exceed attempt 0 (%v)", avg1, avg0)
	}
	if avg2 <= avg1 {
		t.Errorf("average delay for attempt 2 (%v) should exceed attempt 1 (%v)", avg2, avg1)
	}
}

func TestRetryParseRetryAfterHeader(t *testing.T) {
	cfg := RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  60 * time.Second,
	}

	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "5")

	delay := cfg.backoffDelay(0, resp)
	if delay != 5*time.Second {
		t.Errorf("backoffDelay with Retry-After=5 returned %v, want 5s", delay)
	}
}

func TestRetryParseRetryDelayFromMessage(t *testing.T) {
	tests := []struct {
		msg  string
		want time.Duration
	}{
		{"retry after 2 seconds", 2 * time.Second},
		{"try again in 500 ms", 500 * time.Millisecond},
		{"retry in 1.5 seconds", 1500 * time.Millisecond},
		{"no delay hint here", 0},
	}
	for _, tc := range tests {
		got := parseRetryDelay(tc.msg)
		if got != tc.want {
			t.Errorf("parseRetryDelay(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}
