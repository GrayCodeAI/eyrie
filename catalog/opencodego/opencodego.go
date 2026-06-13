// Package opencodego holds shared constants and helpers for the OpenCode Go gateway
// (https://opencode.ai/docs/go/). Eyrie uses the OpenAI-compatible surface only:
// GET /v1/models and POST /v1/chat/completions on a single base URL.
package opencodego

import "strings"

// DefaultBaseURL is the OpenCode Go API root (OpenAI-compatible list + chat paths).
const DefaultBaseURL = "https://opencode.ai/zen/go/v1"

// NativeModelID strips OpenCode config prefixes (opencode-go/kimi-k2.6 → kimi-k2.6).
func NativeModelID(id string) string {
	id = strings.TrimSpace(id)
	lower := strings.ToLower(id)
	for _, prefix := range []string{"opencode-go/", "opencodego/"} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(id[len(prefix):])
		}
	}
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return strings.TrimSpace(id[i+1:])
	}
	return id
}

// ChatCompletionsSupported reports whether a model is served on /v1/chat/completions.
// MiniMax and Qwen3.x models on OpenCode Go require /v1/messages instead; eyrie
// follows the OpenAI-compatible path only and excludes them from live discovery.
func ChatCompletionsSupported(modelID string) bool {
	id := strings.ToLower(NativeModelID(modelID))
	if id == "" {
		return false
	}
	if strings.Contains(id, "minimax") {
		return false
	}
	if strings.HasPrefix(id, "qwen3.") {
		return false
	}
	return true
}
