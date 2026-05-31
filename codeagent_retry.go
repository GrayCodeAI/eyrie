// Package eyrie is a universal LLM provider runtime. It routes requests to
// multiple AI providers (Anthropic, OpenAI, Gemini, Azure, Bedrock, Vertex,
// and OpenAI-compatible endpoints) with reliability features: circuit breaker,
// request coalescing, output guardrails (PII / secrets / injection),
// structured-output validation with retry, request-scoped lifecycle callbacks,
// and code-agent–specific retry strategies.
//
// This file (codeagent_retry.go) implements intelligent retry and fallback
// policies tailored to code agent workloads, distinguishing rate-limit errors,
// context window overflows, and tool-call failures from generic errors.
package eyrie

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// CodeAgentRetry provides intelligent retry and fallback strategies
// specifically for code agent workloads. Unlike generic retry, this
// understands code-specific failures and adapts accordingly.
type CodeAgentRetry struct {
	mu         sync.Mutex
	strategies map[string]*RetryStrategy
	history    []RetryRecord
}

// RetryStrategy defines retry behavior for a specific failure type.
type RetryStrategy struct {
	Name             string
	MaxRetries       int
	BaseDelay        time.Duration
	MaxDelay         time.Duration
	Backoff          float64
	FallbackModel    string // switch to this model on failure
	FallbackProvider string // switch to this provider on failure
}

// RetryRecord captures a retry attempt for learning.
type RetryRecord struct {
	Timestamp    time.Time
	Provider     string
	Model        string
	ErrorType    string
	ErrorMessage string
	RetryCount   int
	Recovered    bool
	FallbackUsed bool
}

// NewCodeAgentRetry creates a retry system with code-agent-specific strategies.
func NewCodeAgentRetry() *CodeAgentRetry {
	cr := &CodeAgentRetry{
		strategies: make(map[string]*RetryStrategy),
		history:    make([]RetryRecord, 0, 1000),
	}
	cr.registerDefaults()
	return cr
}

// registerDefaults sets up default retry strategies for common code agent failures.
func (cr *CodeAgentRetry) registerDefaults() {
	// Rate limiting - wait and retry
	cr.strategies["rate_limit"] = &RetryStrategy{
		Name:       "Rate Limit",
		MaxRetries: 5,
		BaseDelay:  5 * time.Second,
		MaxDelay:   60 * time.Second,
		Backoff:    2.0,
	}

	// Context length exceeded - switch to model with larger context
	cr.strategies["context_length"] = &RetryStrategy{
		Name:             "Context Length",
		MaxRetries:       2,
		BaseDelay:        1 * time.Second,
		FallbackModel:    "claude-3-5-sonnet", // larger context
		FallbackProvider: "anthropic",
	}

	// Tool execution failure - retry with different approach
	cr.strategies["tool_failure"] = &RetryStrategy{
		Name:       "Tool Failure",
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		Backoff:    1.5,
	}

	// Token budget exceeded - switch to cheaper model
	cr.strategies["budget_exceeded"] = &RetryStrategy{
		Name:             "Budget Exceeded",
		MaxRetries:       1,
		FallbackModel:    "gpt-4o-mini", // cheaper
		FallbackProvider: "openai",
	}

	// Server error - retry with backoff
	cr.strategies["server_error"] = &RetryStrategy{
		Name:       "Server Error",
		MaxRetries: 3,
		BaseDelay:  3 * time.Second,
		MaxDelay:   30 * time.Second,
		Backoff:    2.0,
	}

	// Timeout - retry with longer timeout
	cr.strategies["timeout"] = &RetryStrategy{
		Name:       "Timeout",
		MaxRetries: 2,
		BaseDelay:  5 * time.Second,
		Backoff:    2.0,
	}
}

// DecideRetry determines how to handle a failure.
func (cr *CodeAgentRetry) DecideRetry(ctx context.Context, err error, provider, model string) *RetryDecision {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	errorType := classifyError(err)

	strategy, exists := cr.strategies[errorType]
	if !exists {
		strategy = cr.strategies["server_error"] // default
	}

	// Check if we've exceeded max retries for this error type
	recentRetries := cr.countRecentRetries(errorType, provider, model)
	if recentRetries >= strategy.MaxRetries {
		// Try fallback if available
		if strategy.FallbackModel != "" {
			return &RetryDecision{
				ShouldRetry:      true,
				Delay:            0,
				Reason:           fmt.Sprintf("max retries exceeded for %s, switching to fallback", errorType),
				FallbackModel:    strategy.FallbackModel,
				FallbackProvider: strategy.FallbackProvider,
			}
		}
		return &RetryDecision{
			ShouldRetry: false,
			Reason:      fmt.Sprintf("max retries exceeded for %s", errorType),
		}
	}

	// Calculate delay with exponential backoff
	delay := strategy.BaseDelay
	for i := 0; i < recentRetries; i++ {
		delay = time.Duration(float64(delay) * strategy.Backoff)
	}
	if delay > strategy.MaxDelay {
		delay = strategy.MaxDelay
	}

	// Record the retry attempt
	cr.recordRetry(provider, model, errorType, err.Error())

	return &RetryDecision{
		ShouldRetry: true,
		Delay:       delay,
		Reason:      fmt.Sprintf("retrying %s (attempt %d/%d)", errorType, recentRetries+1, strategy.MaxRetries),
	}
}

// RetryDecision describes what to do after a failure.
type RetryDecision struct {
	ShouldRetry      bool
	Delay            time.Duration
	Reason           string
	FallbackModel    string
	FallbackProvider string
}

func (cr *CodeAgentRetry) recordRetry(provider, model, errorType, errorMsg string) {
	cr.history = append(cr.history, RetryRecord{
		Timestamp:    time.Now(),
		Provider:     provider,
		Model:        model,
		ErrorType:    errorType,
		ErrorMessage: errorMsg,
		Recovered:    false,
	})

	// Keep history bounded
	if len(cr.history) > 1000 {
		cr.history = cr.history[500:]
	}
}

func (cr *CodeAgentRetry) countRecentRetries(errorType, provider, model string) int {
	count := 0
	cutoff := time.Now().Add(-5 * time.Minute)

	for _, r := range cr.history {
		if r.Timestamp.After(cutoff) &&
			r.ErrorType == errorType &&
			r.Provider == provider &&
			r.Model == model {
			count++
		}
	}

	return count
}

// classifyError determines the error type from an error message.
func classifyError(err error) string {
	msg := err.Error()
	lower := strings.ToLower(msg)

	if strings.Contains(lower, "rate limit") || strings.Contains(lower, "429") {
		return "rate_limit"
	}
	if strings.Contains(lower, "context length") || strings.Contains(lower, "too long") {
		return "context_length"
	}
	if strings.Contains(lower, "budget") || strings.Contains(lower, "cost") {
		return "budget_exceeded"
	}
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline") {
		return "timeout"
	}
	if strings.Contains(lower, "500") || strings.Contains(lower, "503") || strings.Contains(lower, "server") {
		return "server_error"
	}
	if strings.Contains(lower, "tool") || strings.Contains(lower, "function") {
		return "tool_failure"
	}

	return "unknown"
}

// Stats returns retry statistics.
func (cr *CodeAgentRetry) Stats() map[string]interface{} {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	total := len(cr.history)
	recovered := 0
	fallbacks := 0

	for _, r := range cr.history {
		if r.Recovered {
			recovered++
		}
		if r.FallbackUsed {
			fallbacks++
		}
	}

	return map[string]interface{}{
		"total_retries":  total,
		"recovered":      recovered,
		"fallbacks_used": fallbacks,
		"recovery_rate":  float64(recovered) / float64(imax(1, total)),
	}
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
