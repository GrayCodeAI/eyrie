package legacy

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/GrayCodeAI/eyrie/catalog"
)

var defaultModelCatalog = catalog.ModelCatalog{
	UpdatedAt: "2026-04-09T00:00:00.000Z",
	Source:    "bootstrap",
	Providers: map[string][]catalog.ModelCatalogEntry{},
}

// DefaultModelCatalog returns the embedded default catalog.
//
// Deprecated: use catalog.LoadCatalogV1 instead.
func DefaultModelCatalog() catalog.ModelCatalog {
	return defaultModelCatalog
}

// LoadModelCatalogSync loads a catalog from a cache file, falling back to embedded.
//
// Deprecated: use catalog.LoadCatalogV1 instead.
func LoadModelCatalogSync(cachePath string) catalog.ModelCatalog {
	if cachePath != "" {
		data, err := os.ReadFile(cachePath)
		if err == nil {
			var cat catalog.ModelCatalog
			if json.Unmarshal(data, &cat) == nil && cat.Providers != nil {
				return cat
			}
		}
	}
	return DefaultModelCatalog()
}

// FetchModelCatalog fetches catalog from live provider APIs.
//
// Deprecated: use setup.DiscoverModelCatalog instead.
func FetchModelCatalog(cachePath string, env map[string]string) (catalog.ModelCatalog, error) {
	cat, _ := FetchModelCatalogWithEnrichment(cachePath, env)
	return cat, nil
}

// FetchModelCatalogWithEnrichment returns live provider catalog data and per-provider fetch status.
//
// Deprecated: use setup.DiscoverModelCatalog instead.
func FetchModelCatalogWithEnrichment(cachePath string, env map[string]string) (catalog.ModelCatalog, []catalog.LiveProviderEnrichment) {
	cat, enrichment := catalog.FetchLiveProviderCatalog(env)

	if cachePath != "" {
		data, err := json.MarshalIndent(cat, "", "  ")
		if err == nil {
			_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
			_ = os.WriteFile(cachePath, append(data, '\n'), 0o644)
		}
	}

	return cat, enrichment
}

// ModelsForProvider returns catalog entries for a given provider in a legacy ModelCatalog.
//
// Deprecated: use catalog.ModelEntriesForProvider with catalog.CompiledCatalogV1 instead.
func ModelsForProvider(cat *catalog.ModelCatalog, provider string) []catalog.ModelCatalogEntry {
	if cat == nil || cat.Providers == nil {
		return nil
	}
	return cat.Providers[provider]
}
