package codeagent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewCodeAgentRetryDefaultsLeaveFallbacksUnset(t *testing.T) {
	t.Parallel()
	cr := NewCodeAgentRetry()

	for _, errorType := range []string{"context_length", "budget_exceeded"} {
		strategy, ok := cr.strategies[errorType]
		if !ok {
			t.Fatalf("missing default strategy for %q", errorType)
		}
		if strategy.FallbackModel != "" || strategy.FallbackProvider != "" {
			t.Fatalf("%s fallback should be unset by default, got model=%q provider=%q", errorType, strategy.FallbackModel, strategy.FallbackProvider)
		}
	}
}

func TestWithFallbackConfiguresFallbackDecision(t *testing.T) {
	t.Parallel()
	cr := NewCodeAgentRetry(
		WithFallback("context_length", "claude-sonnet", "anthropic"),
	)
	ctx := context.Background()
	err := errors.New("context length exceeded")

	first := cr.DecideRetry(ctx, err, "openai", "gpt-4o")
	if first == nil || !first.ShouldRetry || first.FallbackModel != "" {
		t.Fatalf("first retry = %+v, want normal retry without fallback", first)
	}

	second := cr.DecideRetry(ctx, err, "openai", "gpt-4o")
	if second == nil || !second.ShouldRetry || second.FallbackModel != "" {
		t.Fatalf("second retry = %+v, want normal retry without fallback", second)
	}

	third := cr.DecideRetry(ctx, err, "openai", "gpt-4o")
	if third == nil {
		t.Fatal("third retry decision is nil")
	}
	if !third.ShouldRetry {
		t.Fatalf("third retry should switch to fallback, got %+v", third)
	}
	if third.FallbackModel != "claude-sonnet" || third.FallbackProvider != "anthropic" {
		t.Fatalf("third retry fallback = %q/%q, want claude-sonnet/anthropic", third.FallbackModel, third.FallbackProvider)
	}
}

func TestWithStrategyOverridesDefaultStrategy(t *testing.T) {
	t.Parallel()
	override := RetryStrategy{
		Name:       "Custom Timeout",
		MaxRetries: 1,
		BaseDelay:  25 * time.Millisecond,
		MaxDelay:   25 * time.Millisecond,
		Backoff:    1,
	}
	cr := NewCodeAgentRetry(WithStrategy("timeout", override))

	got, ok := cr.strategies["timeout"]
	if !ok {
		t.Fatal("timeout strategy missing after override")
	}
	if got.Name != override.Name || got.MaxRetries != override.MaxRetries || got.BaseDelay != override.BaseDelay || got.MaxDelay != override.MaxDelay || got.Backoff != override.Backoff {
		t.Fatalf("timeout strategy = %+v, want %+v", *got, override)
	}
}
