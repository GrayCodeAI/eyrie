package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/runtime"
)

func TestListModels_RequiresProvider(t *testing.T) {
	_, err := runtime.ListModels(context.Background(), runtime.ListModelsOpts{})
	if err == nil {
		t.Fatal("expected error for empty provider")
	}
}

func TestFormatSetupError_Ollama(t *testing.T) {
	err := runtime.FormatSetupError("ollama", context.DeadlineExceeded)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListModels_CacheReadDoesNotRequireDiscover(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "model_catalog.json")
	now := time.Now().UTC().Truncate(time.Second)
	c := catalog.CatalogV1{
		SchemaVersion: catalog.CatalogV1SchemaVersion,
		GeneratedAt:   now,
		StaleAfter:    now.Add(time.Hour),
		Providers: map[string]catalog.ProviderV1{
			"openai": {ID: "openai", Name: "OpenAI"},
		},
		APIProtocols: map[string]catalog.APIProtocolV1{
			"openai-chat-completions": {ID: "openai-chat-completions", Name: "OpenAI Chat Completions"},
		},
		Deployments: map[string]catalog.DeploymentV1{
			"openai-direct": {
				ID: "openai-direct", Name: "OpenAI", ProviderID: "openai",
				APIProtocolID: "openai-chat-completions", AdapterConstructor: "openai",
				NativeModelIDSource: catalog.NativeModelIDCatalogKnown,
			},
		},
		Models: map[string]catalog.ModelV1{
			"openai/gpt-test": {ID: "openai/gpt-test", ProviderID: "openai", Name: "GPT Test"},
		},
		Offerings: []catalog.ModelOfferingV1{{
			ID: "openai-direct:gpt-test", CanonicalModelID: "openai/gpt-test",
			DeploymentID: "openai-direct", NativeModelID: "gpt-test",
			Pricing: catalog.PricingV1{Status: catalog.PricingUnknown},
		}},
	}
	if err := catalog.WriteCatalogV1Cache(cachePath, &c); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EYRIE_MODEL_CATALOG_PATH", cachePath)

	entries, err := runtime.ListModels(context.Background(), runtime.ListModelsOpts{
		ProviderID: "openai",
		Source:     runtime.ListSourceCache,
	})
	if err != nil {
		t.Fatalf("ListModels cache: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "openai/gpt-test" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestWriteCatalogV1Cache_AtomicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model_catalog.json")
	c := catalog.TestSeedCatalogV1()
	if err := catalog.WriteCatalogV1Cache(path, &c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("expected temp file removed after atomic write")
	}
}
