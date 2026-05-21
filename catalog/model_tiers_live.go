package catalog

import "github.com/GrayCodeAI/eyrie/catalog/registry"

func init() {
	stripLiveCatalogProvidersFromTierMatrices()
}

func stripLiveCatalogProvidersFromTierMatrices() {
	for _, spec := range registry.All() {
		if spec.ModelStrategy != registry.StrategyLiveOnly {
			continue
		}
		pid := spec.ProviderID
		delete(preferredKeys, pid)
		for key := range AllModelConfigs {
			delete(AllModelConfigs[key], pid)
		}
	}
}

func usesLiveCatalogOnly(provider string) bool {
	return IsLiveOnlyProvider(provider)
}
