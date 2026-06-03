package registry_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestAllProviders_Count(t *testing.T) {
	if n := len(registry.All()); n != 11 {
		t.Fatalf("expected 11 providers, got %d", n)
	}
}

func TestCredentialRegistry_MatchesAll(t *testing.T) {
	if len(registry.CredentialRegistry()) != len(registry.All()) {
		t.Fatal("credential registry should cover all provider specs")
	}
}

func TestLiveFetcherKeys_AllProviders(t *testing.T) {
	keys := registry.LiveFetcherKeys()
	if len(keys) != 11 {
		t.Fatalf("expected 11 live fetcher keys, got %d", len(keys))
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
	if spec.ProbeBaseURL != "https://opencode.ai/zen/go/v1" {
		t.Fatalf("opencodego probe base URL = %q", spec.ProbeBaseURL)
	}
	if spec.ProbeKind != registry.ProbeOpenAIModels {
		t.Fatalf("opencodego probe kind = %q", spec.ProbeKind)
	}
}

func TestOllamaStrategy_LiveOnly(t *testing.T) {
	spec, ok := registry.SpecByProviderID("ollama")
	if !ok {
		t.Fatal("missing ollama spec")
	}
	if spec.ModelStrategy != registry.StrategyLiveOnly {
		t.Fatalf("ollama strategy = %q", spec.ModelStrategy)
	}
}

func TestProviderSpecs_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		providerID       string
		wantKey          bool
		wantStrategy     registry.ModelStrategy
		wantProbeKind    registry.ProbeKind
		wantLiveFetcher  bool
		wantDeploymentID string
	}{
		{"anthropic", "anthropic", true, registry.StrategyRemoteThenLive, registry.ProbeAnthropic, true, "anthropic-direct"},
		{"openai", "openai", true, registry.StrategyRemoteThenLive, registry.ProbeOpenAIModels, true, "openai-direct"},
		{"gemini", "gemini", true, registry.StrategyRemoteThenLive, registry.ProbeGemini, true, "gemini-direct"},
		{"openrouter", "openrouter", true, registry.StrategyLiveOnly, registry.ProbeOpenAIModels, true, "openrouter"},
		{"grok", "grok", true, registry.StrategyRemoteThenLive, registry.ProbeOpenAIModels, true, "grok-direct"},
		{"z-ai", "z-ai", true, registry.StrategyLiveOnly, registry.ProbeOpenAIModels, true, "z-ai-direct"},
		{"canopywave", "canopywave", true, registry.StrategyLiveOnly, registry.ProbeOpenAIModels, true, "canopywave"},
		{"opencodego", "opencodego", true, registry.StrategyRemoteThenLive, registry.ProbeOpenAIModels, true, "opencodego"},
		{"kimi", "kimi", true, registry.StrategyLiveOnly, registry.ProbeOpenAIModels, true, "kimi-direct"},
		{"xiaomi", "xiaomi", true, registry.StrategyLiveOnly, registry.ProbeOpenAIModels, true, "xiaomi-direct"},
		{"ollama", "ollama", false, registry.StrategyLiveOnly, registry.ProbeOllama, true, "ollama-local"},
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
			if spec.ModelStrategy != tt.wantStrategy {
				t.Errorf("ModelStrategy = %q, want %q", spec.ModelStrategy, tt.wantStrategy)
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
			if spec.ProbeBaseURL == "" && spec.ProbeKind != registry.ProbeAnthropic && spec.ProbeKind != registry.ProbeGemini && spec.ProbeKind != registry.ProbeOllama && spec.ProbeKind != registry.ProbeNone {
				t.Error("ProbeBaseURL is empty for probe kind that requires it")
			}
		})
	}
}
