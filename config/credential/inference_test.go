package credential

import (
	"context"
	"testing"
)

func TestInferCredentialsFromAPIKey_ReturnsNil(t *testing.T) {
	got := InferCredentialsFromAPIKey(context.Background(), "sk-ant-api03-test-key-1234567890")
	if len(got) != 0 {
		t.Fatalf("expected no prefix inference, got %d", len(got))
	}
}

func TestInferenceForProvider_Anthropic(t *testing.T) {
	inf, err := InferenceForProvider("anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if inf.ProviderID != "anthropic" || inf.DeploymentID != "anthropic-direct" || inf.EnvVar != "ANTHROPIC_API_KEY" {
		t.Fatalf("unexpected inference: %+v", inf)
	}
}

func TestInferenceForProvider_Ollama(t *testing.T) {
	inf, err := InferenceForProvider("ollama")
	if err != nil {
		t.Fatal(err)
	}
	if inf.EnvVar != "OLLAMA_BASE_URL" || inf.DeploymentID != "ollama-local" {
		t.Fatalf("unexpected ollama inference: %+v", inf)
	}
}

func TestInferenceForProvider_Unknown(t *testing.T) {
	if _, err := InferenceForProvider("not-a-provider"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}