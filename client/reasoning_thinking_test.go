//nolint:errcheck
package client

import (
	"encoding/json"
	"testing"
)

// --- OpenAI reasoning_effort wiring ---

func TestBuildRequestBase_ReasoningEffort(t *testing.T) {
	tests := []struct {
		name   string
		compat *OpenAICompatConfig
		effort string
		want   string // expected reasoning_effort on the request ("" = unset)
	}{
		{
			name:   "supported and set",
			compat: &OpenAICompatConfig{SupportsReasoningEffort: true, MaxTokensField: "max_completion_tokens"},
			effort: "high",
			want:   "high",
		},
		{
			name:   "supported but empty",
			compat: &OpenAICompatConfig{SupportsReasoningEffort: true},
			effort: "",
			want:   "",
		},
		{
			name:   "not supported but set",
			compat: &OpenAICompatConfig{SupportsReasoningEffort: false},
			effort: "high",
			want:   "",
		},
		{
			name:   "nil compat",
			compat: nil,
			effort: "high",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ChatOptions{Model: "gpt-4o", ReasoningEffort: tt.effort}
			req := buildRequestBase(basicMessages(), opts, false, tt.compat)
			if req.ReasoningEffort != tt.want {
				t.Errorf("ReasoningEffort = %q, want %q", req.ReasoningEffort, tt.want)
			}

			// Confirm JSON serialization honors omitempty.
			b, _ := json.Marshal(req)
			var m map[string]interface{}
			_ = json.Unmarshal(b, &m)
			_, present := m["reasoning_effort"]
			if tt.want == "" && present {
				t.Errorf("reasoning_effort should be omitted, got %v", m["reasoning_effort"])
			}
			if tt.want != "" && m["reasoning_effort"] != tt.want {
				t.Errorf("serialized reasoning_effort = %v, want %q", m["reasoning_effort"], tt.want)
			}
		})
	}
}

// --- GLM/Z.ai thinking toggle wiring ---

func boolPtr(b bool) *bool { return &b }

func TestBuildRequestBase_GLMThinking(t *testing.T) {
	tests := []struct {
		name     string
		compat   *OpenAICompatConfig
		enabled  *bool
		wantType string // expected thinking.type ("" = thinking omitted)
		wantOmit bool
	}{
		{
			name:     "zai enabled",
			compat:   &OpenAICompatConfig{ThinkingFormat: "zai"},
			enabled:  boolPtr(true),
			wantType: "enabled",
		},
		{
			name:     "zai disabled",
			compat:   &OpenAICompatConfig{ThinkingFormat: "zai"},
			enabled:  boolPtr(false),
			wantType: "disabled",
		},
		{
			name:     "zai nil toggle omits thinking",
			compat:   &OpenAICompatConfig{ThinkingFormat: "zai"},
			enabled:  nil,
			wantOmit: true,
		},
		{
			name:     "non-zai compat ignores toggle",
			compat:   &OpenAICompatConfig{ThinkingFormat: "openai"},
			enabled:  boolPtr(true),
			wantOmit: true,
		},
		{
			name:     "nil compat ignores toggle",
			compat:   nil,
			enabled:  boolPtr(true),
			wantOmit: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ChatOptions{Model: "glm-4.6", GLMThinkingEnabled: tt.enabled}
			req := buildRequestBase(basicMessages(), opts, false, tt.compat)

			b, _ := json.Marshal(req)
			var m map[string]interface{}
			_ = json.Unmarshal(b, &m)
			th, present := m["thinking"]
			if tt.wantOmit {
				if present {
					t.Errorf("thinking should be omitted, got %v", th)
				}
				return
			}
			obj, ok := th.(map[string]interface{})
			if !ok {
				t.Fatalf("expected thinking object, got %T (%v)", th, th)
			}
			if obj["type"] != tt.wantType {
				t.Errorf("thinking.type = %v, want %q", obj["type"], tt.wantType)
			}
		})
	}
}

// --- Anthropic thinking budget wiring ---

func TestThinkingForBudget(t *testing.T) {
	tests := []struct {
		name      string
		budget    int
		wantNil   bool
		wantBudge int
	}{
		{name: "positive budget", budget: 2048, wantNil: false, wantBudge: 2048},
		{name: "zero budget", budget: 0, wantNil: true},
		{name: "negative budget", budget: -1, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := thinkingForBudget(tt.budget)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil thinking, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil thinking, got nil")
			}
			if got.Type != "enabled" {
				t.Errorf("Type = %q, want %q", got.Type, "enabled")
			}
			if got.BudgetTokens != tt.wantBudge {
				t.Errorf("BudgetTokens = %d, want %d", got.BudgetTokens, tt.wantBudge)
			}
		})
	}
}

func TestAnthropicRequest_ThinkingSerialization(t *testing.T) {
	t.Run("budget set emits thinking object", func(t *testing.T) {
		req := anthropicRequest{
			Model: "claude-3", MaxTokens: 1024,
			Thinking: thinkingForBudget(4096),
		}
		b, _ := json.Marshal(req)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		th, ok := m["thinking"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected thinking object, got %T", m["thinking"])
		}
		if th["type"] != "enabled" {
			t.Errorf("thinking.type = %v, want enabled", th["type"])
		}
		if th["budget_tokens"] != float64(4096) {
			t.Errorf("thinking.budget_tokens = %v, want 4096", th["budget_tokens"])
		}
	})

	t.Run("zero budget omits thinking", func(t *testing.T) {
		req := anthropicRequest{
			Model: "claude-3", MaxTokens: 1024,
			Thinking: thinkingForBudget(0),
		}
		b, _ := json.Marshal(req)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		if _, present := m["thinking"]; present {
			t.Errorf("thinking should be omitted when budget is zero, got %v", m["thinking"])
		}
	})
}
