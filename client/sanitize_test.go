package client

import (
	"testing"
)

func TestSanitizeMessagesEmpty(t *testing.T) {
	msgs := []EyrieMessage{}
	result := SanitizeMessages(msgs)
	if len(result) != 0 {
		t.Errorf("SanitizeMessages(empty) returned %d messages, want 0", len(result))
	}
}

func TestSanitizeMessagesNil(t *testing.T) {
	result := SanitizeMessages(nil)
	if len(result) != 0 {
		t.Errorf("SanitizeMessages(nil) returned %d messages, want 0", len(result))
	}
}

func TestSanitizeMessagesNoOrphans(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{{ID: "tc-1", Name: "search", Arguments: map[string]interface{}{"q": "test"}}}},
		{Role: "user", ToolResult: &ToolResult{ToolUseID: "tc-1", Content: "result"}},
	}
	result := SanitizeMessages(msgs)
	if len(result) != 2 {
		t.Errorf("SanitizeMessages(no orphans) returned %d messages, want 2", len(result))
	}
}

func TestSanitizeMessagesOrphanedToolUse(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "Do something"},
		{Role: "assistant", ToolUse: []ToolCall{
			{ID: "orphan-1", Name: "search", Arguments: map[string]interface{}{"q": "test"}},
		}},
	}
	result := SanitizeMessages(msgs)

	// Should inject a synthetic tool_result after the assistant message
	if len(result) != 3 {
		t.Fatalf("SanitizeMessages(orphan) returned %d messages, want 3", len(result))
	}

	injected := result[2]
	if injected.Role != "user" {
		t.Errorf("injected message role = %q, want %q", injected.Role, "user")
	}
	if injected.ToolResult == nil {
		t.Fatal("injected message has nil ToolResult")
	}
	if injected.ToolResult.ToolUseID != "orphan-1" {
		t.Errorf("injected ToolResult.ToolUseID = %q, want %q", injected.ToolResult.ToolUseID, "orphan-1")
	}
	if !injected.ToolResult.IsError {
		t.Error("injected ToolResult.IsError should be true")
	}
	if injected.ToolResult.Content != "Tool execution was interrupted" {
		t.Errorf("injected ToolResult.Content = %q, want %q", injected.ToolResult.Content, "Tool execution was interrupted")
	}
}

func TestSanitizeMessagesMultipleOrphans(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{
			{ID: "tc-a", Name: "tool_a", Arguments: map[string]interface{}{}},
			{ID: "tc-b", Name: "tool_b", Arguments: map[string]interface{}{}},
		}},
	}
	result := SanitizeMessages(msgs)

	// 1 original + 2 injected results
	if len(result) != 3 {
		t.Fatalf("SanitizeMessages(multiple orphans) returned %d messages, want 3", len(result))
	}
	if result[1].ToolResult.ToolUseID != "tc-a" {
		t.Errorf("first injected ID = %q, want %q", result[1].ToolResult.ToolUseID, "tc-a")
	}
	if result[2].ToolResult.ToolUseID != "tc-b" {
		t.Errorf("second injected ID = %q, want %q", result[2].ToolResult.ToolUseID, "tc-b")
	}
}

func TestSanitizeMessagesMixedOrphanAndMatched(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{
			{ID: "matched", Name: "tool1", Arguments: map[string]interface{}{}},
			{ID: "orphan", Name: "tool2", Arguments: map[string]interface{}{}},
		}},
		{Role: "user", ToolResult: &ToolResult{ToolUseID: "matched", Content: "ok"}},
	}
	result := SanitizeMessages(msgs)

	// 2 original + 1 injected for "orphan" = 3 total
	if len(result) != 3 {
		t.Fatalf("SanitizeMessages(mixed) returned %d messages, want 3", len(result))
	}
	// result[0] = assistant, result[1] = injected for "orphan", result[2] = original user with "matched"
	injected := result[1]
	if injected.ToolResult.ToolUseID != "orphan" {
		t.Errorf("injected ID = %q, want %q", injected.ToolResult.ToolUseID, "orphan")
	}
	matched := result[2]
	if matched.ToolResult.ToolUseID != "matched" {
		t.Errorf("matched ID = %q, want %q", matched.ToolResult.ToolUseID, "matched")
	}
}

func TestSanitizeMessagesPreservesOrder(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
	}
	result := SanitizeMessages(msgs)
	if len(result) != 3 {
		t.Fatalf("SanitizeMessages preserves: got %d, want 3", len(result))
	}
	for i, want := range []string{"first", "second", "third"} {
		if result[i].Content != want {
			t.Errorf("result[%d].Content = %q, want %q", i, result[i].Content, want)
		}
	}
}

func TestSanitizeMessagesNoToolUse(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}
	result := SanitizeMessages(msgs)
	if len(result) != 2 {
		t.Errorf("SanitizeMessages(no tools) returned %d messages, want 2", len(result))
	}
}

func TestSanitizeMessagesEmptyToolUse(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{}},
	}
	result := SanitizeMessages(msgs)
	if len(result) != 1 {
		t.Errorf("SanitizeMessages(empty tool_use) returned %d messages, want 1", len(result))
	}
}

func TestSanitizeMessagesOrphanWithEmptyID(t *testing.T) {
	// Tool call with empty ID should not be injected (tc.ID != "" guard)
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{
			{Name: "tool", Arguments: map[string]interface{}{}},
		}},
	}
	result := SanitizeMessages(msgs)
	if len(result) != 1 {
		t.Errorf("SanitizeMessages(empty ID) returned %d messages, want 1 (no injection for empty ID)", len(result))
	}
}

func TestSanitizeMessagesDeduplicatesInjections(t *testing.T) {
	// If the same ID appears in two assistant messages, only inject once
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{{ID: "dup-1", Name: "t1", Arguments: map[string]interface{}{}}}},
		{Role: "assistant", ToolUse: []ToolCall{{ID: "dup-1", Name: "t1", Arguments: map[string]interface{}{}}}},
	}
	result := SanitizeMessages(msgs)
	// 2 originals + 1 injection (first time), second time already in resultIDs
	if len(result) != 3 {
		t.Errorf("SanitizeMessages(dedup) returned %d messages, want 3", len(result))
	}
}
