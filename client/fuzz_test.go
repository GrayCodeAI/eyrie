package client

import (
	"testing"
)

func FuzzSanitizeMessages(f *testing.F) {
	// Seed with typical message patterns
	f.Add("user", "hello", "assistant", "response", "tool-123")
	f.Add("assistant", "", "", "", "")

	f.Fuzz(func(t *testing.T, role1, content1, role2, content2, toolID string) {
		messages := []EyrieMessage{
			{Role: role1, Content: content1},
			{Role: role2, Content: content2},
		}
		if toolID != "" {
			messages = append(messages, EyrieMessage{
				Role: "assistant",
				ToolUse: []ToolCall{
					{ID: toolID, Name: "test", Arguments: map[string]interface{}{"k": "v"}},
				},
			})
		}
		result := SanitizeMessages(messages)
		// Should not panic, result should have at least as many messages as input
		if len(result) < len(messages) {
			t.Errorf("sanitize reduced message count: %d -> %d", len(messages), len(result))
		}
	})
}

func FuzzMergeConsecutiveRoles(f *testing.F) {
	f.Add("user", "hello", "user", "world")
	f.Add("assistant", "a", "assistant", "b")
	f.Add("user", "a", "assistant", "b")

	f.Fuzz(func(t *testing.T, role1, content1, role2, content2 string) {
		messages := []EyrieMessage{
			{Role: role1, Content: content1},
			{Role: role2, Content: content2},
		}
		result := MergeConsecutiveRoles(messages)
		// Should not panic, result should have at least 1 message
		if len(result) == 0 && len(messages) > 0 {
			t.Error("merge produced empty result from non-empty input")
		}
		// Result should never have more messages than input
		if len(result) > len(messages) {
			t.Errorf("merge increased message count: %d -> %d", len(messages), len(result))
		}
	})
}

func FuzzBuildCacheKey(f *testing.F) {
	f.Add("system prompt", "user message", "model-name")
	f.Add("", "", "")
	f.Add("a", "b", "c")

	f.Fuzz(func(t *testing.T, system, user, model string) {
		messages := []EyrieMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		}
		opts := ChatOptions{Model: model}
		key := buildCacheKey(messages, opts)
		// Should not panic, key should be non-empty for non-empty input
		if key == "" && (system != "" || user != "") {
			t.Error("buildCacheKey returned empty for non-empty input")
		}
		// Determinism
		key2 := buildCacheKey(messages, opts)
		if key != key2 {
			t.Errorf("buildCacheKey not deterministic: %q != %q", key, key2)
		}
	})
}
