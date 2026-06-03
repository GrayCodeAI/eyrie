package catalog

func usesLiveCatalogOnly(provider string) bool {
	return IsLiveOnlyProvider(provider)
}