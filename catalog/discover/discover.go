package discover

import (
	"context"
	"fmt"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	eyriecfg "github.com/GrayCodeAI/eyrie/config"
)

// Options configures catalog discovery: published catalog + live provider APIs via API keys.
type Options struct {
	catalog.LoadCatalogV1Options
	Credentials catalog.Credentials
}

// Run loads the deployment-aware catalog, optionally refreshes the remote catalog,
// merges live provider model lists when API keys are supplied, writes the cache,
// and returns the compiled catalog.
func Run(ctx context.Context, opts Options) (*catalog.RefreshResult, error) {
	runMu.Lock()
	defer runMu.Unlock()
	return run(ctx, opts)
}

func run(ctx context.Context, opts Options) (*catalog.RefreshResult, error) {
	if opts.CachePath == "" {
		opts.CachePath = catalog.DefaultCachePath()
	}

	var base *catalog.CatalogV1
	source := "embedded"
	refreshed := false
	remoteRefreshed := false
	var liveProviders []catalog.LiveProviderEnrichment

	if opts.RefreshRemote {
		loadOpts := opts.LoadCatalogV1Options
		loadOpts.RemoteURL = catalog.ResolvedRemoteCatalogURL(opts.RemoteURL)
		remote, err := catalog.FetchRemoteCatalogV1(ctx, loadOpts)
		if err != nil {
			if compiled, ok := catalog.LoadValidCatalogCache(opts.CachePath); ok && compiled.Catalog != nil {
				base = compiled.Catalog
				source = "cache-fallback"
			} else {
				bootstrap := catalog.BootstrapCatalogV1()
				base = &bootstrap
				source = catalog.BootstrapSource()
			}
		} else {
			base = remote
			source = "remote"
			refreshed = true
			remoteRefreshed = true
			opts.RemoteURL = loadOpts.RemoteURL
		}
	} else {
		compiled, err := catalog.LoadCatalogV1(ctx, opts.LoadCatalogV1Options)
		if err != nil {
			return nil, fmt.Errorf("catalog discover: load: %w", err)
		}
		base = compiled.Catalog
		if base != nil && base.Provenance != nil && base.Provenance.Source != "" {
			source = base.Provenance.Source
		}
	}

	if base == nil {
		bootstrap := catalog.BootstrapCatalogV1()
		base = &bootstrap
		source = catalog.BootstrapSource()
	}
	catalog.EnsureDeploymentEnvFallbacks(base)
	catalog.EnsureCredentialRegistryInCatalog(base)

	env := opts.Credentials.Env()
	if len(env) == 0 {
		env = eyriecfg.DiscoveryCredentials(ctx).Env()
	}
	if len(env) > 0 {
		legacy, enrichment := catalog.FetchLiveProviderCatalog(env)
		liveProviders = enrichment
		if len(legacy.Providers) > 0 {
			enriched := catalog.CatalogV1FromLegacy(legacy)
			var replaceDeps []string
			for _, item := range enrichment {
				if item.Error != "" || item.ModelCount <= 0 {
					continue
				}
				if dep := catalog.DeploymentIDForLiveCatalogKey(item.Provider); dep != "" {
					replaceDeps = append(replaceDeps, dep)
				}
			}
			base = MergeCatalogV1WithPolicy(base, &enriched, MergePolicy{
				PreferLive:                 true,
				ReplaceDeploymentOfferings: replaceDeps,
			})
			now := time.Now().UTC().Truncate(time.Second)
			base.GeneratedAt = now
			base.StaleAfter = now.Add(catalog.LiveCatalogStaleDuration)
			if base.Provenance == nil {
				base.Provenance = &catalog.CatalogProvenanceV1{}
			}
			base.Provenance.ObservedAt = now
		}
		if source == "embedded" || source == catalog.BootstrapSource() || source == "cache-fallback" {
			if source == "cache-fallback" {
				source = "cache-fallback+providers"
			} else {
				source = "providers"
			}
		} else {
			source += "+providers"
		}
	}

	if err := catalog.WriteCatalogV1Cache(opts.CachePath, base); err != nil {
		return nil, fmt.Errorf("catalog discover: write cache: %w", err)
	}
	compiled, err := catalog.CompileCatalogV1(base)
	if err != nil {
		return nil, fmt.Errorf("catalog discover: compile: %w", err)
	}
	if len(compiled.ModelsByID) == 0 {
		return nil, fmt.Errorf("catalog discover: no models available (remote fetch and live APIs returned nothing; check network and API keys)")
	}
	return &catalog.RefreshResult{
		Compiled:        compiled,
		CachePath:       opts.CachePath,
		Source:          source,
		RemoteURL:       opts.RemoteURL,
		Refreshed:       refreshed || len(liveProviders) > 0,
		RemoteRefreshed: remoteRefreshed,
		LiveProviders:   liveProviders,
		StaleAfter:      base.StaleAfter,
	}, nil
}
