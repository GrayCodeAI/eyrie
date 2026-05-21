package catalog_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/live"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestAllProviders_LiveFetchParity(t *testing.T) {
	specs := registry.All()
	if len(specs) != 9 {
		t.Fatalf("expected 9 providers, got %d", len(specs))
	}
	for _, spec := range specs {
		t.Run(spec.ProviderID, func(t *testing.T) {
			if spec.LiveFetcherKey == "" {
				t.Fatal("missing LiveFetcherKey")
			}
			if spec.LiveCatalogKey == "" {
				t.Fatal("missing LiveCatalogKey")
			}
			if _, ok := live.Registry[spec.LiveFetcherKey]; !ok {
				t.Fatalf("live.Registry missing fetcher %q", spec.LiveFetcherKey)
			}
			if !spec.PreferLiveMerge {
				t.Fatal("PreferLiveMerge should be true")
			}
			dep := catalog.DeploymentIDForLiveCatalogKey(spec.LiveCatalogKey)
			if dep != spec.DeploymentID {
				t.Fatalf("DeploymentIDForLiveCatalogKey = %q, want %q", dep, spec.DeploymentID)
			}
		})
	}
}

func TestAllProviders_LiveOnlySkipHardcodedDefaults(t *testing.T) {
	empty := &catalog.ModelCatalog{}
	for _, spec := range registry.All() {
		if spec.ModelStrategy != registry.StrategyLiveOnly {
			continue
		}
		got := catalog.GetProviderDefaultModel(spec.ProviderID, empty)
		if got != "" {
			t.Errorf("%s: expected empty default without catalog, got %q", spec.ProviderID, got)
		}
	}
}

func TestAllProviders_FirstModelFromCompiledCache(t *testing.T) {
	base := catalog.TestSeedCatalogV1()
	for _, spec := range registry.All() {
		native := "live-" + spec.ProviderID + "-model"
		canonical := spec.ProviderID + "/" + native
		if spec.ProviderID == "gemini" {
			canonical = "google/" + native
		}
		base.Models[canonical] = catalog.ModelV1{
			ID: canonical, ProviderID: catalog.CanonicalProviderID(spec.ProviderID), Name: native,
		}
	}
	compiled, err := catalog.CompileCatalogV1(&base)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range registry.All() {
		id := catalog.FirstModelForProvider(compiled, spec.ProviderID)
		if id == "" {
			t.Errorf("%s: FirstModelForProvider returned empty", spec.ProviderID)
		}
	}
}
