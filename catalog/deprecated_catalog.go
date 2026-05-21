package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var defaultModelCatalog = ModelCatalog{
	UpdatedAt: "2026-04-09T00:00:00.000Z",
	Source:    "bootstrap",
	Providers: map[string][]ModelCatalogEntry{},
}

// DefaultModelCatalog returns the embedded default catalog.
//
// Deprecated: use LoadCatalogV1 instead.
func DefaultModelCatalog() ModelCatalog {
	return defaultModelCatalog
}

// LoadModelCatalogSync loads a catalog from a cache file, falling back to embedded.
//
// Deprecated: use LoadCatalogV1 instead.
func LoadModelCatalogSync(cachePath string) ModelCatalog {
	if cachePath != "" {
		data, err := os.ReadFile(cachePath)
		if err == nil {
			var cat ModelCatalog
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
func FetchModelCatalog(cachePath string, env map[string]string) (ModelCatalog, error) {
	cat, _ := FetchModelCatalogWithEnrichment(cachePath, env)
	return cat, nil
}

// FetchModelCatalogWithEnrichment returns live provider catalog data and per-provider fetch status.
//
// Deprecated: use setup.DiscoverModelCatalog instead.
func FetchModelCatalogWithEnrichment(cachePath string, env map[string]string) (ModelCatalog, []LiveProviderEnrichment) {
	cat, enrichment := FetchLiveProviderCatalog(env)

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
// Deprecated: use ModelEntriesForProvider with CompiledCatalogV1 instead.
func ModelsForProvider(cat *ModelCatalog, provider string) []ModelCatalogEntry {
	if cat == nil || cat.Providers == nil {
		return nil
	}
	return cat.Providers[provider]
}
