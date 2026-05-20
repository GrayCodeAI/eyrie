package setup

import (
	"context"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/discover"
)

// DiscoverModelCatalog refreshes the remote eyrie catalog and enriches it using the supplied API keys.
// Pass config.DiscoveryCredentials(ctx) or explicit keys; eyrie owns all model metadata.
func DiscoverModelCatalog(ctx context.Context, creds catalog.Credentials) (*catalog.RefreshResult, error) {
	cachePath := catalog.DefaultCachePath()
	refreshRemote := true
	if compiled, ok := catalog.LoadValidCatalogCache(cachePath); ok && compiled.Catalog != nil {
		if !compiled.Catalog.StaleAfter.IsZero() && time.Now().UTC().Before(compiled.Catalog.StaleAfter) {
			refreshRemote = false
		}
	}
	return discover.Run(ctx, discover.Options{
		LoadCatalogV1Options: catalog.LoadCatalogV1Options{
			CachePath:     cachePath,
			RefreshRemote: refreshRemote,
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
