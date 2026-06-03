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
		{"anthropic", TierSonnet, "claude-sonnet-4-6"},
		{"anthropic", TierHaiku, "claude-haiku-4-5-20251001"},
		{"openai", TierOpus, "gpt-4o"},
		{"openai", TierSonnet, "gpt-4o"},
		{"openai", TierHaiku, "gpt-4o-mini"},
		{"gemini", TierOpus, "gemini-2.5-pro-preview-03-25"},
		{"gemini", TierSonnet, "gemini-2.0-flash"},
		{"gemini", TierHaiku, "gemini-2.0-flash-lite"},
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
	if model == "" {
		t.Error("expected bootstrap model with nil catalog")
	}
}

func TestLiveOnlyProvidersHaveNoTierHardcode(t *testing.T) {
	for _, provider := range []string{"anthropic", "openai", "gemini", "grok", "opencodego", "kimi", "xiaomi_mimo_payg", "xiaomi_mimo_token_plan", "canopywave", "z-ai", "openrouter", "ollama"} {
		if got := GetProviderModelCandidates(provider, TierSonnet); len(got) != 0 {
			t.Fatalf("%s tier candidates should be empty, got %v", provider, got)
		}
		if got := GetProviderDefaultModel(provider, &ModelCatalog{}); got != "" {
			t.Fatalf("%s default should be empty without catalog, got %q", provider, got)
		}
	}
}

func TestAllProvidersHaveAtLeastOneModelPerTier(t *testing.T) {
	providers := []string{"anthropic", "openai", "grok", "gemini", "opencodego"}
	tiers := []ModelTier{TierOpus, TierSonnet, TierHaiku}

	for _, provider := range providers {
		for _, tier := range tiers {
			if len(tierModelCandidates(provider, tier)) == 0 {
				t.Errorf("provider %q tier %q has no bootstrap candidates", provider, tier)
			}
		}
	}
}

func TestUnknownProviderReturnsEmpty(t *testing.T) {
	candidates := GetProviderModelCandidates("nonexistent_provider", TierSonnet)
	if len(candidates) != 0 {
		t.Errorf("expected empty candidates for unknown provider, got %v", candidates)
	}

	cat := testDefaultModelCatalog()
	model := GetPreferredProviderModel("nonexistent_provider", TierSonnet, &cat)
	if model != "" {
		t.Errorf("expected empty model for unknown provider, got %q", model)
	}
}

func TestGetProviderModelCandidates_Ordering(t *testing.T) {
	if len(GetProviderModelCandidates("anthropic", TierOpus)) != 0 {
		t.Fatal("picker tier candidates should be empty for live setup provider")
	}
	candidates := tierModelCandidates("anthropic", TierOpus)
	if len(candidates) == 0 || candidates[0] != "claude-opus-4-6" {
		t.Fatalf("bootstrap opus = %v", candidates)
	}
	candidates = tierModelCandidates("anthropic", TierHaiku)
	if len(candidates) == 0 || candidates[0] != "claude-haiku-4-5-20251001" {
		t.Fatalf("bootstrap haiku = %v", candidates)
	}
}

func TestGetProviderModelCandidates_NoDuplicates(t *testing.T) {
	providers := []string{"anthropic", "openai", "gemini", "grok"}
	tiers := []ModelTier{TierOpus, TierSonnet, TierHaiku}

	for _, provider := range providers {
		for _, tier := range tiers {
			candidates := GetProviderModelCandidates(provider, tier)
			seen := make(map[string]bool)
			for _, c := range candidates {
				if seen[c] {
					t.Errorf("duplicate candidate %q for provider %q tier %q", c, provider, tier)
				}
				seen[c] = true
			}
		}
	}
}

func TestProviderModelPool(t *testing.T) {
	if len(providerModelPool("anthropic")) != 0 {
		t.Fatal("expected empty pool for live setup provider")
	}
	if len(tierModelCandidates("anthropic", TierSonnet)) == 0 {
		t.Fatal("expected bootstrap tier candidates")
	}
	pool := providerModelPool("nonexistent")
	if len(pool) != 0 {
		t.Errorf("expected empty pool for nonexistent provider, got %v", pool)
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
