package client

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func TestWeightedProviderSingleProvider(t *testing.T) {
	p := NewMockProvider(MockModeFixed)
	p.Response = "only one"

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: p, Weight: 1.0},
	})

	for i := 0; i < 10; i++ {
		resp, err := wp.Chat(context.Background(), []EyrieMessage{
			{Role: "user", Content: "hello"},
		}, ChatOptions{Model: "test"})
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if resp.Content != "only one" {
			t.Errorf("call %d: expected 'only one', got %q", i, resp.Content)
		}
	}

	if p.CallCount() != 10 {
		t.Errorf("expected 10 calls, got %d", p.CallCount())
	}
}

func TestWeightedProviderDistribution(t *testing.T) {
	// Use named providers so stats can distinguish them.
	primary := &namedProvider{name: "primary", mock: NewMockProvider(MockModeFixed)}
	primary.mock.Response = "from primary"

	secondary := &namedProvider{name: "secondary", mock: NewMockProvider(MockModeFixed)}
	secondary.mock.Response = "from secondary"

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: primary, Weight: 0.8},
		{Provider: secondary, Weight: 0.2},
	})

	const iterations = 1000
	counts := map[string]int{"primary": 0, "secondary": 0}

	for i := 0; i < iterations; i++ {
		resp, err := wp.Chat(context.Background(), []EyrieMessage{
			{Role: "user", Content: "hello"},
		}, ChatOptions{Model: "test"})
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if resp.Content == "from primary" {
			counts["primary"]++
		} else if resp.Content == "from secondary" {
			counts["secondary"]++
		} else {
			t.Fatalf("unexpected response: %q", resp.Content)
		}
	}

	// Check distribution is roughly 80/20 with tolerance of 8%.
	primaryRatio := float64(counts["primary"]) / float64(iterations)
	secondaryRatio := float64(counts["secondary"]) / float64(iterations)

	if math.Abs(primaryRatio-0.8) > 0.08 {
		t.Errorf("primary ratio %.3f is too far from expected 0.80", primaryRatio)
	}
	if math.Abs(secondaryRatio-0.2) > 0.08 {
		t.Errorf("secondary ratio %.3f is too far from expected 0.20", secondaryRatio)
	}
}

func TestWeightedProviderFailoverOnRetriableError(t *testing.T) {
	// Primary always returns a retriable error.
	primary := &namedProvider{name: "primary", mock: nil, err: fmt.Errorf("HTTP 503 service unavailable")}
	// Secondary succeeds.
	secondary := &namedProvider{name: "secondary", mock: NewMockProvider(MockModeFixed)}
	secondary.mock.Response = "fallback success"

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: primary, Weight: 1.0},    // will always be selected
		{Provider: secondary, Weight: 0.01}, // extremely low weight
	})

	resp, err := wp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "fallback success" {
		t.Errorf("expected 'fallback success', got %q", resp.Content)
	}
}

func TestWeightedProviderNoFailoverOnNonRetriableError(t *testing.T) {
	// Primary returns a 400 (non-retriable).
	primary := &namedProvider{name: "primary", mock: nil, err: fmt.Errorf("HTTP 400 bad request")}
	// Secondary would succeed if reached.
	secondary := &namedProvider{name: "secondary", mock: NewMockProvider(MockModeFixed)}
	secondary.mock.Response = "should not reach"

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: primary, Weight: 1.0}, // always selected
		{Provider: secondary, Weight: 0.01},
	})

	_, err := wp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err == nil {
		t.Fatal("expected error for non-retriable 400")
	}
	// Secondary should NOT have been called.
	if secondary.mock.CallCount() != 0 {
		t.Errorf("secondary was called %d times; should not be called for non-retriable error", secondary.mock.CallCount())
	}
}

func TestWeightedProviderNoFailoverOn401(t *testing.T) {
	// Primary returns a 401 (non-retriable).
	primary := &namedProvider{name: "primary", mock: nil, err: fmt.Errorf("HTTP 401 unauthorized")}
	secondary := &namedProvider{name: "secondary", mock: NewMockProvider(MockModeFixed)}
	secondary.mock.Response = "should not reach"

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: primary, Weight: 1.0},
		{Provider: secondary, Weight: 0.01},
	})

	_, err := wp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err == nil {
		t.Fatal("expected error for non-retriable 401")
	}
	if secondary.mock.CallCount() != 0 {
		t.Errorf("secondary should not be called for 401 error")
	}
}

func TestWeightedProviderAllFail(t *testing.T) {
	p1 := &namedProvider{name: "p1", mock: nil, err: fmt.Errorf("HTTP 503 service unavailable")}
	p2 := &namedProvider{name: "p2", mock: nil, err: fmt.Errorf("HTTP 502 bad gateway")}
	p3 := &namedProvider{name: "p3", mock: nil, err: fmt.Errorf("HTTP 500 internal error")}

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: p1, Weight: 0.5},
		{Provider: p2, Weight: 0.3},
		{Provider: p3, Weight: 0.2},
	})

	_, err := wp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

func TestWeightedProviderName(t *testing.T) {
	p1 := &namedProvider{name: "anthropic", mock: NewMockProvider(MockModeFixed)}
	p2 := &namedProvider{name: "openai", mock: NewMockProvider(MockModeFixed)}

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: p1, Weight: 0.8},
		{Provider: p2, Weight: 0.2},
	})

	expected := "weighted(anthropic:0.80,openai:0.20)"
	if wp.Name() != expected {
		t.Errorf("expected name %q, got %q", expected, wp.Name())
	}
}

func TestWeightedProviderPing(t *testing.T) {
	p1 := &namedProvider{name: "failing", mock: nil, err: fmt.Errorf("ping failed")}
	p2 := &namedProvider{name: "ok", mock: NewMockProvider(MockModeFixed)}

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: p1, Weight: 0.8},
		{Provider: p2, Weight: 0.2},
	})

	// Should succeed because p2 pings ok (even though p1 fails).
	if err := wp.Ping(context.Background()); err != nil {
		t.Fatalf("expected ping to succeed, got: %v", err)
	}
}

func TestWeightedProviderStreamFailover(t *testing.T) {
	primary := &namedProvider{name: "primary", mock: nil, err: fmt.Errorf("HTTP 429 rate limited")}
	secondary := &namedProvider{name: "secondary", mock: NewMockProvider(MockModeFixed)}
	secondary.mock.Response = "streamed from secondary"

	wp := NewWeightedProvider([]WeightedProviderConfig{
		{Provider: primary, Weight: 1.0},
		{Provider: secondary, Weight: 0.01},
	})

	sr, err := wp.StreamChat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	var content string
	for evt := range sr.Events {
		if evt.Type == "content" {
			content += evt.Content
		}
	}
	if content == "" {
		t.Error("expected some streamed content")
	}
}

func TestWeightedProviderPanicOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with no provider configs")
		}
	}()
	NewWeightedProvider(nil)
}

func TestWeightedProviderPanicOnZeroWeight(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with zero weight")
		}
	}()
	p := NewMockProvider(MockModeFixed)
	NewWeightedProvider([]WeightedProviderConfig{
		{Provider: p, Weight: 0},
	})
}

// namedProvider wraps a mock provider with a custom name, used to distinguish
// providers in stats and test assertions.
type namedProvider struct {
	name string
	mock *MockProvider
	err  error // if set, all calls return this error
}

func (n *namedProvider) Name() string { return n.name }

func (n *namedProvider) Ping(_ context.Context) error {
	if n.err != nil {
		return n.err
	}
	return nil
}

func (n *namedProvider) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if n.err != nil {
		return nil, n.err
	}
	return n.mock.Chat(ctx, messages, opts)
}

func (n *namedProvider) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	if n.err != nil {
		return nil, n.err
	}
	return n.mock.StreamChat(ctx, messages, opts)
}
