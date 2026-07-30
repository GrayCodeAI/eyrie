package client

import (
	"errors"
	"strings"

	"github.com/GrayCodeAI/eyrie/client/core"
)

// ApplyProviderChatDefaults applies provider policy that host applications
// should not need to encode themselves.
func ApplyProviderChatDefaults(provider string, opts ChatOptions) ChatOptions {
	if strings.EqualFold(strings.TrimSpace(provider), "anthropic") {
		opts.EnableCaching = true
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "zai_payg", "zai_coding", "longcat", "agnes":
		// Keep GLMThinkingEnabled — these providers use thinking / chat_template_kwargs.
	default:
		opts.GLMThinkingEnabled = nil
	}
	return opts
}

// IsContextOverflow reports whether an error means the request exceeded the
// selected model's context window.
func IsContextOverflow(err error) bool {
	if err == nil {
		return false
	}
	var providerErr *core.EyrieError
	if errors.As(err, &providerErr) {
		message := strings.ToLower(providerErr.Message)
		if containsContextOverflowSignal(message) {
			return true
		}
	}
	return containsContextOverflowSignal(strings.ToLower(err.Error()))
}

func containsContextOverflowSignal(message string) bool {
	for _, signal := range []string{
		"input exceeds the model's context window",
		"context_length_exceeded",
		"context length",
		"maximum context",
		"too many tokens",
		"prompt is too long",
		"reduce the length",
	} {
		if strings.Contains(message, signal) {
			return true
		}
	}
	return false
}
