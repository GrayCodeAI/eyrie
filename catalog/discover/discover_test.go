package discover_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/discover"
)

func TestDiscoverCatalog_MergesProviderModelsWithAPIKey(t *testing.T) {
	orServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-or-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"vendor/special-model","context_length":32000,"pricing":{"prompt":"0.000001","completion":"0.000002"}}]}`))
	}))
	defer orServer.Close()

	cachePath := filepath.Join(t.TempDir(), "model_catalog.json")
	base := catalog.TestSeedCatalogV1()
	if err := catalog.WriteCatalogV1Cache(cachePath, &base); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	result, err := discover.Run(context.Background(), discover.Options{
		LoadCatalogV1Options: catalog.LoadCatalogV1Options{
			CachePath:     cachePath,
			RefreshRemote: false,
		},
		Credentials: catalog.Credentials{APIKeys: map[string]string{
			"OPENROUTER_API_KEY":  "test-or-key",
			"OPENROUTER_BASE_URL": orServer.URL,
		}},
	})
	if err != nil {
		t.Fatalf("discover.Run: %v", err)
	}
	if result == nil || result.Compiled == nil {
		t.Fatal("expected compiled catalog")
	}
	found := false
	for id := range result.Compiled.ModelsByID {
		if id != "" {
			found = true
			break
		}
	}
	if !found && len(result.Compiled.OfferingsByID) == 0 {
		t.Fatal("expected models or offerings after discover")
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache not written: %v", err)
	}
	if len(result.LiveProviders) != 9 {
		t.Fatalf("LiveProviders: got %d want 9", len(result.LiveProviders))
	}
	var openrouter *catalog.LiveProviderEnrichment
	for i := range result.LiveProviders {
		if result.LiveProviders[i].Provider == "openrouter" {
			openrouter = &result.LiveProviders[i]
			break
		}
	}
	if openrouter == nil || openrouter.ModelCount < 1 {
		t.Fatalf("openrouter enrichment: %+v", openrouter)
	}
}
