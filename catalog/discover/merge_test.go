package discover_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/discover"
)

func TestMergeCatalogV1WithPolicy_ReplacesDeploymentOfferings(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Offerings = append(dst.Offerings, catalog.ModelOfferingV1{
		ID: "canopywave:old-model", CanonicalModelID: "z-ai/old", DeploymentID: "canopywave",
		NativeModelID: "old-model", Pricing: catalog.PricingV1{Status: catalog.PricingUnknown},
	})
	src := catalog.CatalogV1FromLegacy(catalog.ModelCatalog{
		Providers: map[string][]catalog.ModelCatalogEntry{
			"canopywave": {{ID: "moonshotai/kimi-k2.6"}},
		},
	})
	out := discover.MergeCatalogV1WithPolicy(&dst, &src, discover.MergePolicy{
		PreferLive:                 true,
		ReplaceDeploymentOfferings: []string{"canopywave"},
	})
	for _, o := range out.Offerings {
		if o.DeploymentID == "canopywave" && o.NativeModelID == "old-model" {
			t.Fatal("stale canopywave offering should be removed")
		}
	}
	found := false
	for _, o := range out.Offerings {
		if o.DeploymentID == "canopywave" && o.NativeModelID == "moonshotai/kimi-k2.6" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected live canopywave offering after replace merge")
	}
}

func TestMergeCatalogV1WithPolicy_PreferLiveReplacesExistingModel(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Models["anthropic/claude-sonnet-4-6"] = catalog.ModelV1{
		ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic", Name: "Claude Sonnet",
		ContextWindow: 100000, MaxOutput: 4096, Aliases: []string{"claude-sonnet"},
	}
	src := &catalog.CatalogV1{
		Models: map[string]catalog.ModelV1{
			"anthropic/claude-sonnet-4-6": {
				ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic", Name: "Claude Sonnet 4.6",
				ContextWindow: 200000, MaxOutput: 8192, Aliases: []string{"sonnet-4-6"},
				Provenance: &catalog.CatalogProvenanceV1{Source: "live", ObservedAt: time.Now().UTC()},
			},
		},
	}
	out := discover.MergeCatalogV1WithPolicy(&dst, src, discover.MergePolicy{
		PreferLiveProviders: []string{"anthropic"},
	})
	got := out.Models["anthropic/claude-sonnet-4-6"]
	if got.ContextWindow != 200000 || got.MaxOutput != 8192 {
		t.Fatalf("context/max = %d/%d", got.ContextWindow, got.MaxOutput)
	}
	if got.Name != "Claude Sonnet 4.6" {
		t.Fatalf("name = %q", got.Name)
	}
	if len(got.Aliases) != 1 || got.Aliases[0] != "sonnet-4-6" {
		t.Fatalf("aliases = %#v (expected live replacement)", got.Aliases)
	}
}

func TestMergeCatalogV1WithPolicy_PreferLiveUpdatesExistingOffering(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Models["anthropic/claude-sonnet-4-6"] = catalog.ModelV1{
		ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic", Name: "Claude Sonnet",
	}
	dst.Offerings = []catalog.ModelOfferingV1{{
		ID:               "anthropic-direct:claude-sonnet-4-6",
		CanonicalModelID: "anthropic/claude-sonnet-4-6",
		DeploymentID:     "anthropic-direct",
		NativeModelID:    "claude-sonnet-4-6",
		Pricing:          catalog.PricingV1{Status: catalog.PricingUnknown},
	}}
	liveMeta, _ := json.Marshal(map[string]any{"mode": "live"})
	src := &catalog.CatalogV1{
		Models: map[string]catalog.ModelV1{
			"anthropic/claude-sonnet-4-6": {
				ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic", Name: "Claude Sonnet 4.6",
			},
		},
		Offerings: []catalog.ModelOfferingV1{{
			ID:               "anthropic-direct:claude-sonnet-4-6",
			CanonicalModelID: "anthropic/claude-sonnet-4-6",
			DeploymentID:     "anthropic-direct",
			NativeModelID:    "claude-sonnet-4-6",
			Capabilities: catalog.CapabilitySetV1{
				FunctionCalling:        "supported",
				ExplicitThinkingBudget: "supported",
			},
			Pricing: catalog.PricingV1{
				Status:     catalog.PricingKnown,
				Currency:   "USD",
				RatesPer1M: map[string]float64{"input_tokens": 3, "output_tokens": 15},
				Source:     "live",
			},
			LiveMetadata: liveMeta,
			Provenance:   &catalog.CatalogProvenanceV1{Source: "live", ObservedAt: time.Now().UTC()},
		}},
	}
	out := discover.MergeCatalogV1WithPolicy(&dst, src, discover.MergePolicy{
		PreferLiveProviders: []string{"anthropic"},
	})
	if len(out.Offerings) != 1 {
		t.Fatalf("offerings = %d", len(out.Offerings))
	}
	got := out.Offerings[0]
	if got.Pricing.Status != catalog.PricingKnown {
		t.Fatalf("pricing status = %q", got.Pricing.Status)
	}
	if got.Capabilities.FunctionCalling != "supported" {
		t.Fatalf("function calling = %q", got.Capabilities.FunctionCalling)
	}
	if string(got.LiveMetadata) == "" {
		t.Fatal("expected live metadata")
	}
}

func TestMergeCatalogV1WithPolicy_PreferLiveFullReplace(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Models["openrouter/model-a"] = catalog.ModelV1{
		ID: "openrouter/model-a", ProviderID: "openrouter", Name: "Model A (old)",
		ContextWindow: 100000, MaxOutput: 4096,
	}
	src := &catalog.CatalogV1{
		Models: map[string]catalog.ModelV1{
			"openrouter/model-a": {
				ID: "openrouter/model-a", ProviderID: "openrouter", Name: "Model A (live)",
				ContextWindow: 0, MaxOutput: 0,
			},
		},
	}
	out := discover.MergeCatalogV1WithPolicy(&dst, src, discover.MergePolicy{
		PreferLiveProviders: []string{"openrouter"},
	})
	got := out.Models["openrouter/model-a"]
	if got.Name != "Model A (live)" {
		t.Fatalf("name = %q (expected live replacement)", got.Name)
	}
	if got.ContextWindow != 0 {
		t.Fatalf("context = %d (expected 0 from live)", got.ContextWindow)
	}
	if got.MaxOutput != 0 {
		t.Fatalf("max_output = %d (expected 0 from live)", got.MaxOutput)
	}
}

func TestMergeCatalogV1WithPolicy_PreferLiveUnconditionalPricing(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Models["openrouter/model-a"] = catalog.ModelV1{
		ID: "openrouter/model-a", ProviderID: "openrouter",
	}
	dst.Offerings = []catalog.ModelOfferingV1{{
		ID: "openrouter:model-a", CanonicalModelID: "openrouter/model-a",
		DeploymentID: "openrouter", NativeModelID: "model-a",
		Pricing: catalog.PricingV1{
			Status: catalog.PricingKnown, Currency: "USD",
			RatesPer1M: map[string]float64{"input_tokens": 5, "output_tokens": 15},
		},
	}}
	src := &catalog.CatalogV1{
		Models: map[string]catalog.ModelV1{
			"openrouter/model-a": {ID: "openrouter/model-a", ProviderID: "openrouter"},
		},
		Offerings: []catalog.ModelOfferingV1{{
			ID: "openrouter:model-a", CanonicalModelID: "openrouter/model-a",
			DeploymentID: "openrouter", NativeModelID: "model-a",
			Pricing: catalog.PricingV1{
				Status: catalog.PricingKnown, Currency: "EUR",
				RatesPer1M: map[string]float64{"input_tokens": 3, "output_tokens": 10},
			},
		}},
	}
	out := discover.MergeCatalogV1WithPolicy(&dst, src, discover.MergePolicy{
		PreferLiveProviders: []string{"openrouter"},
	})
	if len(out.Offerings) != 1 {
		t.Fatalf("offerings = %d", len(out.Offerings))
	}
	got := out.Offerings[0]
	if got.Pricing.Currency != "EUR" {
		t.Fatalf("pricing currency = %q (expected live EUR)", got.Pricing.Currency)
	}
	if got.Pricing.RatesPer1M["input_tokens"] != 3 {
		t.Fatalf("input price = %f (expected live 3)", got.Pricing.RatesPer1M["input_tokens"])
	}
}

func TestMergeCatalogV1WithPolicy_PreferLiveZeroContextOverwrites(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Models["anthropic/claude-sonnet-4-6"] = catalog.ModelV1{
		ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic",
		ContextWindow: 200000, MaxOutput: 8192,
	}
	src := &catalog.CatalogV1{
		Models: map[string]catalog.ModelV1{
			"anthropic/claude-sonnet-4-6": {
				ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic",
				ContextWindow: 0, MaxOutput: 0,
			},
		},
	}
	out := discover.MergeCatalogV1WithPolicy(&dst, src, discover.MergePolicy{
		PreferLiveProviders: []string{"anthropic"},
	})
	got := out.Models["anthropic/claude-sonnet-4-6"]
	if got.ContextWindow != 0 {
		t.Fatalf("context = %d (expected 0 from prefer-live)", got.ContextWindow)
	}
	if got.MaxOutput != 0 {
		t.Fatalf("max_output = %d (expected 0 from prefer-live)", got.MaxOutput)
	}
}

func TestMergeCatalogV1WithPolicy_NonPreferLivePreservesExisting(t *testing.T) {
	dst := catalog.TestSeedCatalogV1()
	dst.Models["anthropic/claude-sonnet-4-6"] = catalog.ModelV1{
		ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic",
		ContextWindow: 200000, MaxOutput: 8192, Name: "Old",
	}
	src := &catalog.CatalogV1{
		Models: map[string]catalog.ModelV1{
			"anthropic/claude-sonnet-4-6": {
				ID: "anthropic/claude-sonnet-4-6", ProviderID: "anthropic",
				ContextWindow: 0, MaxOutput: 0, Name: "New",
			},
		},
	}
	out := discover.MergeCatalogV1WithPolicy(&dst, src, discover.MergePolicy{})
	got := out.Models["anthropic/claude-sonnet-4-6"]
	if got.ContextWindow != 200000 {
		t.Fatalf("context = %d (expected preserved without live policy)", got.ContextWindow)
	}
	if got.MaxOutput != 8192 {
		t.Fatalf("max_output = %d (expected preserved)", got.MaxOutput)
	}
}
