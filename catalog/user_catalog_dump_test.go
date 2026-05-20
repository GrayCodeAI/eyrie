package catalog_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestUserCatalog_GatewayCountsMatchDeploymentOfferings(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	path := filepath.Join(home, ".eyrie", "model_catalog.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip("no user catalog")
	}
	compiled, err := catalog.LoadCatalogV1(context.Background(), catalog.LoadCatalogV1Options{
		CachePath:    path,
		RequireCache: true,
	})
	if err != nil {
		t.Skip(fmt.Sprintf("no user catalog: %v", err))
	}
	for _, gw := range []string{"openrouter", "canopywave", "gemini"} {
		spec, ok := registry.SpecByProviderID(gw)
		if !ok {
			t.Fatalf("missing spec for %q", gw)
		}
		entries := catalog.ModelEntriesForProvider(compiled, gw)
		raw := len(compiled.OfferingsByDeployment[spec.DeploymentID])
		if len(entries) != raw {
			t.Fatalf("%s: entries=%d raw deployment offerings=%d", gw, len(entries), raw)
		}
	}
}
