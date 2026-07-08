package client

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	randv2 "math/rand/v2"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// WeightedProviderConfig associates a Provider with a selection weight.
type WeightedProviderConfig struct {
	Provider Provider
	Weight   float64 // relative weight (e.g., 0.8 for 80%)
}

// WeightedProvider selects a provider based on configured weights,
// with automatic failover to remaining providers on retriable errors.
//
// WeightedProvider is safe for concurrent use.
type WeightedProvider struct {
	configs []normalizedConfig // sorted by descending weight
	mu      sync.Mutex
	rng     *randv2.Rand

	// stats tracks how many times each provider served a request.
	stats map[string]*atomic.Int64
}

// normalizedConfig holds a provider with its normalized (0-1) weight.
type normalizedConfig struct {
	provider Provider
	weight   float64
}

// Compile-time check that WeightedProvider implements Provider.
var _ Provider = (*WeightedProvider)(nil)

// NewWeightedProvider creates a WeightedProvider that selects providers
// based on the configured weights. At least one provider must be supplied
// and every weight must be positive; an error is returned otherwise.
// Weights are normalized to sum to 1.0.
func NewWeightedProvider(configs []WeightedProviderConfig) (*WeightedProvider, error) {
	if len(configs) == 0 {
		return nil, errors.New("eyrie: WeightedProvider requires at least one provider config")
	}

	// Compute total weight for normalization.
	var total float64
	for _, c := range configs {
		if c.Weight <= 0 {
			return nil, errors.New("eyrie: WeightedProvider weights must be positive")
		}
		total += c.Weight
	}

	normalized := make([]normalizedConfig, len(configs))
	for i, c := range configs {
		normalized[i] = normalizedConfig{
			provider: c.Provider,
			weight:   c.Weight / total,
		}
	}

	// Sort by descending weight for failover ordering.
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].weight > normalized[j].weight
	})

	stats := make(map[string]*atomic.Int64, len(configs))
	for _, c := range normalized {
		if _, ok := stats[c.provider.Name()]; !ok {
			stats[c.provider.Name()] = &atomic.Int64{}
		}
	}

	// Seed from crypto/rand to avoid deterministic sequences.
	var seed [16]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return nil, fmt.Errorf("eyrie: failed to read crypto entropy: %w", err)
	}
	s1 := binary.BigEndian.Uint64(seed[:8])
	s2 := binary.BigEndian.Uint64(seed[8:])

	return &WeightedProvider{
		configs: normalized,
		rng:     randv2.New(randv2.NewPCG(s1, s2)), // #nosec G404 -- non-cryptographic weighted provider selection, not a security decision
		stats:   stats,
	}, nil
}

// Name returns a composite name showing providers and their weights.
func (wp *WeightedProvider) Name() string {
	parts := make([]string, len(wp.configs))
	for i, c := range wp.configs {
		parts[i] = fmt.Sprintf("%s:%.2f", c.provider.Name(), c.weight)
	}
	return "weighted(" + strings.Join(parts, ",") + ")"
}

// Ping tries to ping each provider, returning nil on the first success.
func (wp *WeightedProvider) Ping(ctx context.Context) error {
	var lastErr error
	for _, c := range wp.configs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.provider.Ping(ctx); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("eyrie: all weighted providers failed ping: %w", lastErr)
}

// Chat sends a non-streaming chat request using weighted random selection
// with failover on retriable errors.
func (wp *WeightedProvider) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	selected := wp.selectProvider()
	var lastErr error

	// Try the selected provider first.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resp, err := selected.Chat(ctx, messages, opts)
	if err == nil {
		wp.recordSuccess(selected.Name())
		return resp, nil
	}

	// If non-retriable, return immediately.
	if !isRetriableError(err) {
		return nil, err
	}
	lastErr = err

	// Failover: try remaining providers in weight-descending order.
	for _, c := range wp.configs {
		if c.provider == selected {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := c.provider.Chat(ctx, messages, opts)
		if err == nil {
			wp.recordSuccess(c.provider.Name())
			return resp, nil
		}
		if !isRetriableError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("eyrie: all weighted providers failed: %w", lastErr)
}

// StreamChat sends a streaming chat request using weighted random selection
// with failover on retriable errors.
func (wp *WeightedProvider) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	selected := wp.selectProvider()
	var lastErr error

	// Try the selected provider first.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sr, err := selected.StreamChat(ctx, messages, opts)
	if err == nil {
		wp.recordSuccess(selected.Name())
		return sr, nil
	}

	// If non-retriable, return immediately.
	if !isRetriableError(err) {
		return nil, err
	}
	lastErr = err

	// Failover: try remaining providers in weight-descending order.
	for _, c := range wp.configs {
		if c.provider == selected {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sr, err := c.provider.StreamChat(ctx, messages, opts)
		if err == nil {
			wp.recordSuccess(c.provider.Name())
			return sr, nil
		}
		if !isRetriableError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("eyrie: all weighted providers failed streaming: %w", lastErr)
}

// Stats returns a snapshot of how many times each provider served a request.
func (wp *WeightedProvider) Stats() map[string]int64 {
	result := make(map[string]int64, len(wp.stats))
	for name, counter := range wp.stats {
		result[name] = counter.Load()
	}
	return result
}

// selectProvider picks a provider based on weighted random selection.
func (wp *WeightedProvider) selectProvider() Provider {
	wp.mu.Lock()
	r := wp.rng.Float64()
	wp.mu.Unlock()

	var cumulative float64
	for _, c := range wp.configs {
		cumulative += c.weight
		if r < cumulative {
			return c.provider
		}
	}
	// Fallback to last provider (handles floating point edge case).
	return wp.configs[len(wp.configs)-1].provider
}

func (wp *WeightedProvider) recordSuccess(name string) {
	if counter, ok := wp.stats[name]; ok {
		counter.Add(1)
	}
}
