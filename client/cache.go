package client

// CacheControl adds Anthropic prompt caching breakpoints to messages.
// Anthropic caches content at breakpoints marked with cache_control,
// reducing cost and latency for repeated prefixes.
//
// Usage: wrap messages with AddCacheBreakpoints before passing to Chat/StreamChat.
// Breakpoints are set on: the last tool definition and the second-to-last message.

// CacheControlType is the type of cache control.
type CacheControlType string

const (
	// CacheControlEphemeral caches content for up to 5 minutes.
	CacheControlEphemeral CacheControlType = "ephemeral"
)

// CachedContent wraps a content string with cache control metadata.
type CachedContent struct {
	Type         string      `json:"type"`
	Text         string      `json:"text"`
	CacheControl interface{} `json:"cache_control,omitempty"`
}

// cacheControlParam is the Anthropic cache_control object.
type cacheControlParam struct {
	Type CacheControlType `json:"type"`
}

// AddCacheBreakpoints returns a copy of messages with Anthropic cache_control
// breakpoints applied following the recommended pattern:
//   - Breakpoint on the second-to-last message (caches conversation prefix)
//   - The last user message is left uncached (always new)
//
// Only applies to messages with role "user" or "assistant".
// No-op if fewer than 2 messages.
func AddCacheBreakpoints(messages []EyrieMessage) []anthropicCachedMessage {
	result := make([]anthropicCachedMessage, len(messages))
	for i, m := range messages {
		result[i] = anthropicCachedMessage{Role: m.Role, Content: m.Content}
	}

	// Find the second-to-last non-system message index
	nonSystem := []int{}
	for i, m := range messages {
		if m.Role != "system" {
			nonSystem = append(nonSystem, i)
		}
	}

	if len(nonSystem) >= 2 {
		// Set cache breakpoint on second-to-last message
		idx := nonSystem[len(nonSystem)-2]
		result[idx].CacheControl = &cacheControlParam{Type: CacheControlEphemeral}
	}

	return result
}

// anthropicCachedMessage is an Anthropic message with optional cache_control.
type anthropicCachedMessage struct {
	Role         string      `json:"role"`
	Content      interface{} `json:"content"` // string or []CachedContent
	CacheControl interface{} `json:"cache_control,omitempty"`
}

// buildAnthropicCachedRequest builds an Anthropic request body with cache_control.
// Use this instead of the standard request builder when prompt caching is desired.
func buildAnthropicCachedRequest(messages []EyrieMessage, model string, maxTokens int, temperature *float64, stream bool) map[string]interface{} {
	var system string
	var msgs []interface{}

	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": m.Content})
	}

	// Apply cache breakpoint to second-to-last message
	if len(msgs) >= 2 {
		idx := len(msgs) - 2
		if msg, ok := msgs[idx].(map[string]interface{}); ok {
			content := msg["content"].(string)
			msg["content"] = []map[string]interface{}{
				{
					"type": "text",
					"text": content,
					"cache_control": map[string]string{"type": "ephemeral"},
				},
			}
		}
	}

	req := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   msgs,
		"stream":     stream,
	}
	if system != "" {
		req["system"] = []map[string]interface{}{
			{
				"type": "text",
				"text": system,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		}
	}
	if temperature != nil {
		req["temperature"] = *temperature
	}
	return req
}
