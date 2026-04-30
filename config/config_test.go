package config

import (
	"os"
	"testing"
)

func TestResolveProviderRequest(t *testing.T) {
	r := ResolveProviderRequest("gpt-4o", "", "")
	if r.ResolvedModel != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", r.ResolvedModel)
	}
	if r.BaseURL != DefaultOpenAIBaseURL {
		t.Errorf("expected default base URL, got %s", r.BaseURL)
	}
	if r.Transport != TransportChatCompletions {
		t.Errorf("expected chat_completions transport")
	}
}

func TestResolveProviderRequestWithReasoning(t *testing.T) {
	r := ResolveProviderRequest("gpt-4o?reasoning=high", "", "")
	if r.ResolvedModel != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", r.ResolvedModel)
	}
	if r.Reasoning == nil || r.Reasoning.Effort != ReasoningHigh {
		t.Error("expected reasoning effort high")
	}
}

func TestIsLocalProviderURL(t *testing.T) {
	tests := []struct{ url string; want bool }{
		{"http://localhost:11434/v1", true},
		{"http://127.0.0.1:8080", true},
		{"https://api.openai.com/v1", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsLocalProviderURL(tt.url); got != tt.want {
			t.Errorf("IsLocalProviderURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestIsOpenAICompatibleRuntimeEnabled(t *testing.T) {
	// Clear all keys first
	for _, k := range []string{"OPENROUTER_API_KEY", "GROK_API_KEY", "XAI_API_KEY", "GEMINI_API_KEY", "ANTHROPIC_API_KEY", "CANOPYWAVE_API_KEY", "OPENAI_API_KEY", "OPENCODEGO_API_KEY", "OLLAMA_BASE_URL"} {
		os.Unsetenv(k)
	}
	if IsOpenAICompatibleRuntimeEnabled() {
		t.Error("expected false with no keys set")
	}
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")
	if !IsOpenAICompatibleRuntimeEnabled() {
		t.Error("expected true with OPENAI_API_KEY set")
	}
}

func TestNormalizeOllamaOpenAIBaseURL(t *testing.T) {
	tests := []struct{ input, want string }{
		{"http://localhost:11434", "http://localhost:11434/v1"},
		{"http://localhost:11434/v1", "http://localhost:11434/v1"},
		{"http://localhost:11434/", "http://localhost:11434/v1"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeOllamaOpenAIBaseURL(tt.input); got != tt.want {
			t.Errorf("NormalizeOllamaOpenAIBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProviderDetectionOrder(t *testing.T) {
	if len(APIProviderDetectionOrder) != 8 {
		t.Errorf("expected 8 providers in detection order, got %d", len(APIProviderDetectionOrder))
	}
	if APIProviderDetectionOrder[0] != ProviderAnthropic {
		t.Error("expected anthropic first in detection order")
	}
}

func TestValidateAPIKey(t *testing.T) {
	if msg := ValidateAPIKey("", "test"); msg == "" {
		t.Error("expected error for empty key")
	}
	if msg := ValidateAPIKey("SUA_CHAVE", "test"); msg == "" {
		t.Error("expected error for placeholder key")
	}
	if msg := ValidateAPIKey("short", "test"); msg == "" {
		t.Error("expected error for short key")
	}
	if msg := ValidateAPIKey("sk-valid-api-key-here-1234567890", "test"); msg != "" {
		t.Errorf("expected no error for valid key, got %q", msg)
	}
}
