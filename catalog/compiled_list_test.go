package catalog

import "testing"

func TestModelEntriesForProvider_OpenRouterUsesOfferings(t *testing.T) {
	raw := []byte(`{"id":"anthropic/claude-sonnet-4-6","architecture":{"modality":"text"}}`)
	compiled := &CompiledCatalogV1{
		ModelsByID: map[string]ModelV1{
			"anthropic/claude-sonnet-4-6": {ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet", ProviderID: "anthropic"},
		},
		OfferingsByDeployment: map[string][]ModelOfferingV1{
			"openrouter": {{
				CanonicalModelID: "anthropic/claude-sonnet-4-6",
				DeploymentID:     "openrouter",
				NativeModelID:    "anthropic/claude-sonnet-4-6",
				LiveMetadata:     raw,
			}},
		},
	}
	entries := ModelEntriesForProvider(compiled, "openrouter")
	if len(entries) != 1 || entries[0].ID != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("openrouter entries: %+v", entries)
	}
	if string(entries[0].LiveMetadata) != string(raw) {
		t.Fatalf("live metadata missing: %+v", entries[0])
	}
}

func TestModelEntriesForProvider_CanopyWaveUsesDeploymentOfferings(t *testing.T) {
	raw := []byte(`{"id":"moonshotai/kimi-k2.6","name":"Kimi K2.6","owned_by":"moonshotai"}`)
	compiled := &CompiledCatalogV1{
		ModelsByID: map[string]ModelV1{
			"moonshotai/kimi-k2.6": {ID: "moonshotai/kimi-k2.6", Name: "Kimi K2.6", ProviderID: "moonshotai"},
		},
		OfferingsByDeployment: map[string][]ModelOfferingV1{
			"canopywave": {{
				CanonicalModelID: "moonshotai/kimi-k2.6",
				DeploymentID:     "canopywave",
				NativeModelID:    "moonshotai/kimi-k2.6",
				LiveMetadata:     raw,
			}},
		},
	}
	entries := ModelEntriesForProvider(compiled, "canopywave")
	if len(entries) != 1 || entries[0].ID != "moonshotai/kimi-k2.6" {
		t.Fatalf("canopywave entries: %+v", entries)
	}
	if string(entries[0].LiveMetadata) != string(raw) {
		t.Fatalf("live metadata missing: %+v", entries[0])
	}
}

func TestModelEntriesForProvider_GeminiUsesDirectDeploymentOfferings(t *testing.T) {
	compiled := &CompiledCatalogV1{
		ModelsByID: map[string]ModelV1{
			"gemini-flash": {ID: "gemini-flash", Name: "Flash", ProviderID: "google"},
			"gemini-pro":   {ID: "gemini-pro", Name: "Pro", ProviderID: "google"},
			"other-model":  {ID: "other-model", Name: "Other", ProviderID: "google"},
		},
		OfferingsByDeployment: map[string][]ModelOfferingV1{
			"gemini-direct": {
				{CanonicalModelID: "gemini-flash", DeploymentID: "gemini-direct", NativeModelID: "gemini-flash"},
				{CanonicalModelID: "gemini-pro", DeploymentID: "gemini-direct", NativeModelID: "gemini-pro"},
			},
		},
	}
	entries := ModelEntriesForProvider(compiled, "gemini")
	if len(entries) != 2 {
		t.Fatalf("expected 2 gemini-direct offerings, got %d: %+v", len(entries), entries)
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
