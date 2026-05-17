package router

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/GrayCodeAI/eyrie/client"
)

type RouteEntry struct {
	Provider client.Provider
	Weight   int
	Retry    *RetryConfig
}

type Router struct {
	entries      []RouteEntry
	fallback     []client.Provider
	totalWeight  int
	defaultRetry RetryConfig
	mu           sync.RWMutex
	stats        map[string]*atomic.Int64
}

var _ client.Provider = (*Router)(nil)

func New(entries []RouteEntry, fallback []client.Provider, defaultRetry *RetryConfig) *Router {
	total := 0
	for _, e := range entries {
		total += e.Weight
	}
	stats := make(map[string]*atomic.Int64)
	for _, e := range entries {
		stats[e.Provider.Name()] = &atomic.Int64{}
	}
	for _, p := range fallback {
		if _, ok := stats[p.Name()]; !ok {
			stats[p.Name()] = &atomic.Int64{}
		}
	}
	dr := DefaultRetryConfig()
	if defaultRetry != nil {
		dr = *defaultRetry
	}
	return &Router{
		entries:      entries,
		fallback:     fallback,
		totalWeight:  total,
		defaultRetry: dr,
		stats:        stats,
	}
}

func (r *Router) Name() string {
	names := make([]string, 0, len(r.entries))
	for _, e := range r.entries {
		names = append(names, e.Provider.Name())
	}
	return "router[" + strings.Join(names, ",") + "]"
}

func (r *Router) Ping(ctx context.Context) error {
	if len(r.entries) > 0 {
		return r.entries[0].Provider.Ping(ctx)
	}
	if len(r.fallback) > 0 {
		return r.fallback[0].Ping(ctx)
	}
	return fmt.Errorf("router: no providers configured")
}

func (r *Router) Chat(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions) (*client.EyrieResponse, error) {
	provider, retry := r.selectProvider()
	resp, err := r.chatWithRetry(ctx, provider, messages, opts, retry)
	if err == nil {
		r.recordSuccess(provider.Name())
		return resp, nil
	}
	if !IsTransient(err) {
		return nil, err
	}
	for _, fp := range r.fallback {
		if fp.Name() == provider.Name() {
			continue
		}
		resp, err = fp.Chat(ctx, messages, opts)
		if err == nil {
			r.recordSuccess(fp.Name())
			return resp, nil
		}
		if !IsTransient(err) {
			return nil, err
		}
	}
	return nil, err
}

func (r *Router) StreamChat(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions) (*client.StreamResult, error) {
	provider, _ := r.selectProvider()
	sr, err := provider.StreamChat(ctx, messages, opts)
	if err == nil {
		r.recordSuccess(provider.Name())
		return sr, nil
	}
	if !IsTransient(err) {
		return nil, err
	}
	for _, fp := range r.fallback {
		if fp.Name() == provider.Name() {
			continue
		}
		sr, err = fp.StreamChat(ctx, messages, opts)
		if err == nil {
			r.recordSuccess(fp.Name())
			return sr, nil
		}
		if !IsTransient(err) {
			return nil, err
		}
	}
	return nil, err
}

func (r *Router) Stats() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]int64, len(r.stats))
	for k, v := range r.stats {
		result[k] = v.Load()
	}
	return result
}

func (r *Router) selectProvider() (client.Provider, RetryConfig) {
	if r.totalWeight == 0 && len(r.entries) > 0 {
		e := r.entries[0]
		rc := r.defaultRetry
		if e.Retry != nil {
			rc = *e.Retry
		}
		return e.Provider, rc
	}
	n := rand.IntN(r.totalWeight)
	cumulative := 0
	for _, e := range r.entries {
		cumulative += e.Weight
		if n < cumulative {
			rc := r.defaultRetry
			if e.Retry != nil {
				rc = *e.Retry
			}
			return e.Provider, rc
		}
	}
	e := r.entries[len(r.entries)-1]
	rc := r.defaultRetry
	if e.Retry != nil {
		rc = *e.Retry
	}
	return e.Provider, rc
}

func (r *Router) chatWithRetry(ctx context.Context, p client.Provider, messages []client.EyrieMessage, opts client.ChatOptions, cfg RetryConfig) (*client.EyrieResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		resp, err := p.Chat(ctx, messages, opts)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !IsTransient(err) {
			return nil, err
		}
		if attempt < cfg.MaxRetries {
			delay := BackoffDelay(attempt, cfg)
			if cfg.OnRetry != nil {
				cfg.OnRetry(RetryEvent{Err: err, Attempt: attempt + 1, MaxRetries: cfg.MaxRetries, Delay: delay})
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-afterFunc(delay):
			}
		}
	}
	return nil, lastErr
}

func (r *Router) recordSuccess(name string) {
	if s, ok := r.stats[name]; ok {
		s.Add(1)
	}
}
