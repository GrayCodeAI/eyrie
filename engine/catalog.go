package engine

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/discover"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
	"github.com/GrayCodeAI/eyrie/config"
)

// Catalog returns the current cached catalog through a stable snapshot.
func (e *Engine) Catalog(ctx context.Context) (CatalogSnapshot, error) {
	ctx = nonNilContext(ctx)
	compiled, err := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath, RequireCache: true})
	if err != nil {
		return CatalogSnapshot{}, &Error{Code: ErrorCatalogUnavailable, Operation: "catalog", Message: err.Error(), Cause: err}
	}
	snapshot := snapshotFromCompiled(compiled)
	snapshot.CachePath = e.catalogPath
	return snapshot, nil
}

// RefreshCatalog refreshes one provider when providerID is non-empty, or all
// configured providers otherwise. Credentials are read from this Engine's
// injected store rather than a package-global store.
func (e *Engine) RefreshCatalog(ctx context.Context, providerID string) (CatalogSnapshot, error) {
	ctx = nonNilContext(ctx)
	compiled, _ := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath})
	creds := catalog.Credentials{APIKeys: e.credentialEnv(ctx, compiled)}
	var (
		result *catalog.RefreshResult
		err    error
	)
	if providerID = strings.TrimSpace(providerID); providerID != "" {
		if _, ok := registry.SpecByProviderID(providerID); !ok {
			return CatalogSnapshot{}, &Error{Code: ErrorInvalidRequest, Operation: "refresh_catalog", Provider: providerID, Message: "eyrie engine: unknown provider"}
		}
		// The refresh shares one compiled-cache transaction so custom host paths
		// remain isolated. Credential presence limits live discovery to configured
		// providers; providerID is retained for host status/error attribution.
	}
	result, err = discover.Run(ctx, discover.Options{
		LoadCatalogOptions: catalog.LoadCatalogOptions{CachePath: e.catalogPath, RefreshRemote: true},
		Credentials:        creds,
	})
	if err != nil {
		return CatalogSnapshot{}, &Error{Code: ErrorCatalogUnavailable, Operation: "refresh_catalog", Provider: providerID, Message: err.Error(), Cause: err}
	}
	if result == nil || result.Compiled == nil {
		return CatalogSnapshot{}, &Error{Code: ErrorCatalogUnavailable, Operation: "refresh_catalog", Provider: providerID, Message: "eyrie engine: catalog refresh returned no compiled catalog"}
	}
	snapshot := snapshotFromCompiled(result.Compiled)
	snapshot.CachePath = e.catalogPath
	return snapshot, nil
}

// ApplyCredentials refreshes catalog state and persists sanitized deployment
// routing derived from the Engine's credential store. Secret fields are never
// written to provider.json.
func (e *Engine) ApplyCredentials(ctx context.Context, providerID string) (CatalogSnapshot, error) {
	ctx = nonNilContext(ctx)
	snapshot, err := e.RefreshCatalog(ctx, providerID)
	if err != nil {
		return CatalogSnapshot{}, err
	}
	compiled, err := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath, RequireCache: true})
	if err != nil {
		return CatalogSnapshot{}, &Error{Code: ErrorCatalogUnavailable, Operation: "apply_credentials", Provider: providerID, Message: err.Error(), Cause: err}
	}
	persisted := config.LoadProviderConfig(e.providerConfigPath)
	if persisted == nil {
		persisted = &config.ProviderConfig{}
	}
	deployments := buildDeployments(compiled, persisted.Deployments, e.credentialEnv(ctx, compiled))
	sanitized := make(map[string]config.DeploymentConfig, len(deployments))
	for id, deployment := range deployments {
		sanitized[id] = config.SanitizeDeploymentConfigForDisk(deployment)
	}
	persisted.ConfigVersion = 2
	persisted.Deployments = sanitized
	persisted.Routing = config.BuildRoutingPolicyFromDeployments(sanitized)
	if err := config.SaveProviderConfig(persisted, e.providerConfigPath); err != nil {
		return CatalogSnapshot{}, &Error{Code: ErrorInternal, Operation: "apply_credentials", Provider: providerID, Message: err.Error(), Cause: err}
	}
	return snapshot, nil
}

func snapshotFromCompiled(compiled *catalog.CompiledCatalog) CatalogSnapshot {
	snapshot := CatalogSnapshot{CachePath: catalog.DefaultCachePath(), LoadedAt: time.Now().UTC()}
	if compiled == nil {
		return snapshot
	}
	ids := make([]string, 0, len(compiled.ModelsByID))
	for id := range compiled.ModelsByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		model := compiled.ModelsByID[id]
		offering := firstOffering(compiled.OfferingsByCanonicalModel[id])
		inputPrice, outputPrice := 0.0, 0.0
		if offering.Pricing.RatesPer1M != nil {
			inputPrice = offering.Pricing.RatesPer1M["input_tokens"]
			outputPrice = offering.Pricing.RatesPer1M["output_tokens"]
		}
		snapshot.Models = append(snapshot.Models, Model{
			ID: id, DisplayName: catalog.DisplayModelLabel(id, model.Name),
			Description: model.Name,
			Owner:       catalog.DisplayModelOwner(model.ProviderID, id), ProviderID: model.ProviderID,
			GatewayID:     catalog.GatewayForModel(compiled, id),
			ContextWindow: model.ContextWindow, MaxOutputTokens: model.MaxOutput,
			InputPricePer1M: inputPrice, OutputPricePer1M: outputPrice,
			PriceKnown:   modelPriceKnown(id, model.Name, inputPrice, outputPrice, model.ContextWindow),
			Capabilities: capabilityNames(offering.Capabilities), Source: "catalog",
		})
	}
	if compiled.Catalog != nil && !compiled.Catalog.StaleAfter.IsZero() {
		snapshot.Stale = time.Now().UTC().After(compiled.Catalog.StaleAfter)
	}
	return snapshot
}

func firstOffering(offerings []catalog.ModelOffering) catalog.ModelOffering {
	if len(offerings) == 0 {
		return catalog.ModelOffering{}
	}
	ordered := append([]catalog.ModelOffering(nil), offerings...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].DeploymentID < ordered[j].DeploymentID })
	return ordered[0]
}

func capabilityNames(caps catalog.CapabilitySet) []string {
	var out []string
	if caps.FunctionCalling == catalog.CapabilitySupported {
		out = append(out, "tools")
	}
	if caps.ImageInput == catalog.CapabilitySupported {
		out = append(out, "vision")
	}
	if caps.StructuredOutput == catalog.CapabilitySupported {
		out = append(out, "structured_json")
	}
	if caps.ExplicitThinkingBudget == catalog.CapabilitySupported || caps.AdaptiveThinking == catalog.CapabilitySupported || caps.Effort == catalog.CapabilitySupported {
		out = append(out, "reasoning")
	}
	sort.Strings(out)
	return out
}
