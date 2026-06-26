package catalog

import "testing"

func TestModelEntriesForProvider_OpenRouterUsesOfferings(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestCanonicalModelForProviderNative_PrefersDeploymentOverGlobalAlias(t *testing.T) {
	t.Parallel()
	compiled := &CompiledCatalogV1{
		Catalog: &CatalogV1{
			Aliases: map[string]string{
				"mimo-v2.5-pro": "opencodego/mimo-v2.5-pro",
			},
		},
		ModelsByID: map[string]ModelV1{
			"opencodego/mimo-v2.5-pro":             {ID: "opencodego/mimo-v2.5-pro", ProviderID: "opencodego"},
			"xiaomi_mimo_token_plan/mimo-v2.5-pro": {ID: "xiaomi_mimo_token_plan/mimo-v2.5-pro", ProviderID: "xiaomi_mimo_token_plan"},
		},
		OfferingsByDeployment: map[string][]ModelOfferingV1{
			"xiaomi_mimo_token_plan-direct": {{
				CanonicalModelID: "xiaomi_mimo_token_plan/mimo-v2.5-pro",
				DeploymentID:     "xiaomi_mimo_token_plan-direct",
				NativeModelID:    "mimo-v2.5-pro",
			}},
		},
	}
	canonical, ok := CanonicalModelForProviderNative(compiled, "xiaomi_mimo_token_plan", "mimo-v2.5-pro")
	if !ok || canonical != "xiaomi_mimo_token_plan/mimo-v2.5-pro" {
		t.Fatalf("canonical=%q ok=%v", canonical, ok)
	}
}

func TestModelEntriesForProvider_AnthropicUsesDirectDeploymentOfferings(t *testing.T) {
	t.Parallel()
	compiled := &CompiledCatalogV1{
		ModelsByID: map[string]ModelV1{
			"anthropic/claude-sonnet-4-6": {ID: "anthropic/claude-sonnet-4-6", Name: "Sonnet", ProviderID: "anthropic"},
			"openai/gpt-4o":               {ID: "openai/gpt-4o", Name: "GPT-4o", ProviderID: "openai"},
		},
		OfferingsByDeployment: map[string][]ModelOfferingV1{
			"anthropic-direct": {{
				CanonicalModelID: "anthropic/claude-sonnet-4-6",
				DeploymentID:     "anthropic-direct",
				NativeModelID:    "claude-sonnet-4-6",
			}},
		},
	}
	entries := ModelEntriesForProvider(compiled, "anthropic")
	if len(entries) != 1 || entries[0].ID != "claude-sonnet-4-6" {
		t.Fatalf("anthropic entries: %+v", entries)
	}
}
