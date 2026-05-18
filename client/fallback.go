package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// FallbackProvider wraps multiple Providers and automatically falls back to the
// next one when the current provider returns a retriable error (429, 500, 502,
// 503, timeout). It does NOT fall back on client errors (400, 401, 403) because
// those indicate a problem with the request itself, not the provider.
//
// Inspired by BerriAI/litellm's fallback chain feature.
//
// FallbackProvider is safe for concurrent use.
type FallbackProvider struct {
	providers []Provider
	logger    *slog.Logger

	// stats tracks how many times each provider served a request.
	mu    sync.RWMutex
	stats map[string]*atomic.Int64
}

// Compile-time check that FallbackProvider implements Provider.
var _ Provider = (*FallbackProvider)(nil)

// NewFallbackProvider creates a FallbackProvider that tries providers in order.
// At least one provider must be supplied.
func NewFallbackProvider(providers ...Provider) *FallbackProvider {
	if len(providers) == 0 {
		panic("eyrie: FallbackProvider requires at least one provider")
	}
	stats := make(map[string]*atomic.Int64, len(providers))
	for _, p := range providers {
		if _, ok := stats[p.Name()]; !ok {
			stats[p.Name()] = &atomic.Int64{}
		}
	}
	return &FallbackProvider{
		providers: providers,
		logger:    slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
		stats:     stats,
	}
}

// SetLogger sets a custom logger for the FallbackProvider.
func (fp *FallbackProvider) SetLogger(l *slog.Logger) {
	fp.mu.Lock()
	fp.logger = l
	fp.mu.Unlock()
}

// Name returns a composite name listing all providers in the chain.
func (fp *FallbackProvider) Name() string {
	names := make([]string, len(fp.providers))
	for i, p := range fp.providers {
		names[i] = p.Name()
	}
	return "fallback(" + strings.Join(names, "->") + ")"
}

// Ping tries to ping each provider in order, returning nil on the first success.
func (fp *FallbackProvider) Ping(ctx context.Context) error {
	var lastErr error
	for _, p := range fp.providers {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.Ping(ctx); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("eyrie: all providers failed ping: %w", lastErr)
}

// Chat sends a non-streaming chat request, falling back through the provider
// chain on retriable errors. Returns the first successful response.
func (fp *FallbackProvider) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	var lastErr error

	for i, p := range fp.providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		fp.logger.Debug(
			"fallback: trying provider",
			"provider", p.Name(),
			"index", i,
			"total", len(fp.providers),
		)

		resp, err := p.Chat(ctx, messages, opts)
		if err == nil {
			fp.recordSuccess(p.Name())
			fp.logger.Debug(
				"fallback: provider succeeded",
				"provider", p.Name(),
				"index", i,
			)
			return resp, nil
		}

		// Check if the error is retriable; if not, return immediately.
		if !isRetriableError(err) {
			fp.logger.Warn(
				"fallback: non-retriable error, not falling back",
				"provider", p.Name(),
				"error", err,
			)
			return nil, err
		}

		fp.logger.Warn(
			"fallback: provider failed, trying next",
			"provider", p.Name(),
			"index", i,
			"error", err,
		)
		lastErr = err
	}

	return nil, fmt.Errorf("eyrie: all %d fallback providers failed: %w", len(fp.providers), lastErr)
}

// StreamChat sends a streaming chat request, falling back through the provider
// chain on retriable errors. Returns the first successful stream.
func (fp *FallbackProvider) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	var lastErr error

	for i, p := range fp.providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		fp.logger.Debug(
			"fallback: trying provider for stream",
			"provider", p.Name(),
			"index", i,
			"total", len(fp.providers),
		)

		sr, err := p.StreamChat(ctx, messages, opts)
		if err == nil {
			fp.recordSuccess(p.Name())
			fp.logger.Debug(
				"fallback: provider stream succeeded",
				"provider", p.Name(),
				"index", i,
			)
			return sr, nil
		}

		if !isRetriableError(err) {
			fp.logger.Warn(
				"fallback: non-retriable stream error, not falling back",
				"provider", p.Name(),
				"error", err,
			)
			return nil, err
		}

		fp.logger.Warn(
			"fallback: provider stream failed, trying next",
			"provider", p.Name(),
			"index", i,
			"error", err,
		)
		lastErr = err
	}

	return nil, fmt.Errorf("eyrie: all %d fallback providers failed streaming: %w", len(fp.providers), lastErr)
}

// Stats returns a snapshot of how many times each provider served a request.
func (fp *FallbackProvider) Stats() map[string]int64 {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	result := make(map[string]int64, len(fp.stats))
	for name, counter := range fp.stats {
		result[name] = counter.Load()
	}
	return result
}

func (fp *FallbackProvider) recordSuccess(name string) {
	fp.mu.RLock()
	counter, ok := fp.stats[name]
	fp.mu.RUnlock()
	if ok {
		counter.Add(1)
	}
}

// httpStatusRe extracts HTTP status codes from error messages produced by
// the eyrie client (e.g. "eyrie: openai API error (request_id=...): ..."
// or "HTTP 429 from ...").
var httpStatusRe = regexp.MustCompile(`(?:HTTP|http)\s+(\d{3})`)

// retriableStatusCodes are HTTP status codes that indicate a provider-side
// or transient issue worth retrying on a different provider.
var retriableStatusCodes = map[int]bool{
	429: true, // rate limited
	500: true, // internal server error
	502: true, // bad gateway
	503: true, // service unavailable
	529: true, // overloaded (Anthropic)
}

// nonRetriableStatusCodes are HTTP status codes that indicate a client-side
// error -- falling back won't help because the request itself is bad.
var nonRetriableStatusCodes = map[int]bool{
	400: true, // bad request
	401: true, // unauthorized
	403: true, // forbidden
	404: true, // not found (wrong model name, etc.)
	422: true, // unprocessable entity
}

// isRetriableError inspects an error to determine if a fallback should be attempted.
// It extracts HTTP status codes from the error message format used by eyrie providers,
// and also treats context timeouts as retriable (the next provider may respond faster).
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Context deadline exceeded is retriable (the next provider may be faster).
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Context cancelled is NOT retriable (the caller wants to stop).
	if errors.Is(err, context.Canceled) {
		return false
	}

	errMsg := err.Error()

	// Try to extract an HTTP status code from the error message.
	if matches := httpStatusRe.FindStringSubmatch(errMsg); len(matches) >= 2 {
		code, _ := strconv.Atoi(matches[1])
		if nonRetriableStatusCodes[code] {
			return false
		}
		if retriableStatusCodes[code] {
			return true
		}
	}

	// Heuristic: common transient-error substrings.
	transientPatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"EOF",
		"server error",
		"bad gateway",
		"service unavailable",
		"overloaded",
		"rate limit",
		"max retries",
	}
	lower := strings.ToLower(errMsg)
	for _, pattern := range transientPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Default: treat unknown errors as retriable so we at least try the next provider.
	return true
}
