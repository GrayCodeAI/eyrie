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

func TestUsesMessagesAPI(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"kimi-k2.6", false},
		{"glm-5.1", false},
		{"deepseek-v4-flash", false},
		{"mimo-v2.5", false},
		{"minimax-m2.7", true},
		{"minimax-m3", true},
		{"qwen3.7-max", true},
		{"qwen3.6-plus", true},
		{"qwen3.5-plus", true},
		{"opencode-go/kimi-k2.6", false},
		{"opencodego/minimax-m2.5", true},
	}
	for _, tc := range tests {
		if got := UsesMessagesAPI(tc.model); got != tc.want {
			t.Errorf("UsesMessagesAPI(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}
