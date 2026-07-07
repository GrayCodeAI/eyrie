package catalog

import (
	"testing"
	"time"
)

func testPolicyCatalog(t *testing.T) *CompiledCatalog {
	t.Helper()
	now := time.Now().UTC()
	cat := &Catalog{
		SchemaVersion: CatalogSchemaVersion,
		GeneratedAt:   now,
		StaleAfter:    now.Add(time.Hour),
		Providers: map[string]Provider{
			"anthropic": {ID: "anthropic", Name: "Anthropic"},
		},
		Protocols: map[string]Protocol{
			"anthropic-messages": {ID: "anthropic-messages", Name: "Anthropic Messages"},
		},
		Deployments: map[string]Deployment{
			"anthropic-direct": {
				ID:                    "anthropic-direct",
				Name:                  "Anthropic",
				ProviderID:            "anthropic",
				APIProtocolID:         "anthropic-messages",
				AdapterConstructor:    "anthropic",
				NativeModelIDSource:   NativeModelIDCatalogKnown,
				ModelMappingsRequired: false,
			},
		},
		Models: map[string]Model{
			"anthropic/claude-haiku": {
				ID:         "anthropic/claude-haiku",
				ProviderID: "anthropic",
				Name:       "Claude Haiku",
				Family:     "haiku",
			},
			"anthropic/claude-sonnet": {
				ID:         "anthropic/claude-sonnet",
				ProviderID: "anthropic",
				Name:       "Claude Sonnet",
				Family:     "sonnet",
			},
			"anthropic/claude-opus": {
				ID:         "anthropic/claude-opus",
				ProviderID: "anthropic",
				Name:       "Claude Opus",
				Family:     "opus",
			},
		},
		Offerings: []ModelOffering{
			testPolicyOffering("anthropic/claude-haiku", "claude-haiku", 0.25, 1),
			testPolicyOffering("anthropic/claude-sonnet", "claude-sonnet", 3, 15),
			testPolicyOffering("anthropic/claude-opus", "claude-opus", 15, 75),
		},
		Aliases: map[string]string{
			"claude-haiku":  "anthropic/claude-haiku",
			"claude-sonnet": "anthropic/claude-sonnet",
			"claude-opus":   "anthropic/claude-opus",
		},
	}
	compiled, err := CompileCatalog(cat)
	if err != nil {
		t.Fatalf("CompileCatalog failed: %v", err)
	}
	return compiled
}

func testPolicyOffering(canonicalModelID, nativeModelID string, input, output float64) ModelOffering {
	return ModelOffering{
		ID:               "anthropic-direct:" + nativeModelID,
		CanonicalModelID: canonicalModelID,
		DeploymentID:     "anthropic-direct",
		NativeModelID:    nativeModelID,
		Pricing: Pricing{
			Status:     PricingKnown,
			Currency:   "USD",
			RatesPer1M: map[string]float64{"input_tokens": input, "output_tokens": output},
		},
	}
}

func TestModelPolicyPreferredProviderModel(t *testing.T) {
	t.Parallel()
	compiled := testPolicyCatalog(t)

	tests := []struct {
		tier ModelTier
		want string
	}{
		{TierHaiku, "claude-haiku"},
		{TierSonnet, "claude-sonnet"},
		{TierOpus, "claude-opus"},
	}
	for _, tt := range tests {
		got := PreferredProviderModel(compiled, "anthropic", tt.tier, "")
		if got != tt.want {
			t.Fatalf("PreferredProviderModel(%s) = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestModelPolicyPreferredModelsForTier(t *testing.T) {
	t.Parallel()
	compiled := testPolicyCatalog(t)

	got := PreferredModelsForTier(compiled, "anthropic", TierHaiku, 3)
	if len(got) != 1 || got[0] != "claude-haiku" {
		t.Fatalf("PreferredModelsForTier = %v, want [claude-haiku]", got)
	}
}

func TestModelPolicyCostTierOf(t *testing.T) {
	t.Parallel()
	compiled := testPolicyCatalog(t)

	tests := []struct {
		model string
		want  ModelCostTier
	}{
		{"claude-haiku", CostTierCheap},
		{"claude-sonnet", CostTierMid},
		{"claude-opus", CostTierExpensive},
		{"unknown-model-mini", CostTierCheap},
	}
	for _, tt := range tests {
		got := ModelCostTierOf(compiled, tt.model)
		if got != tt.want {
			t.Fatalf("ModelCostTierOf(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestModelPolicyDefaultRolesV1(t *testing.T) {
	t.Parallel()
	compiled := testPolicyCatalog(t)

	roles := DefaultModelRolesV1(compiled, "claude-opus")
	if roles.Planner != "claude-opus" || roles.Coder != "claude-opus" || roles.Reviewer != "claude-opus" {
		t.Fatalf("interactive roles should use primary model, got %+v", roles)
	}
	if roles.Commit != "claude-haiku" {
		t.Fatalf("commit role = %q, want cheapest same-provider model", roles.Commit)
	}
	if got := ModelForRoleV1(compiled, ModelRoleAssignments{Coder: "coder"}, ModelRolePlanner); got != "coder" {
		t.Fatalf("ModelForRoleV1 fallback = %q, want coder", got)
	}
}
