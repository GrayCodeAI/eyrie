package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var defaultModelCatalog = ModelCatalog{
	UpdatedAt: "2026-04-09T00:00:00.000Z",
	Source:    "embedded",
	Providers: DefaultProviderCatalogs(),
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

// FetchModelCatalog fetches catalog from remote APIs (OpenRouter, CanopyWave)
// and merges with embedded data. Writes result to cachePath if provided.
func FetchModelCatalog(cachePath string, env map[string]string) (ModelCatalog, error) {
	cat := ModelCatalog{
		UpdatedAt: defaultModelCatalog.UpdatedAt,
		Source:    defaultModelCatalog.Source,
		Providers: make(map[string][]ModelCatalogEntry),
	}
	for k, v := range defaultModelCatalog.Providers {
		cat.Providers[k] = v
	}

	// Fetch OpenRouter models
	orModels, err := fetchOpenRouterCatalog(env)
	if err == nil && len(orModels) > 0 {
		cat.Providers["openrouter"] = orModels
	}

	// Fetch CanopyWave models
	cwModels, err := fetchCanopyWaveCatalog(env)
	if err == nil && len(cwModels) > 0 {
		cat.Providers["canopywave"] = cwModels
	}

	if cachePath != "" {
		data, err := json.MarshalIndent(cat, "", "  ")
		if err == nil {
			_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
			_ = os.WriteFile(cachePath, append(data, '\n'), 0o644)
		}
	}

	return cat, nil
}

// ModelsForProvider returns catalog entries for a given provider.
func ModelsForProvider(catalog *ModelCatalog, provider string) []ModelCatalogEntry {
	if catalog == nil || catalog.Providers == nil {
		return nil
	}
	return catalog.Providers[provider]
}
