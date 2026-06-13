package catalog_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
)

func TestIsLiveOnlyProvider(t *testing.T) {
	// All providers are now fully dynamic
	allProviders := []string{
		"anthropic", "openai", "gemini", "grok", "canopywave", "z-ai", "openrouter", "ollama", "opencodego",
		"azure", "bedrock", "vertex", "kimi", "xiaomi_mimo_payg", "xiaomi_mimo_token_plan", "deepseek",
	}
	for _, p := range allProviders {
		if !catalog.IsLiveOnlyProvider(p) {
			t.Fatalf("%s should be live-only (all providers are fully dynamic)", p)
		}
	}
}

func TestFirstModelForProvider(t *testing.T) {
	c := catalog.TestSeedCatalogV1()
	c.Models["z-ai/glm-5.1"] = catalog.ModelV1{ID: "z-ai/glm-5.1", ProviderID: "z-ai", Name: "GLM-5.1"}
	compiled, err := catalog.CompileCatalogV1(&c)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.FirstModelForProvider(compiled, "z-ai"); got != "z-ai/glm-5.1" {
		t.Fatalf("FirstModelForProvider = %q", got)
	}
}

func TestGetProviderDefaultModel_AllProvidersEmptyWithoutCatalog(t *testing.T) {
	// All providers return empty without a catalog (fully dynamic)
	allProviders := []string{
		"anthropic", "openai", "gemini", "grok", "bedrock", "kimi",
	}
	for _, p := range allProviders {
		if got := catalog.GetProviderDefaultModel(p, &catalog.ModelCatalog{}); got != "" {
			t.Fatalf("%s: expected empty default without catalog, got %q", p, got)
		}
	}
}
