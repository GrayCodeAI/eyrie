package credentials

import (
	"testing"
)

func TestAccountForEnv(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "uppercase", input: "ANTHROPIC_API_KEY", want: "anthropic_api_key"},
		{name: "mixed_case", input: "OpenAi_API_Key", want: "openai_api_key"},
		{name: "already_lower", input: "openrouter_api_key", want: "openrouter_api_key"},
		{name: "leading_trailing_spaces", input: "  GEMINI_API_KEY  ", want: "gemini_api_key"},
		{name: "tab_whitespace", input: "\tGROK_API_KEY\t", want: "grok_api_key"},
		{name: "empty", input: "", want: ""},
		{name: "only_spaces", input: "   ", want: ""},
		{name: "with_hyphens", input: "MY-CUSTOM-KEY", want: "my-custom-key"},
		{name: "single_char", input: "X", want: "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AccountForEnv(tt.input)
			if got != tt.want {
				t.Errorf("AccountForEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnvForAccount(t *testing.T) {
	tests := []struct {
		name    string
		account string
		want    string
	}{
		{name: "anthropic", account: "anthropic_api_key", want: "ANTHROPIC_API_KEY"},
		{name: "openai", account: "openai_api_key", want: "OPENAI_API_KEY"},
		{name: "openrouter", account: "openrouter_api_key", want: "OPENROUTER_API_KEY"},
		{name: "gemini", account: "gemini_api_key", want: "GEMINI_API_KEY"},
		{name: "grok_alias", account: "grok_api_key", want: "GROK_API_KEY"},
		{name: "xai_alias", account: "xai_api_key", want: "GROK_API_KEY"},
		{name: "zai", account: "zai_api_key", want: "ZAI_API_KEY"},
		{name: "canopywave", account: "canopywave_api_key", want: "CANOPYWAVE_API_KEY"},
		{name: "opencodego", account: "opencodego_api_key", want: "OPENCODEGO_API_KEY"},
		{name: "kimi_alias", account: "kimi_api_key", want: "KIMI_API_KEY"},
		{name: "moonshot_alias", account: "moonshot_api_key", want: "KIMI_API_KEY"},
		{name: "xiaomi_alias", account: "xiaomi_api_key", want: "XIAOMI_API_KEY"},
		{name: "mimo_alias", account: "mimo_api_key", want: "XIAOMI_API_KEY"},
		{name: "ollama_base_url", account: "ollama_base_url", want: "OLLAMA_BASE_URL"},
		{name: "unknown_passthrough", account: "custom_api_key", want: "CUSTOM_API_KEY"},
		{name: "unknown_with_hyphens", account: "my-custom-key", want: "MY_CUSTOM_KEY"},
		{name: "mixed_case_known", account: "Anthropic_Api_Key", want: "ANTHROPIC_API_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnvForAccount(tt.account)
			if got != tt.want {
				t.Errorf("EnvForAccount(%q) = %q, want %q", tt.account, got, tt.want)
			}
		})
	}
}

func TestAccountForEnv_EnvForAccount_RoundTrip(t *testing.T) {
	// For known keys, EnvForAccount(AccountForEnv(key)) should return the canonical form.
	canonical := []string{
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY",
		"GEMINI_API_KEY", "ZAI_API_KEY", "CANOPYWAVE_API_KEY",
		"OPENCODEGO_API_KEY", "OLLAMA_BASE_URL",
	}
	for _, envKey := range canonical {
		t.Run(envKey, func(t *testing.T) {
			account := AccountForEnv(envKey)
			got := EnvForAccount(account)
			if got != envKey {
				t.Errorf("EnvForAccount(AccountForEnv(%q)) = %q, want %q", envKey, got, envKey)
			}
		})
	}
}
