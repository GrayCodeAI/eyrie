package catalog

import "testing"

func testDefaultModelCatalog() ModelCatalog {
	return ModelCatalog{
		Source: "test",
		Providers: map[string][]ModelCatalogEntry{
			"anthropic": {
				{ID: "claude-opus-4-6", DisplayName: "Claude Opus 4.6"},
				{ID: "claude-sonnet-4-6", DisplayName: "Claude Sonnet 4.6"},
				{ID: "claude-haiku-4-5-20251001", DisplayName: "Claude Haiku 4.5"},
				{ID: "claude-haiku-3-5", DisplayName: "Claude Haiku 3.5"},
			},
			"openai": {
				{ID: "gpt-4o", DisplayName: "GPT-4o"},
				{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini"},
			},
			"gemini": {
				{ID: "gemini-2.5-pro-preview-03-25", DisplayName: "Gemini 2.5 Pro"},
				{ID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash"},
				{ID: "gemini-2.0-flash-lite", DisplayName: "Gemini 2.0 Flash Lite"},
			},
			"grok": {
				{ID: "grok-2", DisplayName: "Grok 2"},
			},
		},
	}
}

func TestGetPreferredProviderModel_AllTiers(t *testing.T) {
	cat := testDefaultModelCatalog()
	tests := []struct {
		provider string
		tier     ModelTier
		want     string
	}{
		{"anthropic", TierOpus, "claude-opus-4-6"},
		{"anthropic", TierSonnet, "claude-opus-4-6"},
		{"anthropic", TierHaiku, "claude-opus-4-6"},
		{"openai", TierOpus, "gpt-4o"},
		{"openai", TierSonnet, "gpt-4o"},
		{"openai", TierHaiku, "gpt-4o"},
		{"gemini", TierOpus, "gemini-2.5-pro-preview-03-25"},
		{"gemini", TierSonnet, "gemini-2.5-pro-preview-03-25"},
		{"gemini", TierHaiku, "gemini-2.5-pro-preview-03-25"},
		{"grok", TierOpus, "grok-2"},
		{"grok", TierSonnet, "grok-2"},
		{"grok", TierHaiku, "grok-2"},
	}
	for _, tt := range tests {
		got := GetPreferredProviderModel(tt.provider, tt.tier, &cat)
		if got != tt.want {
			t.Errorf("GetPreferredProviderModel(%q, %q) = %q, want %q", tt.provider, tt.tier, got, tt.want)
		}
	}
}

func TestGetPreferredProviderModel_NilCatalog(t *testing.T) {
	model := GetPreferredProviderModel("anthropic", TierSonnet, nil)
	if model != "" {
		t.Errorf("expected empty model with nil catalog, got %q", model)
	}
}

func TestGetPreferredProviderModel_EmptyCatalog(t *testing.T) {
	cat := ModelCatalog{Source: "test", Providers: map[string][]ModelCatalogEntry{}}
	model := GetPreferredProviderModel("anthropic", TierSonnet, &cat)
	if model != "" {
		t.Errorf("expected empty model with empty catalog, got %q", model)
	}
}

func TestAllProvidersReturnDefaultEmptyWithoutCatalog(t *testing.T) {
	allProviders := []string{
		"anthropic", "openai", "gemini", "grok", "opencodego",
		"canopywave", "z_ai", "openrouter", "ollama",
		"azure", "bedrock", "vertex", "kimi",
		"xiaomi_mimo_payg", "xiaomi_mimo_token_plan", "deepseek",
	}
	for _, provider := range allProviders {
		if got := GetProviderDefaultModel(provider, nil); got != "" {
			t.Fatalf("%s default should be empty without catalog, got %q", provider, got)
		}
	}
}

func TestUnknownProviderReturnsEmpty(t *testing.T) {
	cat := testDefaultModelCatalog()
	model := GetPreferredProviderModel("nonexistent_provider", TierSonnet, &cat)
	if model != "" {
		t.Errorf("expected empty model for unknown provider, got %q", model)
	}
}

func TestModelTierAliases(t *testing.T) {
	if len(ModelTierAliases) != 3 {
		t.Errorf("expected 3 tier aliases, got %d", len(ModelTierAliases))
	}
	found := map[ModelTier]bool{}
	for _, tier := range ModelTierAliases {
		found[tier] = true
	}
	if !found[TierOpus] || !found[TierSonnet] || !found[TierHaiku] {
		t.Error("ModelTierAliases missing expected tiers")
	}
}
