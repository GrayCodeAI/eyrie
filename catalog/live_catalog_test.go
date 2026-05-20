package catalog_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
)

func TestIsLiveOnlyProvider(t *testing.T) {
	if !catalog.IsLiveOnlyProvider("canopywave") {
		t.Fatal("canopywave should be live-only")
	}
	if !catalog.IsLiveOnlyProvider("z-ai") {
		t.Fatal("z-ai should be live-only")
	}
	if catalog.IsLiveOnlyProvider("anthropic") {
		t.Fatal("anthropic should not be live-only")
	}
}

func TestFirstModelForProvider(t *testing.T) {
	c := catalog.TestSeedCatalogV1()
	c.Models["z-ai/glm-5.1"] = catalog.ModelV1{ID: "z-ai/glm-5.1", ProviderID: "z-ai", Name: "GLM-5.1"}
	compiled, err := catalog.CompileCatalogV1(&c)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.FirstModelForProvider(compiled, "z-ai"); got != "z-ai/glm-5.1" {
		t.Fatalf("FirstModelForProvider = %q", got)
	}
}

func TestGetProviderDefaultModel_LiveOnlySkipsHardcoded(t *testing.T) {
	if got := catalog.GetProviderDefaultModel("canopywave", &catalog.ModelCatalog{}); got != "" {
		t.Fatalf("expected empty default without catalog, got %q", got)
	}
}
