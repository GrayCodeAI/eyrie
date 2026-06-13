package catalog

// All providers are fully dynamic — model discovery comes from the live API catalog.

func usesLiveCatalogOnly(provider string) bool {
	return true
}

func hasStaticTierEntries(provider string) bool {
	return false
}
