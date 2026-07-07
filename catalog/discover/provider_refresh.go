package discover

import (
	"context"
	"encoding/json"
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
	var base *catalog.Catalog
	source := "cache"
	if compiled, err := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{
		CachePath: cachePath,
	}); err == nil && compiled != nil && compiled.Catalog != nil {
		base = compiled.Catalog
	} else if compiled, ok := catalog.LoadValidCatalogCache(cachePath); ok && compiled.Catalog != nil {
		base = compiled.Catalog
	} else {
		bootstrap := catalog.BootstrapCatalog()
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

	generatedAt := time.Now().UTC().Truncate(time.Second)
	cat := catalog.SeedCatalog()
	cat.SchemaVersion = catalog.CatalogSchemaVersion
	cat.GeneratedAt = generatedAt
	cat.StaleAfter = generatedAt.Add(catalog.LiveStaleDuration)
	catalog.EnsureDeploymentEnvFallbacks(&cat)
	catalog.EnsureCredentialRegistryInCatalog(&cat)
	cat.Provenance = &catalog.Provenance{Source: "live-providers", ObservedAt: generatedAt}

	provID := spec.ProviderID
	if _, ok := cat.Providers[provID]; !ok {
		cat.Providers[provID] = catalog.Provider{ID: provID, Name: provID}
	}
	depID := spec.DeploymentID
	if _, ok := cat.Deployments[depID]; !ok {
		cat.Deployments[depID] = catalog.Deployment{
			ID:                     depID,
			Name:                   provID,
			ProviderID:             provID,
			APIProtocolID:          "openai-chat-completions",
			AdapterConstructor:     "openai",
			NativeModelIDSource:    catalog.NativeModelIDDiscovered,
			ModelMappingsRequired:  false,
		}
	}

	for _, entry := range entries {
		entryID := entry.ID
		if entryID == "" {
			continue
		}
		name := entry.DisplayName
		if name == "" {
			name = entryID
		}
		canonicalID := entryID
		var liveMeta map[string]interface{}
		if json.Unmarshal(entry.RawJSON, &liveMeta) == nil {
			if inputPrice, ok := liveMeta["input_token_price_per_m"]; ok {
				if _, ok := inputPrice.(float64); ok {
					canonicalID = provID + "/" + entryID
				}
			}
		}
		cat.Models[canonicalID] = catalog.Model{
			ID:            canonicalID,
			ProviderID:    provID,
			Name:          name,
			ContextWindow:  entry.ContextWindow,
			MaxOutput:     entry.MaxOutput,
		}
		cat.Aliases[entryID] = canonicalID
		offeringID := depID + ":" + entryID
		cat.Offerings = append(cat.Offerings, catalog.ModelOffering{
			ID:               offeringID,
			CanonicalModelID: canonicalID,
			DeploymentID:     depID,
			NativeModelID:    entryID,
			Capabilities:     catalog.CapabilitySetFromEntry(entry),
			Pricing:          catalog.PricingFromEntry(entry),
			LiveMetadata:     entry.RawJSON,
		})
	}

	base = MergeCatalogWithPolicy(base, &cat, MergePolicy{
		PreferLive:                 true,
		PreferLiveProviders:        []string{catalog.CanonicalProviderID(spec.ProviderID)},
		ReplaceDeploymentOfferings: []string{spec.DeploymentID},
	})
	now := time.Now().UTC().Truncate(time.Second)
	base.GeneratedAt = now
	base.StaleAfter = now.Add(catalog.LiveStaleDuration)
	if base.Provenance == nil {
		base.Provenance = &catalog.Provenance{}
	}
	base.Provenance.ObservedAt = now
	source = appendSourceSuffix(source, "providers")

	if err := catalog.WriteCatalogCache(cachePath, base); err != nil {
		return nil, fmt.Errorf("catalog discover: write cache: %w", err)
	}
	compiled, err := catalog.CompileCatalog(base)
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
