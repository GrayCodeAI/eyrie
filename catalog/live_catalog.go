package catalog

import (
	"sort"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// LiveCatalogStaleDuration is how long a cache remains fresh after live provider APIs were merged.
const LiveCatalogStaleDuration = 24 * time.Hour

// IsLiveOnlyProvider reports providers whose models come from live list APIs (not static tiers).
func IsLiveOnlyProvider(providerID string) bool {
	spec, ok := registry.SpecByProviderID(normalizeLiveProviderID(providerID))
	if !ok {
		return false
	}
	return spec.ModelStrategy == registry.StrategyLiveOnly
}

// DeploymentIDForLiveCatalogKey maps a live fetch catalog key to a deployment ID.
func DeploymentIDForLiveCatalogKey(catalogKey string) string {
	for _, spec := range registry.All() {
		if spec.LiveCatalogKey == catalogKey {
			return spec.DeploymentID
		}
	}
	return ""
}

// FirstModelForProvider returns the first canonical model ID for a provider from compiled catalog.
func FirstModelForProvider(compiled *CompiledCatalogV1, providerID string) string {
	if compiled == nil {
		return ""
	}
	providerID = normalizeLiveProviderID(providerID)
	var ids []string
	for id, model := range compiled.ModelsByID {
		if normalizeLiveProviderID(model.ProviderID) == providerID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func normalizeLiveProviderID(providerID string) string {
	return CanonicalProviderID(providerID)
}
