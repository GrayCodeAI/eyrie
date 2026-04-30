package types

import (
	"testing"
)

func TestSessionIdAndAgentId(t *testing.T) {
	sid := AsSessionId("session-123")
	if string(sid) != "session-123" {
		t.Error("AsSessionId failed")
	}
	aid := AsAgentId("a1234567890abcdef")
	if string(aid) != "a1234567890abcdef" {
		t.Error("AsAgentId failed")
	}
}

func TestToAgentId(t *testing.T) {
	tests := []struct{ input string; valid bool }{
		{"a1234567890abcdef", true},
		{"alabel-1234567890abcdef", true},
		{"invalid", false},
		{"", false},
		{"a123", false},
	}
	for _, tt := range tests {
		result, _ := ToAgentId(tt.input)
		if tt.valid && result == nil {
			t.Errorf("ToAgentId(%q) = nil, expected valid", tt.input)
		}
		if !tt.valid && result != nil {
			t.Errorf("ToAgentId(%q) = %v, expected nil", tt.input, result)
		}
	}
}

func TestIsTextBlock(t *testing.T) {
	tb := TextBlock{Type: "text", Text: "hello"}
	if b, ok := IsTextBlock(tb); !ok || b.Text != "hello" {
		t.Error("IsTextBlock failed for valid TextBlock")
	}
	if _, ok := IsTextBlock("not a block"); ok {
		t.Error("IsTextBlock should fail for string")
	}
}

func TestIsToolUseBlock(t *testing.T) {
	tub := ToolUseBlock{Type: "tool_use", ID: "1", Name: "test", Input: nil}
	if b, ok := IsToolUseBlock(tub); !ok || b.Name != "test" {
		t.Error("IsToolUseBlock failed")
	}
}

func TestCreateMessages(t *testing.T) {
	um := CreateUserMessage("hello")
	if um.Role != "user" || um.Content != "hello" {
		t.Error("CreateUserMessage failed")
	}
	am := CreateAssistantMessage("hi")
	if am.Role != "assistant" || am.Content != "hi" {
		t.Error("CreateAssistantMessage failed")
	}
	sm := CreateSystemMessage("sys")
	if sm.Role != "system" || sm.Content != "sys" {
		t.Error("CreateSystemMessage failed")
	}
}

func TestEmptyUsage(t *testing.T) {
	u := EmptyUsage()
	if u.InputTokens != 0 || u.OutputTokens != 0 {
		t.Error("EmptyUsage should have zero tokens")
	}
	if u.Iterations == nil {
		t.Error("expected non-nil iterations slice")
	}
}

func TestAPIError(t *testing.T) {
	err := NewAPIError(429, nil, nil, "rate limited")
	if err.Error() != "rate limited" {
		t.Errorf("expected 'rate limited', got %q", err.Error())
	}
	if err.Status != 429 {
		t.Errorf("expected status 429, got %d", err.Status)
	}

	connErr := NewAPIConnectionError("timeout")
	if connErr.Error() != "timeout" {
		t.Error("connection error message wrong")
	}
}

func TestIsConnectorTextBlock(t *testing.T) {
	valid := map[string]interface{}{"type": "connector_text", "text": "hello"}
	if !IsConnectorTextBlock(valid) {
		t.Error("expected true for valid connector text block")
	}
	invalid := map[string]interface{}{"type": "text", "text": "hello"}
	if IsConnectorTextBlock(invalid) {
		t.Error("expected false for non-connector block")
	}
}
