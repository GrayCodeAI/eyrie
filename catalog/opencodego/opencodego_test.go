package opencodego

import "testing"

func TestNativeModelID(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"kimi-k2.6", "kimi-k2.6"},
		{"opencode-go/kimi-k2.7-code", "kimi-k2.7-code"},
		{"opencodego/glm-5.1", "glm-5.1"},
		{"  opencode-go/deepseek-v4-flash  ", "deepseek-v4-flash"},
	}
	for _, tc := range tests {
		if got := NativeModelID(tc.in); got != tc.want {
			t.Errorf("NativeModelID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestProtocolForModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"kimi-k2.6", "openai"},
		{"glm-5.1", "openai"},
		{"deepseek-v4-flash", "openai"},
		{"mimo-v2.5", "openai"},
		{"minimax-m2.7", "anthropic"},
		{"minimax-m3", "anthropic"},
		{"qwen3.7-max", "anthropic"},
		{"qwen3.6-plus", "anthropic"},
		{"qwen3.5-plus", "anthropic"},
		{"opencode-go/kimi-k2.6", "openai"},
		{"opencodego/minimax-m2.5", "anthropic"},
	}
	for _, tc := range tests {
		if got := ProtocolForModel(tc.model); got != tc.want {
			t.Errorf("ProtocolForModel(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}
