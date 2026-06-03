package credential

import (
	"context"
	"testing"
)

func TestInferCredentialsFromAPIKey_Anthropic(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "sk-ant-api03-test-key-1234567890")
	if len(got) == 0 {
		t.Fatal("expected anthropic inference")
	}
	if got[0].ProviderID != "anthropic" {
		t.Fatalf("provider = %q, want anthropic", got[0].ProviderID)
	}
	if got[0].EnvVar != "ANTHROPIC_API_KEY" {
		t.Fatalf("env = %q", got[0].EnvVar)
	}
}

func TestInferCredentialsFromAPIKey_OpenRouter(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "sk-or-v1-test-key-1234567890")
	if len(got) == 0 {
		t.Fatal("expected openrouter inference")
	}
	found := false
	for _, c := range got {
		if c.ProviderID == "openrouter" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected openrouter in %#v", got)
	}
}

func TestInferCredentialsFromAPIKey_OpenAI(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "sk-proj-test-key-1234567890")
	if len(got) == 0 {
		t.Fatal("expected openai inference")
	}
	if got[0].ProviderID != "openai" {
		t.Fatalf("provider = %q, want openai", got[0].ProviderID)
	}
}

func TestInferCredentialsFromAPIKey_GenericOpenAICompatible(t *testing.T) {
	got := InferCredentialsFromAPIKey(ContextWithoutProbeDisambiguation(context.Background()), "sk-test-key-that-could-belong-to-any-compatible-provider")
	if len(got) != 0 {
		t.Fatalf("generic sk- keys should not infer a provider, got %#v", got)
	}
}

func TestInferCredentialsFromAPIKey_Gemini(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "AIzaSyD-test-key-1234567890")
	if len(got) == 0 {
		t.Fatal("expected gemini inference")
	}
	if got[0].ProviderID != "gemini" {
		t.Fatalf("provider = %q, want gemini", got[0].ProviderID)
	}
}

func TestInferCredentialsFromAPIKey_Unknown(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "not-a-real-key-format")
	if len(got) != 0 {
		t.Fatalf("expected no inference, got %#v", got)
	}
}

func TestInferCredentialsFromAPIKey_Placeholder(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "your-api-key")
	if len(got) != 0 {
		t.Fatalf("expected no inference for placeholder, got %#v", got)
	}
}
