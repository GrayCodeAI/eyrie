package discover_test

import (
	"testing"

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
