package discover

import (
	"context"
	"fmt"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/live"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
	eyriecfg "github.com/GrayCodeAI/eyrie/config"
)

// RefreshProvider fetches live models for one provider, merges into the catalog cache,
// and returns the compiled catalog. Used after the user saves an API key in /config.
func RefreshProvider(ctx context.Context, providerID string, creds catalog.Credentials) (*catalog.RefreshResult, error) {
	runMu.Lock()
	defer runMu.Unlock()
	return refreshProvider(ctx, providerID, creds)
}

func refreshProvider(ctx context.Context, providerID string, creds catalog.Credentials) (*catalog.RefreshResult, error) {
	spec, ok := registry.SpecByProviderID(providerID)
	if !ok {
		return nil, fmt.Errorf("catalog discover: unknown provider %q", providerID)
	}
	if spec.LiveFetcherKey == "" {
		return nil, fmt.Errorf("catalog discover: provider %q has no live model list API", providerID)
	}

	cachePath := catalog.DefaultCachePath()
	var base *catalog.CatalogV1
	source := "cache"
	if compiled, err := catalog.LoadCatalogV1(ctx, catalog.LoadCatalogV1Options{
		CachePath: cachePath,
	}); err == nil && compiled != nil && compiled.Catalog != nil {
		base = compiled.Catalog
	} else if compiled, ok := catalog.LoadValidCatalogCache(cachePath); ok && compiled.Catalog != nil {
		base = compiled.Catalog
	} else {
		bootstrap := catalog.BootstrapCatalogV1()
		base = &bootstrap
		source = catalog.BootstrapSource()
	}
	catalog.EnsureDeploymentEnvFallbacks(base)
	catalog.EnsureCredentialRegistryInCatalog(base)

	env := creds.Env()
	if len(env) == 0 {
		env = eyriecfg.DiscoveryCredentials(ctx).Env()
	}
	if !registry.CredentialPresent(spec, env) {
		return nil, fmt.Errorf("catalog discover: no credentials for provider %q", providerID)
	}

	entries, err := live.Fetch(spec.LiveFetcherKey, env)
	if err != nil {
		return nil, fmt.Errorf("catalog discover: live fetch %q: %w", providerID, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("catalog discover: live API returned no models for %q", providerID)
	}

	legacy := catalog.ModelCatalog{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    "live-providers",
		Providers: map[string][]catalog.ModelCatalogEntry{
			spec.LiveCatalogKey: catalog.LiveEntriesToCatalog(entries),
		},
	}
	enriched := catalog.CatalogV1FromLegacy(legacy)
	base = MergeCatalogV1WithPolicy(base, &enriched, MergePolicy{
		PreferLive:                 true,
		ReplaceDeploymentOfferings: []string{spec.DeploymentID},
	})
	now := time.Now().UTC().Truncate(time.Second)
	base.GeneratedAt = now
	base.StaleAfter = now.Add(catalog.LiveCatalogStaleDuration)
	if base.Provenance == nil {
		base.Provenance = &catalog.CatalogProvenanceV1{}
	}
	base.Provenance.ObservedAt = now
	if source == catalog.BootstrapSource() {
		source = "providers"
	} else {
		source += "+providers"
	}

	if err := catalog.WriteCatalogV1Cache(cachePath, base); err != nil {
		return nil, fmt.Errorf("catalog discover: write cache: %w", err)
	}
	compiled, err := catalog.CompileCatalogV1(base)
	if err != nil {
		return nil, fmt.Errorf("catalog discover: compile: %w", err)
	}
	return &catalog.RefreshResult{
		Compiled:  compiled,
		CachePath: cachePath,
		Source:    source,
		Refreshed: true,
		LiveProviders: []catalog.LiveProviderEnrichment{{
			Provider:   spec.LiveCatalogKey,
			ModelCount: len(entries),
		}},
		StaleAfter: base.StaleAfter,
	}, nil
}
