package engine

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/runtime"
)

// NormalizeProviderID resolves provider aliases to Eyrie's runtime identifier.
func NormalizeProviderID(providerID string) string {
	return runtime.NormalizeProviderID(providerID)
}

// EffectiveSelection resolves persisted state plus optional host overrides
// using this Engine's injected catalog, provider config, and credential store.
func (e *Engine) EffectiveSelection(ctx context.Context, opts SelectionOptions) Selection {
	ctx = nonNilContext(ctx)
	active := e.ActiveSelection(ctx)
	provider := NormalizeProviderID(active.Provider)
	model := strings.TrimSpace(active.Model)
	if override := NormalizeProviderID(opts.ProviderOverride); override != "" {
		provider = override
	}
	if override := strings.TrimSpace(opts.ModelOverride); override != "" {
		model = override
	}

	compiled, _ := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath})
	if provider == "" && model != "" && compiled != nil {
		provider = NormalizeProviderID(catalog.GatewayForModel(compiled, model))
		if provider == "" {
			provider = NormalizeProviderID(catalog.ProviderForModel(compiled, model))
		}
	}

	gateways := e.Gateways(ctx)
	hasConfigured := false
	configured := make(map[string]bool, len(gateways))
	for _, gateway := range gateways {
		ready := gateway.DeploymentConfigured
		configured[NormalizeProviderID(gateway.ID)] = ready
		if ready {
			hasConfigured = true
			if provider == "" {
				provider = NormalizeProviderID(gateway.ID)
			}
		}
	}
	if provider != "" && hasConfigured && !configured[provider] {
		for _, gateway := range gateways {
			if configured[NormalizeProviderID(gateway.ID)] {
				provider = NormalizeProviderID(gateway.ID)
				model = ""
				break
			}
		}
	}
	if provider != "" && model == "" {
		if models, err := e.ListModels(ctx, provider, false); err == nil && len(models) > 0 {
			model = models[0].ID
		}
	}
	if compiled != nil && model != "" {
		if canonical, ok := compiled.CanonicalModelForAliasOrID(model); ok {
			model = canonical
		}
	}

	routing := active.DeploymentRouting
	if opts.DeploymentRoutingOverride != nil {
		routing = *opts.DeploymentRoutingOverride
	}
	return Selection{
		Provider: provider, Model: model,
		HasConfiguredDeployment: hasConfigured,
		DeploymentRouting:       routing,
	}
}

// ActiveSelection reads the persisted host selection from this Engine's state.
func (e *Engine) ActiveSelection(ctx context.Context) Route {
	_ = nonNilContext(ctx)
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	return Route{
		Provider:          NormalizeProviderID(config.ActiveProvider(cfg)),
		Model:             config.ActiveModel(cfg),
		DeploymentRouting: cfg != nil && (cfg.ConfigVersion >= 2 || len(cfg.Deployments) > 0 || cfg.Routing != nil),
	}
}

// SetActiveProvider persists a provider choice without requiring a model.
func (e *Engine) SetActiveProvider(ctx context.Context, providerID string) error {
	_ = nonNilContext(ctx)
	providerID = NormalizeProviderID(providerID)
	if providerID == "" {
		return invalid("set_active_provider", "eyrie engine: provider id is required")
	}
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	config.SetActiveProvider(cfg, providerID)
	if err := config.SaveProviderConfig(cfg, e.providerConfigPath); err != nil {
		return &Error{Code: ErrorInternal, Operation: "set_active_provider", Provider: providerID, Message: err.Error(), Cause: err}
	}
	return nil
}

// SetActiveModel persists a model choice and infers its serving provider from
// the configured catalog.
func (e *Engine) SetActiveModel(ctx context.Context, modelID string) error {
	return e.SetSelection(ctx, "", modelID)
}

// SetSelection persists the host/user's provider and model choice in this
// Engine's configured state path. It does not persist credentials.
func (e *Engine) SetSelection(ctx context.Context, providerID, modelID string) error {
	ctx = nonNilContext(ctx)
	_ = ctx
	providerID = catalog.CanonicalProviderID(strings.TrimSpace(providerID))
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return invalid("set_selection", "eyrie engine: model id is required")
	}
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	compiled, err := catalog.LoadCatalog(context.Background(), catalog.LoadCatalogOptions{CachePath: e.catalogPath, RequireCache: true})
	if err == nil && compiled != nil {
		canonical, ok := compiled.CanonicalModelForAliasOrID(modelID)
		if ok {
			modelID = canonical
		}
		if providerID == "" {
			providerID = catalog.GatewayForModel(compiled, modelID)
			if providerID == "" {
				providerID = catalog.ProviderForModel(compiled, modelID)
			}
		}
	}
	if providerID == "" {
		providerID = config.ActiveProvider(cfg)
	}
	config.SetProviderModel(cfg, providerID, modelID)
	if err := config.SaveProviderConfig(cfg, e.providerConfigPath); err != nil {
		return &Error{Code: ErrorInternal, Operation: "set_selection", Provider: providerID, Model: modelID, Message: err.Error(), Cause: err}
	}
	return nil
}

// ClearSelection removes active provider/model state without modifying
// credentials, deployments, or routing.
func (e *Engine) ClearSelection(ctx context.Context) error {
	_ = nonNilContext(ctx)
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	if cfg == nil {
		return nil
	}
	config.ClearActiveSelection(cfg)
	if err := config.SaveProviderConfig(cfg, e.providerConfigPath); err != nil {
		return &Error{Code: ErrorInternal, Operation: "clear_selection", Message: err.Error(), Cause: err}
	}
	return nil
}
