// Package opencodego holds shared constants and helpers for the OpenCode Go gateway
// (https://opencode.ai/docs/go/). Models are discovered via GET /v1/models; chat
// routing picks /v1/chat/completions or /v1/messages per model family.
package opencodego

import "strings"

// DefaultBaseURL is the OpenCode Go API root.
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

// ProtocolForModel derives the API protocol from model name patterns.
// Returns "anthropic" or "openai". This is the heuristic fallback when
// the live API doesn't include protocol metadata.
// MiniMax and Qwen3.x use Anthropic /v1/messages; everything else uses OpenAI /v1/chat/completions.
func ProtocolForModel(modelID string) string {
	id := strings.ToLower(NativeModelID(modelID))
	if id == "" {
		return "openai"
	}
	if strings.Contains(id, "minimax") || strings.HasPrefix(id, "qwen3.") {
		return "anthropic"
	}
	return "openai"
}
