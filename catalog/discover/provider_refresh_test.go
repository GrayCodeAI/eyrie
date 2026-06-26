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

func TestRefreshProvider_MergesLiveModelsIntoCache(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("../live/testdata/canopywave_models.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cachePath := filepath.Join(t.TempDir(), "model_catalog.json")
	base := catalog.TestSeedCatalogV1()
	if err := catalog.WriteCatalogV1Cache(cachePath, &base); err != nil {
		t.Fatal(err)
	}

	result, err := discover.RefreshProvider(context.Background(), "canopywave", catalog.Credentials{APIKeys: map[string]string{
		"CANOPYWAVE_API_KEY":  "test-key-12345678",
		"CANOPYWAVE_BASE_URL": srv.URL,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Compiled == nil {
		t.Fatal("expected compiled catalog")
	}
	entries := catalog.ModelEntriesForProvider(result.Compiled, "canopywave")
	if len(entries) != 2 {
		t.Fatalf("expected 2 canopywave models, got %d", len(entries))
	}
	if len(entries[0].LiveMetadata) == 0 {
		t.Fatal("expected live metadata on cached offering")
	}
}
