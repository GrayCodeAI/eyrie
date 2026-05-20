package catalog

import "testing"

func TestModelEntriesForProvider_OpenRouterUsesOfferings(t *testing.T) {
	compiled := &CompiledCatalogV1{
		ModelsByID: map[string]ModelV1{
			"anthropic/claude-sonnet-4-6": {ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet", ProviderID: "anthropic"},
		},
		OfferingsByDeployment: map[string][]ModelOfferingV1{
			"openrouter": {{
				CanonicalModelID: "anthropic/claude-sonnet-4-6",
				DeploymentID:     "openrouter",
			}},
		},
	}
	entries := ModelEntriesForProvider(compiled, "openrouter")
	if len(entries) != 1 || entries[0].ID != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("openrouter entries: %+v", entries)
	}
}

func TestModelEntriesForProvider_AnthropicFiltersByProvider(t *testing.T) {
	compiled := &CompiledCatalogV1{
		ModelsByID: map[string]ModelV1{
			"anthropic/claude-sonnet-4-6": {ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet", ProviderID: "anthropic"},
			"openai/gpt-4o":               {ID: "openai/gpt-4o", Name: "GPT-4o", ProviderID: "openai"},
		},
	}
	entries := ModelEntriesForProvider(compiled, "anthropic")
	if len(entries) != 1 || entries[0].ID != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("anthropic entries: %+v", entries)
	}
}
