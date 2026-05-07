package client

import (
	"context"
	"fmt"
	"strings"
)

// ContinuationConfig controls output continuation behavior.
type ContinuationConfig struct {
	// MaxContinuations is the maximum number of continuation calls (default 3).
	MaxContinuations int
	// MaxTotalTokens caps the total output tokens across all continuations (0 = unlimited).
	MaxTotalTokens int
}

// DefaultContinuationConfig returns sensible defaults.
func DefaultContinuationConfig() ContinuationConfig {
	return ContinuationConfig{MaxContinuations: 3, MaxTotalTokens: 32000}
}

// ChatWithContinuation calls Chat and automatically continues if stop_reason is "max_tokens".
// It appends the partial response as an assistant message and retries, accumulating content.
// Returns the fully assembled response.
func ChatWithContinuation(ctx context.Context, p Provider, messages []EyrieMessage, opts ChatOptions, cfg ContinuationConfig) (*EyrieResponse, error) {
	if cfg.MaxContinuations <= 0 {
		cfg.MaxContinuations = 3
	}

	var accumulated strings.Builder
	var finalUsage *EyrieUsage
	var finalToolCalls []ToolCall
	msgs := make([]EyrieMessage, len(messages))
	copy(msgs, messages)

	for i := 0; i <= cfg.MaxContinuations; i++ {
		resp, err := p.Chat(ctx, msgs, opts)
		if err != nil {
			return nil, fmt.Errorf("eyrie: continuation call %d failed: %w", i, err)
		}

		accumulated.WriteString(resp.Content)
		finalToolCalls = append(finalToolCalls, resp.ToolCalls...)

		// Merge usage (nil-safe)
		if resp != nil && resp.Usage != nil {
			if finalUsage == nil {
				finalUsage = &EyrieUsage{}
			}
			finalUsage.PromptTokens += resp.Usage.PromptTokens
			finalUsage.CompletionTokens += resp.Usage.CompletionTokens
			finalUsage.TotalTokens += resp.Usage.TotalTokens
		}

		// Check token cap
		if cfg.MaxTotalTokens > 0 && finalUsage != nil && finalUsage.CompletionTokens >= cfg.MaxTotalTokens {
			return &EyrieResponse{
				Content: accumulated.String(), FinishReason: "max_tokens",
				ToolCalls: finalToolCalls, Usage: finalUsage,
			}, nil
		}

		// If response ended with tool calls, don't continue — tool results needed
		if len(resp.ToolCalls) > 0 {
			return &EyrieResponse{
				Content: accumulated.String(), FinishReason: resp.FinishReason,
				ToolCalls: finalToolCalls, Usage: finalUsage, RequestID: resp.RequestID,
			}, nil
		}

		// Not max_tokens — we're done
		if resp.FinishReason != "max_tokens" {
			return &EyrieResponse{
				Content: accumulated.String(), FinishReason: resp.FinishReason,
				ToolCalls: finalToolCalls, Usage: finalUsage, RequestID: resp.RequestID,
			}, nil
		}

		// Hit max_tokens — append partial as assistant and continue
		if i < cfg.MaxContinuations {
			msgs = append(msgs, EyrieMessage{Role: "assistant", Content: accumulated.String()})
			msgs = append(msgs, EyrieMessage{Role: "user", Content: "Continue."})
		}
	}

	return &EyrieResponse{
		Content: accumulated.String(), FinishReason: "max_tokens",
		ToolCalls: finalToolCalls, Usage: finalUsage,
	}, nil
}
