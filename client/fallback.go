package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/GrayCodeAI/eyrie/types"
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
// NewFallbackProvider creates a fallback provider chain. Panics if no providers are given (programmer error).
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
		logger:    slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
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

// nonRetriableStatusCodes are HTTP status codes that indicate a client-side
// error -- falling back won't help because the request itself is bad.
var nonRetriableStatusCodes = map[int]bool{
	400: true, // bad request
	401: true, // unauthorized
	403: true, // forbidden
	404: true, // not found (wrong model name, etc.)
	422: true, // unprocessable entity
}

// isRetriableError determines whether a fallback to the next provider should
// be attempted. It delegates to types.IsTransient for known error patterns
// but diverges on unknown errors: where IsTransient is conservative (returns
// false for unrecognized errors), isRetriableError is optimistic (returns true).
//
// Rationale: in a fallback chain, trying the next provider is cheap and may
// succeed even if the current provider failed with an unexpected error type.
// In contrast, retry middleware (which uses IsTransient) should be conservative
// to avoid wasting requests on errors that won't resolve with a retry.
//
// *EyrieError (returned by formatAPIError and friends) is preferred over
// the string-based heuristic: it carries the structured status code so the
// classification is exact rather than regex-parsed.
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	// Structured path: trust *EyrieError's IsRetriable.
	var eyrieErr *EyrieError
	if errors.As(err, &eyrieErr) {
		return eyrieErr.IsRetriable()
	}
	// Heuristic path for legacy errors that predate the EyrieError migration.
	if types.IsTransient(err) {
		return true
	}
	// If the error contains a known non-retriable HTTP status code, give up
	// immediately — the request itself is bad, not the provider.
	if code, ok := types.ExtractHTTPStatus(err); ok {
		if nonRetriableStatusCodes[code] {
			return false
		}
	}
	// Unknown error types: treat as retriable so we at least try the next provider.
	return true
}
