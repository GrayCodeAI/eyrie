package catalog

import "testing"

func TestAnthropicNameToCanonical(t *testing.T) {
	tests := []struct{ input, want string }{
		{"claude-sonnet-4-6-20250814", "claude-sonnet-4-6"},
		{"us.graycode.claude-opus-4-6-v1:0", "claude-opus-4-6"},
		{"claude-3-5-haiku-20241022", "claude-3-5-haiku"},
		{"claude-3-7-sonnet-20250219", "claude-3-7-sonnet"},
		{"claude-opus-4-5-20251101", "claude-opus-4-5"},
		{"claude-opus-4-1-20250805", "claude-opus-4-1"},
		{"gpt-4o", "gpt-4o"},
	}
	for _, tt := range tests {
		if got := AnthropicNameToCanonical(tt.input); got != tt.want {
			t.Errorf("AnthropicNameToCanonical(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetModelMarketingName(t *testing.T) {
	tests := []struct{ input, want string }{
		{"claude-opus-4-6", "Opus 4.6"},
		{"claude-sonnet-4-6[1m]", "Sonnet 4.6 (1M context)"},
		{"claude-sonnet-4-6", "Sonnet 4.6"},
		{"claude-3-7-sonnet-20250219", "Sonnet 3.7"},
		{"claude-haiku-4-5-20251001", "Haiku 4.5"},
		{"gpt-4o", ""},
	}
	for _, tt := range tests {
		if got := GetModelMarketingName(tt.input); got != tt.want {
			t.Errorf("GetModelMarketingName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetProviderModelCandidates(t *testing.T) {
	candidates := GetProviderModelCandidates("anthropic", TierSonnet)
	if len(candidates) == 0 {
		t.Fatal("expected candidates for anthropic sonnet")
	}
	if candidates[0] != "claude-sonnet-4-6" {
		t.Errorf("expected first candidate claude-sonnet-4-6, got %s", candidates[0])
	}
}

func TestGetProviderDefaultModel(t *testing.T) {
	model := GetProviderDefaultModel("anthropic", nil)
	if model == "" {
		t.Error("expected non-empty default model for anthropic")
	}
	model = GetProviderDefaultModel("openai", nil)
	if model == "" {
		t.Error("expected non-empty default model for openai")
	}
}

func TestGetModelDeprecationWarning(t *testing.T) {
	warning := GetModelDeprecationWarning("claude-3-7-sonnet-20250219", "anthropic")
	if warning == "" {
		t.Error("expected deprecation warning for claude-3-7-sonnet on anthropic")
	}
	warning = GetModelDeprecationWarning("claude-sonnet-4-6", "anthropic")
	if warning != "" {
		t.Errorf("expected no warning for claude-sonnet-4-6, got %q", warning)
	}
}

func TestModelsForProvider(t *testing.T) {
	cat := DefaultModelCatalog()
	models := ModelsForProvider(&cat, "anthropic")
	if len(models) == 0 {
		t.Error("expected anthropic models in default catalog")
	}
	models = ModelsForProvider(&cat, "nonexistent")
	if len(models) != 0 {
		t.Error("expected no models for nonexistent provider")
	}
	models = ModelsForProvider(nil, "anthropic")
	if len(models) != 0 {
		t.Error("expected no models for nil catalog")
	}
}

func TestCanonicalModelIDs(t *testing.T) {
	ids := CanonicalModelIDs()
	if len(ids) != len(AllModelConfigs) {
		t.Errorf("expected %d canonical IDs, got %d", len(AllModelConfigs), len(ids))
	}
}

func TestCanonicalIDToKey(t *testing.T) {
	m := CanonicalIDToKey()
	if _, ok := m["claude-sonnet-4-6"]; !ok {
		t.Error("expected claude-sonnet-4-6 in canonical ID to key map")
	}
}
