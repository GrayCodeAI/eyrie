package router

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GrayCodeAI/eyrie/client"
)

type mockProvider struct {
	name string
	err  error
}

func (m *mockProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &client.EyrieResponse{Content: "from " + m.name}, nil
}

func (m *mockProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan client.EyrieStreamEvent, 1)
	ch <- client.EyrieStreamEvent{Type: "done"}
	close(ch)
	return &client.StreamResult{Events: ch}, nil
}
func (m *mockProvider) Ping(_ context.Context) error { return m.err }
func (m *mockProvider) Name() string                 { return m.name }

func TestRouterImplementsProvider(t *testing.T) {
	var _ client.Provider = (*Router)(nil)
}

func TestWeightedSelection(t *testing.T) {
	p1 := &mockProvider{name: "p1"}
	p2 := &mockProvider{name: "p2"}
	r := New([]RouteEntry{{Provider: p1, Weight: 80}, {Provider: p2, Weight: 20}}, nil, nil)

	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		resp, _ := r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{})
		counts[resp.Content]++
	}
	if counts["from p1"] < 600 {
		t.Errorf("p1 (weight 80) got %d/1000, expected ~800", counts["from p1"])
	}
	if counts["from p2"] < 100 {
		t.Errorf("p2 (weight 20) got %d/1000, expected ~200", counts["from p2"])
	}
}

func TestFallbackOnError(t *testing.T) {
	p1 := &mockProvider{name: "p1", err: fmt.Errorf("HTTP 500 internal")}
	p2 := &mockProvider{name: "p2"}
	r := New([]RouteEntry{{Provider: p1, Weight: 100}}, []client.Provider{p2}, &RetryConfig{MaxRetries: 0})

	resp, err := r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "from p2" {
		t.Errorf("expected fallback to p2, got %s", resp.Content)
	}
}

func TestAllProvidersFail(t *testing.T) {
	p1 := &mockProvider{name: "p1", err: fmt.Errorf("HTTP 500")}
	p2 := &mockProvider{name: "p2", err: fmt.Errorf("HTTP 502")}
	r := New([]RouteEntry{{Provider: p1, Weight: 100}}, []client.Provider{p2}, &RetryConfig{MaxRetries: 0})

	_, err := r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestNonTransientNoFallback(t *testing.T) {
	p1 := &mockProvider{name: "p1", err: fmt.Errorf("HTTP 401 unauthorized")}
	p2 := &mockProvider{name: "p2"}
	r := New([]RouteEntry{{Provider: p1, Weight: 100}}, []client.Provider{p2}, &RetryConfig{MaxRetries: 0})

	_, err := r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{})
	if err == nil {
		t.Error("expected error — 401 should not fallback")
	}
}

func TestIsTransientCodes(t *testing.T) {
	cases := []struct {
		msg    string
		expect bool
	}{
		{"HTTP 429 rate limited", true},
		{"HTTP 500 internal", true},
		{"HTTP 502 bad gateway", true},
		{"connection refused", true},
		{"context deadline exceeded", true},
		{"HTTP 401 unauthorized", false},
		{"HTTP 404 not found", false},
		{"invalid input", false},
	}
	for _, tc := range cases {
		got := IsTransient(fmt.Errorf("%s", tc.msg))
		if got != tc.expect {
			t.Errorf("IsTransient(%q) = %v, want %v", tc.msg, got, tc.expect)
		}
	}
}

func TestBackoffDelay(t *testing.T) {
	cfg := RetryConfig{BaseDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second}
	d0 := BackoffDelay(0, cfg)
	d1 := BackoffDelay(1, cfg)
	d2 := BackoffDelay(2, cfg)
	if d0 > 200*time.Millisecond {
		t.Errorf("attempt 0 delay too large: %v", d0)
	}
	if d1 <= d0 {
		t.Errorf("expected d1 > d0, got %v <= %v", d1, d0)
	}
	if d2 <= d1 {
		t.Errorf("expected d2 > d1, got %v <= %v", d2, d1)
	}
}

func TestOnRetryCallback(t *testing.T) {
	calls := 0
	p := &mockProvider{name: "p", err: fmt.Errorf("HTTP 500")}
	cfg := &RetryConfig{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, OnRetry: func(e RetryEvent) { calls++ }}
	r := New([]RouteEntry{{Provider: p, Weight: 100}}, nil, cfg)

	r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{})
	if calls != 2 {
		t.Errorf("expected 2 OnRetry calls, got %d", calls)
	}
}

func TestToolFilter(t *testing.T) {
	f := NewToolFilter(map[string][]string{
		"claude-3": {"web_search"},
	})
	tools := []client.EyrieTool{
		{Name: "web_search", Description: "search"},
		{Name: "code_exec", Description: "exec"},
		{Name: "my_func", Description: "custom", Parameters: map[string]interface{}{"type": "object"}},
	}
	filtered := f.FilterTools("claude-3", tools)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools (web_search + my_func), got %d", len(filtered))
	}
	names := map[string]bool{}
	for _, t := range filtered {
		names[t.Name] = true
	}
	if !names["web_search"] || !names["my_func"] {
		t.Errorf("unexpected tools: %v", names)
	}
}

func TestStreamFallback(t *testing.T) {
	p1 := &mockProvider{name: "p1", err: fmt.Errorf("HTTP 503")}
	p2 := &mockProvider{name: "p2"}
	r := New([]RouteEntry{{Provider: p1, Weight: 100}}, []client.Provider{p2}, &RetryConfig{MaxRetries: 0})

	sr, err := r.StreamChat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for range sr.Events {
	}
}
