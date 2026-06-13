package registry_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog/opencodego"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestAllProviders_Count(t *testing.T) {
	if n := len(registry.All()); n != 16 {
		t.Fatalf("expected 16 providers, got %d", n)
	}
}

func TestCredentialRegistry_MatchesAll(t *testing.T) {
	if len(registry.CredentialRegistry()) != len(registry.All()) {
		t.Fatal("credential registry should cover all provider specs")
	}
}

func TestLiveFetcherKeys_AllProviders(t *testing.T) {
	keys := registry.LiveFetcherKeys()
	if len(keys) != 16 {
		t.Fatalf("expected 16 live fetcher keys, got %d", len(keys))
	}
}

func TestOpenCodeGo_HasProbeBaseURL(t *testing.T) {
	spec, ok := registry.SpecByProviderID("opencodego")
	if !ok {
		t.Fatal("missing opencodego spec")
	}
	if spec.ProbeBaseURL == "" {
		t.Fatal("opencodego must have a ProbeBaseURL")
	}
	if spec.ProbeBaseURL != opencodego.DefaultBaseURL {
		t.Fatalf("opencodego probe base URL = %q", spec.ProbeBaseURL)
	}
	if spec.ProbeKind != registry.ProbeOpenAIModels {
		t.Fatalf("opencodego probe kind = %q", spec.ProbeKind)
	}
}

func TestProviderSpecs_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		providerID       string
		wantKey          bool
		wantProbeKind    registry.ProbeKind
		wantLiveFetcher  bool
		wantDeploymentID string
	}{
		{"anthropic", "anthropic", true, registry.ProbeAnthropic, true, "anthropic-direct"},
		{"openai", "openai", true, registry.ProbeOpenAIModels, true, "openai-direct"},
		{"azure", "azure", true, registry.ProbeNone, true, "openai-azure"},
		{"gemini", "gemini", true, registry.ProbeGemini, true, "gemini-direct"},
		{"bedrock", "bedrock", true, registry.ProbeNone, true, "anthropic-bedrock"},
		{"vertex", "vertex", true, registry.ProbeNone, true, "gemini-vertex"},
		{"openrouter", "openrouter", true, registry.ProbeOpenAIModels, true, "openrouter"},
		{"grok", "grok", true, registry.ProbeOpenAIModels, true, "grok-direct"},
		{"z-ai", "z-ai", true, registry.ProbeOpenAIModels, true, "z-ai-direct"},
		{"canopywave", "canopywave", true, registry.ProbeOpenAIModels, true, "canopywave"},
		{"opencodego", "opencodego", true, registry.ProbeOpenAIModels, true, "opencodego"},
		{"kimi", "kimi", true, registry.ProbeOpenAIModels, true, "kimi-direct"},
		{"xiaomi_mimo_payg", "xiaomi_mimo_payg", true, registry.ProbeOpenAIModels, true, "xiaomi_mimo_payg-direct"},
		{"xiaomi_mimo_token_plan", "xiaomi_mimo_token_plan", true, registry.ProbeOpenAIModels, true, "xiaomi_mimo_token_plan-direct"},
		{"ollama", "ollama", false, registry.ProbeOllama, true, "ollama-local"},
		{"deepseek", "deepseek", true, registry.ProbeOpenAIModels, true, "deepseek-direct"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, ok := registry.SpecByProviderID(tt.providerID)
			if !ok {
				t.Fatalf("provider %q not found", tt.providerID)
			}
			if spec.RequiresKey != tt.wantKey {
				t.Errorf("RequiresKey = %v, want %v", spec.RequiresKey, tt.wantKey)
			}
			if spec.ProbeKind != tt.wantProbeKind {
				t.Errorf("ProbeKind = %q, want %q", spec.ProbeKind, tt.wantProbeKind)
			}
			if (spec.LiveFetcherKey != "") != tt.wantLiveFetcher {
				t.Errorf("LiveFetcherKey presence = %v, want %v", spec.LiveFetcherKey != "", tt.wantLiveFetcher)
			}
			if spec.DeploymentID != tt.wantDeploymentID {
				t.Errorf("DeploymentID = %q, want %q", spec.DeploymentID, tt.wantDeploymentID)
			}
			if spec.CredentialEnv == "" && spec.RequiresKey {
				t.Error("RequiresKey=true but CredentialEnv is empty")
			}
			if spec.ProbeBaseURL == "" && spec.ProbeKind != registry.ProbeAnthropic && spec.ProbeKind != registry.ProbeGemini && spec.ProbeKind != registry.ProbeOllama && spec.ProbeKind != registry.ProbeNone && spec.ProviderID != "xiaomi_mimo_token_plan" {
				t.Error("ProbeBaseURL is empty for probe kind that requires it")
			}
		})
	}
}
