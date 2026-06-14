package catalog

import (
	"fmt"
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
		catalogKey := registry.LiveCatalogKeyForFetcher(fetcherKey)
		start := time.Now()
		if !registry.CredentialPresent(spec, env) {
			reason := "skipped (no API key)"
			if !spec.RequiresKey {
				reason = "skipped (no base URL)"
			}
			enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, Error: reason, DurationMs: time.Since(start).Milliseconds()})
			continue
		}
		models, err := live.Fetch(fetcherKey, env)
		elapsed := time.Since(start)
		duration := elapsed.Milliseconds()
		if err != nil {
			enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, Error: err.Error(), DurationMs: duration})
			continue
		}
		if len(models) == 0 {
			enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, Error: "no models returned", DurationMs: duration})
			continue
		}
		cat.Providers[catalogKey] = LiveEntriesToCatalog(models)
		enrichment = append(enrichment, LiveProviderEnrichment{Provider: catalogKey, ModelCount: len(models), DurationMs: duration})
	}
	return cat, enrichment
}

// FetchLiveModelEntriesForProvider lists models from one provider's live API with full JSON metadata.
func FetchLiveModelEntriesForProvider(env map[string]string, providerID string) ([]ModelCatalogEntry, error) {
	spec, ok := registry.SpecByProviderID(providerID)
	if !ok {
		return nil, fmt.Errorf("catalog: unknown provider %q", providerID)
	}
	if spec.LiveFetcherKey == "" {
		return nil, fmt.Errorf("catalog: provider %q has no live model list API", providerID)
	}
	if !registry.CredentialPresent(spec, env) {
		if !spec.RequiresKey {
			return nil, fmt.Errorf("catalog: set %s for %s", spec.CredentialEnv, providerID)
		}
		return nil, fmt.Errorf("catalog: set %s for %s", spec.CredentialEnv, providerID)
	}
	entries, err := live.Fetch(spec.LiveFetcherKey, env)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("catalog: live API returned no models for %q", providerID)
	}
	return LiveEntriesToCatalog(entries), nil
}

// LiveDiscoverableDeploymentIDs returns provider keys with live model-list APIs.
func LiveDiscoverableDeploymentIDs() []string {
	return registry.LiveFetcherKeys()
}
