package catalog

import (
	"time"

	"github.com/GrayCodeAI/eyrie/catalog/live"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// FetchLiveProviderCatalog enriches catalog from live provider list APIs.
func FetchLiveProviderCatalog(env map[string]string) (ModelCatalog, []LiveProviderEnrichment) {
	cat := ModelCatalog{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    "live-providers",
		Providers: make(map[string][]ModelCatalogEntry),
	}
	var enrichment []LiveProviderEnrichment
	for _, fetcherKey := range registry.LiveFetcherKeys() {
		spec, ok := registry.SpecForLiveFetcher(fetcherKey)
		if !ok {
			continue
		}
		if !registry.CredentialPresent(spec, env) {
			continue
		}
		catalogKey := registry.LiveCatalogKeyForFetcher(fetcherKey)
		models, err := live.Fetch(fetcherKey, env)
		if err != nil {
			enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, Error: err.Error()})
			continue
		}
		if len(models) == 0 {
			if spec.ModelStrategy == registry.StrategyLiveOnly {
				enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, Error: "no models returned"})
			}
			continue
		}
		cat.Providers[catalogKey] = LiveEntriesToCatalog(models)
		enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, ModelCount: len(models)})
	}
	return cat, enrichment
}

// LiveDiscoverableDeploymentIDs returns provider keys with live model-list APIs.
func LiveDiscoverableDeploymentIDs() []string {
	return registry.LiveFetcherKeys()
}
