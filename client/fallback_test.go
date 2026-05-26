package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GrayCodeAI/eyrie/types"
)

func TestFallbackProviderSuccess(t *testing.T) {
	primary := NewMockProvider(MockModeFixed)
	primary.Response = "from primary"

	fp := NewFallbackProvider(primary)
	resp, err := fp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "from primary" {
		t.Errorf("expected 'from primary', got %q", resp.Content)
	}
}

func TestFallbackProviderFallsBack(t *testing.T) {
	// Primary always errors (retriable).
	primary := NewMockProvider(MockModeError)
	// Secondary succeeds.
	secondary := NewMockProvider(MockModeFixed)
	secondary.Response = "from secondary"

	fp := NewFallbackProvider(primary, secondary)
	resp, err := fp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "from secondary" {
		t.Errorf("expected 'from secondary', got %q", resp.Content)
	}
	if primary.CallCount() != 1 {
		t.Errorf("expected primary to be called once, got %d", primary.CallCount())
	}
	if secondary.CallCount() != 1 {
		t.Errorf("expected secondary to be called once, got %d", secondary.CallCount())
	}
}

func TestFallbackProviderAllFail(t *testing.T) {
	p1 := NewMockProvider(MockModeError)
	p2 := NewMockProvider(MockModeError)
	p3 := NewMockProvider(MockModeError)

	fp := NewFallbackProvider(p1, p2, p3)
	_, err := fp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	if p1.CallCount() != 1 || p2.CallCount() != 1 || p3.CallCount() != 1 {
		t.Errorf("expected each provider called once: p1=%d p2=%d p3=%d",
			p1.CallCount(), p2.CallCount(), p3.CallCount())
	}
}

func TestFallbackProviderStats(t *testing.T) {
	primary := NewMockProvider(MockModeError)
	secondary := NewMockProvider(MockModeFixed)
	secondary.Response = "ok"

	fp := NewFallbackProvider(primary, secondary)

	for i := 0; i < 5; i++ {
		_, err := fp.Chat(context.Background(), []EyrieMessage{
			{Role: "user", Content: "hello"},
		}, ChatOptions{Model: "test"})
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}
	}

	stats := fp.Stats()
	if stats["mock"] != 5 {
		t.Errorf("expected mock to have 5 successes, got %d", stats["mock"])
	}
}

func TestFallbackProviderRespectsContextCancellation(t *testing.T) {
	// A slow primary provider.
	primary := NewMockProvider(MockModeFixed)
	primary.Response = "slow"
	primary.Delay = 5 * time.Second

	secondary := NewMockProvider(MockModeFixed)
	secondary.Response = "fast"

	fp := NewFallbackProvider(primary, secondary)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := fp.Chat(ctx, []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestFallbackProviderStreamFallback(t *testing.T) {
	primary := NewMockProvider(MockModeError)
	secondary := NewMockProvider(MockModeFixed)
	secondary.Response = "streamed from secondary"

	fp := NewFallbackProvider(primary, secondary)

	sr, err := fp.StreamChat(context.Background(), []EyrieMessage{
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

func TestFallbackProviderPing(t *testing.T) {
	p1 := NewMockProvider(MockModeFixed)
	p2 := NewMockProvider(MockModeFixed)

	fp := NewFallbackProvider(p1, p2)
	if err := fp.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestFallbackProviderName(t *testing.T) {
	p1 := NewMockProvider(MockModeFixed)
	p2 := NewMockProvider(MockModeFixed)

	fp := NewFallbackProvider(p1, p2)
	name := fp.Name()
	if name != "fallback(mock->mock)" {
		t.Errorf("unexpected name: %s", name)
	}
}

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retriable bool
	}{
		// nil
		{"nil error", nil, false},

		// Retriable HTTP status codes: 408, 429, 500, 502, 503, 504, 529
		{"HTTP 408", fmt.Errorf("HTTP 408 from api.openai.com"), true},
		{"HTTP 429", fmt.Errorf("HTTP 429 from api.openai.com"), true},
		{"HTTP 500", fmt.Errorf("HTTP 500 from api.openai.com"), true},
		{"HTTP 502", fmt.Errorf("HTTP 502 from api.openai.com"), true},
		{"HTTP 503", fmt.Errorf("HTTP 503 from api.openai.com"), true},
		{"HTTP 504", fmt.Errorf("HTTP 504 from api.openai.com"), true},
		{"HTTP 529", fmt.Errorf("HTTP 529 from api.openai.com"), true},

		// Non-retriable HTTP status codes: 400, 401, 403, 404, 422
		{"HTTP 400", fmt.Errorf("HTTP 400 from api.openai.com"), false},
		{"HTTP 401", fmt.Errorf("HTTP 401 from api.openai.com"), false},
		{"HTTP 403", fmt.Errorf("HTTP 403 from api.openai.com"), false},
		{"HTTP 404", fmt.Errorf("HTTP 404 from api.openai.com"), false},
		{"HTTP 422", fmt.Errorf("HTTP 422 from api.openai.com"), false},

		// Context errors
		{"context deadline", context.DeadlineExceeded, true},
		{"context cancelled", context.Canceled, false},

		// TransientError type — delegated to IsTransient, returns true
		{"TransientError 500", &types.TransientError{StatusCode: 500, Message: "oops"}, true},
		{"TransientError 429", &types.TransientError{StatusCode: 429, Message: "rate limit"}, true},
		{"wrapped TransientError", fmt.Errorf("outer: %w", &types.TransientError{StatusCode: 503, Message: "down"}), true},

		// Message-based pattern matching (delegated to IsTransient)
		{"timeout message", fmt.Errorf("request timeout"), true},
		{"timed out", fmt.Errorf("connection timed out"), true},
		{"deadline exceeded message", fmt.Errorf("deadline exceeded"), true},
		{"connection refused", fmt.Errorf("connection refused"), true},
		{"connection reset", fmt.Errorf("connection reset by peer"), true},
		{"EOF", fmt.Errorf("unexpected EOF"), true},
		{"broken pipe", fmt.Errorf("broken pipe"), true},
		{"temporarily unavailable", fmt.Errorf("temporarily unavailable"), true},
		{"overloaded", fmt.Errorf("server overloaded"), true},
		{"try again", fmt.Errorf("please try again later"), true},
		{"rate limit", fmt.Errorf("eyrie: rate limit exceeded"), true},
		{"rate_limit", fmt.Errorf("rate_limit hit"), true},

		// APIConnectionTimeoutError (recognized by IsTransient)
		{"APIConnectionTimeoutError", types.NewAPIConnectionTimeoutError("timed out"), true},

		// KEY DIVERGENCE: unknown errors are retriable in isRetriableError
		// (optimistic) but NOT retriable in IsTransient (conservative).
		{"unknown error — optimistic", fmt.Errorf("eyrie: mock error"), true},
		{"unknown error — weird", fmt.Errorf("something weird happened"), true},
		{"unknown error — crash", fmt.Errorf("provider crashed"), true},
		{"unknown error — assertion", fmt.Errorf("internal assertion failed"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetriableError(tt.err)
			if got != tt.retriable {
				t.Errorf("isRetriableError(%v) = %v, want %v", tt.err, got, tt.retriable)
			}
		})
	}
}

// TestIsRetriableErrorVsIsTransientDivergence documents the intentional
// behavioral difference: isRetriableError is optimistic (unknown errors retriable),
// while types.IsTransient is conservative (unknown errors NOT retriable).
func TestIsRetriableErrorVsIsTransientDivergence(t *testing.T) {
	unknownErrors := []error{
		fmt.Errorf("something weird happened"),
		fmt.Errorf("provider crashed"),
		fmt.Errorf("internal assertion failed"),
		fmt.Errorf("unexpected response format"),
	}

	for _, err := range unknownErrors {
		// IsTransient: conservative — unknown errors are NOT retriable.
		if types.IsTransient(err) {
			t.Errorf("types.IsTransient should be conservative for %q, got true", err)
		}
		// isRetriableError: optimistic — unknown errors ARE retriable.
		if !isRetriableError(err) {
			t.Errorf("isRetriableError should be optimistic for %q, got false", err)
		}
	}

	// Known retriable errors should be retriable in BOTH functions.
	retriableErrors := []error{
		context.DeadlineExceeded,
		fmt.Errorf("HTTP 503 service unavailable"),
		fmt.Errorf("rate limit exceeded"),
		&types.TransientError{StatusCode: 500, Message: "oops"},
	}
	for _, err := range retriableErrors {
		if !types.IsTransient(err) {
			t.Errorf("types.IsTransient should be true for %q", err)
		}
		if !isRetriableError(err) {
			t.Errorf("isRetriableError should be true for %q", err)
		}
	}

	// context.Canceled is NOT retriable in either function.
	if types.IsTransient(context.Canceled) {
		t.Error("types.IsTransient(context.Canceled) should be false")
	}
	if isRetriableError(context.Canceled) {
		t.Error("isRetriableError(context.Canceled) should be false")
	}
}

func TestFallbackProviderPanicOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with no providers")
		}
	}()
	NewFallbackProvider()
}

func TestFallbackProviderNonRetriableDoesNotFallback(t *testing.T) {
	// Create a custom mock that returns a 401-like error.
	primary := &errorProvider{err: fmt.Errorf("HTTP 401 unauthorized")}
	secondary := NewMockProvider(MockModeFixed)
	secondary.Response = "should not reach"

	fp := NewFallbackProvider(primary, secondary)
	_, err := fp.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hello"},
	}, ChatOptions{Model: "test"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	// Secondary should NOT have been called.
	if secondary.CallCount() != 0 {
		t.Errorf("secondary was called %d times; should not be called for non-retriable error", secondary.CallCount())
	}
}

// errorProvider is a minimal Provider that always returns a specific error.
type errorProvider struct {
	err error
}

func (e *errorProvider) Chat(_ context.Context, _ []EyrieMessage, _ ChatOptions) (*EyrieResponse, error) {
	return nil, e.err
}

func (e *errorProvider) StreamChat(_ context.Context, _ []EyrieMessage, _ ChatOptions) (*StreamResult, error) {
	return nil, e.err
}
func (e *errorProvider) Ping(_ context.Context) error { return e.err }
func (e *errorProvider) Name() string                 { return "error-provider" }
