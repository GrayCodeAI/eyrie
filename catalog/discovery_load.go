package catalog

import "context"

// LoadCatalogForDiscovery returns the cached catalog or bootstrap wiring (no network).
func LoadCatalogForDiscovery(ctx context.Context) (*CompiledCatalogV1, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	compiled, err := LoadCatalogV1(ctx, LoadCatalogV1Options{
		CachePath: DefaultCachePath(),
	})
	if err == nil && compiled != nil {
		return compiled, nil
	}
	bootstrap := BootstrapCatalogV1()
	return CompileCatalogV1(&bootstrap)
}

// DiscoveryEnvKeyNames returns env var names used for credential discovery from the catalog.
func DiscoveryEnvKeyNames(ctx context.Context) []string {
	compiled, err := LoadCatalogForDiscovery(ctx)
	if err != nil || compiled == nil {
		return nil
	}
	return DiscoveryEnvKeysFromCatalog(compiled)
}
