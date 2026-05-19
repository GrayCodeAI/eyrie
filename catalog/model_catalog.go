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
func DefaultModelCatalog() ModelCatalog {
	return defaultModelCatalog
}

// LoadModelCatalogSync loads a catalog from a cache file, falling back to embedded.
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

// LiveProviderEnrichment records a live provider API fetch during catalog discovery.
type LiveProviderEnrichment struct {
	Provider   string `json:"provider"`
	ModelCount int    `json:"model_count"`
	Error      string `json:"error,omitempty"`
}

// FetchModelCatalog fetches catalog from remote APIs (OpenRouter, CanopyWave)
// and merges with embedded data. Writes result to cachePath if provided.
func FetchModelCatalog(cachePath string, env map[string]string) (ModelCatalog, error) {
	cat, _ := FetchModelCatalogWithEnrichment(cachePath, env)
	return cat, nil
}

// FetchModelCatalogWithEnrichment returns live provider catalog data and per-provider fetch status.
func FetchModelCatalogWithEnrichment(cachePath string, env map[string]string) (ModelCatalog, []LiveProviderEnrichment) {
	cat, enrichment := fetchLiveProviderCatalog(env)

	if cachePath != "" {
		data, err := json.MarshalIndent(cat, "", "  ")
		if err == nil {
			_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
			_ = os.WriteFile(cachePath, append(data, '\n'), 0o644)
		}
	}

	return cat, enrichment
}

// ModelsForProvider returns catalog entries for a given provider.
func ModelsForProvider(catalog *ModelCatalog, provider string) []ModelCatalogEntry {
	if catalog == nil || catalog.Providers == nil {
		return nil
	}
	return catalog.Providers[provider]
}
