package catalog_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/live"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestAllProviders_LiveFetchParity(t *testing.T) {
	specs := registry.All()
	if len(specs) != 15 {
		t.Fatalf("expected 15 providers, got %d", len(specs))
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
		owner := catalog.CanonicalProviderID(spec.ProviderID)
		canonical := owner + "/" + native
		if spec.ProviderID == "gemini" || spec.ProviderID == "vertex" {
			owner = "google"
			canonical = "google/" + native
		} else if spec.ProviderID == "azure" {
			owner = "openai"
			canonical = "openai/" + native
		} else if spec.ProviderID == "bedrock" {
			owner = "anthropic"
			canonical = "anthropic/" + native
		}
		base.Models[canonical] = catalog.ModelV1{
			ID: canonical, ProviderID: owner, Name: native,
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
