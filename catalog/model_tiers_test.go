package catalog

import "testing"

func TestGetPreferredProviderModel_AllTiers(t *testing.T) {
	cat := DefaultModelCatalog()
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
	// With nil catalog, function should still return a model (loads default internally)
	model := GetPreferredProviderModel("anthropic", TierSonnet, nil)
	if model == "" {
		t.Error("expected non-empty model with nil catalog")
	}
}

func TestLiveOnlyProvidersHaveNoTierHardcode(t *testing.T) {
	for _, provider := range []string{"canopywave", "z-ai", "openrouter", "ollama"} {
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
			candidates := GetProviderModelCandidates(provider, tier)
			if len(candidates) == 0 {
				t.Errorf("provider %q tier %q has no model candidates", provider, tier)
			}
		}
	}
}

func TestUnknownProviderReturnsEmpty(t *testing.T) {
	candidates := GetProviderModelCandidates("nonexistent_provider", TierSonnet)
	if len(candidates) != 0 {
		t.Errorf("expected empty candidates for unknown provider, got %v", candidates)
	}

	cat := DefaultModelCatalog()
	model := GetPreferredProviderModel("nonexistent_provider", TierSonnet, &cat)
	if model != "" {
		t.Errorf("expected empty model for unknown provider, got %q", model)
	}
}

func TestGetProviderModelCandidates_Ordering(t *testing.T) {
	// Preferred model should come first
	candidates := GetProviderModelCandidates("anthropic", TierOpus)
	if len(candidates) == 0 {
		t.Fatal("expected candidates")
	}
	// The preferred key for anthropic opus is opus46 -> "claude-opus-4-6"
	if candidates[0] != "claude-opus-4-6" {
		t.Errorf("expected first candidate to be claude-opus-4-6, got %s", candidates[0])
	}

	// Haiku tier should have haiku45 first
	candidates = GetProviderModelCandidates("anthropic", TierHaiku)
	if len(candidates) == 0 {
		t.Fatal("expected candidates for haiku")
	}
	if candidates[0] != "claude-haiku-4-5-20251001" {
		t.Errorf("expected first haiku candidate to be claude-haiku-4-5-20251001, got %s", candidates[0])
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
	pool := providerModelPool("anthropic")
	if len(pool) == 0 {
		t.Fatal("expected non-empty pool for anthropic")
	}
	// Should contain all unique anthropic model IDs
	seen := make(map[string]bool)
	for _, m := range pool {
		if seen[m] {
			t.Errorf("duplicate in pool: %s", m)
		}
		seen[m] = true
	}

	pool = providerModelPool("nonexistent")
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
