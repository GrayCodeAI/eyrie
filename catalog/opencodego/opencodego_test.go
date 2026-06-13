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

func TestChatCompletionsSupported(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"kimi-k2.6", true},
		{"glm-5.1", true},
		{"deepseek-v4-flash", true},
		{"mimo-v2.5", true},
		{"minimax-m2.7", false},
		{"minimax-m3", false},
		{"qwen3.7-max", false},
		{"qwen3.6-plus", false},
		{"opencode-go/kimi-k2.6", true},
		{"opencodego/qwen3.7-plus", false},
	}
	for _, tc := range tests {
		if got := ChatCompletionsSupported(tc.model); got != tc.want {
			t.Errorf("ChatCompletionsSupported(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}
