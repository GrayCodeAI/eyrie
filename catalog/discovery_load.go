package catalog

import "context"

// LoadCatalogForDiscovery returns the cached catalog or bootstrap wiring (no network).
func LoadCatalogForDiscovery(ctx context.Context) (*CompiledCatalog, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	compiled, err := LoadCatalog(ctx, LoadCatalogOptions{
		CachePath: DefaultCachePath(),
	})
	if err == nil && compiled != nil {
		return compiled, nil
	}
	bootstrap := BootstrapCatalog()
	return CompileCatalog(&bootstrap)
}

// DiscoveryEnvKeyNames returns env var names used for credential discovery from the catalog.
func DiscoveryEnvKeyNames(ctx context.Context) []string {
	compiled, err := LoadCatalogForDiscovery(ctx)
	if err != nil || compiled == nil {
		return nil
	}
	return DiscoveryEnvKeysFromCatalog(compiled)
}
