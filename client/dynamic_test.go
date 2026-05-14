package client

import (
	"os"
	"testing"
)

func TestRegisterDynamicProvider(t *testing.T) {
	// Ensure provider doesn't exist yet
	name := "test-dynamic-provider"
	delete(OpenAICompatibleProviders, name)

	_ = RegisterDynamicProvider(name, "http://localhost:9999/v1", "TEST_DYN_API_KEY")

	info, ok := OpenAICompatibleProviders[name]
	if !ok {
		t.Fatalf("expected provider %q to be registered", name)
	}
	if info.BaseURL != "http://localhost:9999/v1" {
		t.Errorf("expected base URL http://localhost:9999/v1, got %s", info.BaseURL)
	}
	if info.EnvKey != "TEST_DYN_API_KEY" {
		t.Errorf("expected env key TEST_DYN_API_KEY, got %s", info.EnvKey)
	}
	if info.Type != ProviderTypeOpenAICompatible {
		t.Errorf("expected type openai-compatible, got %s", info.Type)
	}
	if !info.SupportsStreaming {
		t.Error("expected SupportsStreaming to be true")
	}
	if info.Compat == nil {
		t.Fatal("expected compat config to be set")
	}
	if info.Compat.MaxTokensField != "max_tokens" {
		t.Errorf("expected max_tokens field, got %s", info.Compat.MaxTokensField)
	}

	// Clean up
	delete(OpenAICompatibleProviders, name)
}

func TestRegisterDynamicProviderNoKey(t *testing.T) {
	name := "test-no-key-provider"
	delete(OpenAICompatibleProviders, name)

	_ = RegisterDynamicProvider(name, "http://localhost:11434/v1", "")

	info, ok := OpenAICompatibleProviders[name]
	if !ok {
		t.Fatalf("expected provider %q to be registered", name)
	}
	if info.EnvKey != "" {
		t.Errorf("expected empty env key, got %s", info.EnvKey)
	}

	// Clean up
	delete(OpenAICompatibleProviders, name)
}

func TestOpenaiBaseFallbackURL(t *testing.T) {
	// Clear both env vars
	os.Unsetenv("OPENAI_API_BASE")
	os.Unsetenv("OPENAI_BASE_URL")

	if u := openaiBaseFallbackURL(); u != "" {
		t.Errorf("expected empty fallback, got %s", u)
	}

	os.Setenv("OPENAI_API_BASE", "http://example.com/v1")
	defer os.Unsetenv("OPENAI_API_BASE")

	if u := openaiBaseFallbackURL(); u != "http://example.com/v1" {
		t.Errorf("expected http://example.com/v1, got %s", u)
	}

	// OPENAI_API_BASE takes precedence
	os.Setenv("OPENAI_BASE_URL", "http://other.com/v1")
	defer os.Unsetenv("OPENAI_BASE_URL")

	if u := openaiBaseFallbackURL(); u != "http://example.com/v1" {
		t.Errorf("expected OPENAI_API_BASE to take precedence, got %s", u)
	}

	// Only OPENAI_BASE_URL
	os.Unsetenv("OPENAI_API_BASE")
	if u := openaiBaseFallbackURL(); u != "http://other.com/v1" {
		t.Errorf("expected http://other.com/v1, got %s", u)
	}
}

func TestGetProviderInfoDynamic(t *testing.T) {
	name := "test-info-dyn"
	delete(OpenAICompatibleProviders, name)

	c := Client(nil)

	// Before registration, should return nil
	if info := c.GetProviderInfo(name); info != nil {
		t.Error("expected nil for unregistered provider")
	}

	_ = RegisterDynamicProvider(name, "http://localhost:5000/v1", "MY_KEY")

	info := c.GetProviderInfo(name)
	if info == nil {
		t.Fatal("expected provider info after registration")
	}
	if info.BaseURL != "http://localhost:5000/v1" {
		t.Errorf("expected base URL http://localhost:5000/v1, got %s", info.BaseURL)
	}

	// Clean up
	delete(OpenAICompatibleProviders, name)
}
