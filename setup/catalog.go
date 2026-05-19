package setup

import (
	"context"

	"github.com/GrayCodeAI/eyrie/catalog"
)

// DiscoverModelCatalog refreshes the remote eyrie catalog and enriches it using the supplied API keys.
// Pass config.DiscoveryCredentialsFromOS() or explicit keys; eyrie owns all model metadata.
func DiscoverModelCatalog(ctx context.Context, creds catalog.Credentials) (*catalog.RefreshResult, error) {
	return catalog.DiscoverCatalog(ctx, catalog.DiscoverOptions{
		LoadCatalogV1Options: catalog.LoadCatalogV1Options{
			CachePath:     catalog.DefaultCachePath(),
			RefreshRemote: true,
		},
		Credentials: creds,
	})
}

// LoadCompiledCatalog returns the compiled catalog from cache/embedded data without network refresh.
func LoadCompiledCatalog(ctx context.Context) (*catalog.CompiledCatalogV1, error) {
	return catalog.LoadCatalogV1(ctx, catalog.LoadCatalogV1Options{
		CachePath:    catalog.DefaultCachePath(),
		RequireCache: true,
	})
}
