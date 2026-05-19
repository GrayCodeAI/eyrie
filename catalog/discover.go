package catalog

import (
	"context"
	"fmt"
)

// DiscoverOptions configures catalog discovery: published catalog (langdag.com by default) + live provider APIs via API keys.
type DiscoverOptions struct {
	LoadCatalogV1Options
	Credentials Credentials
}

// DiscoverCatalog loads the deployment-aware catalog, optionally refreshes the remote catalog,
// merges live provider model lists when API keys are supplied, writes the cache,
// and returns the compiled catalog. Hawk should call this instead of embedding model data.
func DiscoverCatalog(ctx context.Context, opts DiscoverOptions) (*RefreshResult, error) {
	if opts.CachePath == "" {
		opts.CachePath = DefaultCachePath()
	}

	var base *CatalogV1
	source := "embedded"
	refreshed := false
	remoteRefreshed := false
	var liveProviders []LiveProviderEnrichment

	if opts.RefreshRemote {
		loadOpts := opts.LoadCatalogV1Options
		loadOpts.RemoteURL = ResolvedRemoteCatalogURL(opts.RemoteURL)
		remote, err := FetchRemoteCatalogV1(ctx, loadOpts)
		if err != nil {
			return nil, fmt.Errorf("catalog discover: remote: %w", err)
		}
		base = remote
		source = "remote"
		refreshed = true
		remoteRefreshed = true
		opts.RemoteURL = loadOpts.RemoteURL
	} else {
		compiled, err := LoadCatalogV1(ctx, opts.LoadCatalogV1Options)
		if err != nil {
			return nil, fmt.Errorf("catalog discover: load: %w", err)
		}
		base = compiled.Catalog
		if base != nil && base.Provenance != nil && base.Provenance.Source != "" {
			source = base.Provenance.Source
		}
	}

	if base == nil {
		bootstrap := BootstrapCatalogV1()
		base = &bootstrap
		source = bootstrapSource
	}
	EnsureDeploymentEnvFallbacks(base)

	env := opts.Credentials.Env()
	if len(env) == 0 {
		compiledSeed, err := CompileCatalogV1(base)
		if err == nil {
			env = CredentialsFromOSEnv(compiledSeed).Env()
		}
	}
	if len(env) > 0 {
		legacy, enrichment := fetchLiveProviderCatalog(env)
		liveProviders = enrichment
		if len(legacy.Providers) > 0 {
			enriched := CatalogV1FromLegacy(legacy)
			base = MergeCatalogV1(base, &enriched)
		}
		if source == "embedded" {
			source = "providers"
		} else {
			source = source + "+providers"
		}
	}

	if err := WriteCatalogV1Cache(opts.CachePath, base); err != nil {
		return nil, fmt.Errorf("catalog discover: write cache: %w", err)
	}
	compiled, err := CompileCatalogV1(base)
	if err != nil {
		return nil, fmt.Errorf("catalog discover: compile: %w", err)
	}
	if len(compiled.ModelsByID) == 0 {
		return nil, fmt.Errorf("catalog discover: no models available (remote fetch and live APIs returned nothing; check network and API keys)")
	}
	return &RefreshResult{
		Compiled:         compiled,
		CachePath:        opts.CachePath,
		Source:           source,
		RemoteURL:        opts.RemoteURL,
		Refreshed:        refreshed || len(liveProviders) > 0,
		RemoteRefreshed: remoteRefreshed,
		LiveProviders:    liveProviders,
		StaleAfter:       base.StaleAfter,
	}, nil
}
