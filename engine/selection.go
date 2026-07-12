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
	compiled, err := catalog.LoadCatalog(context.Background(), catalog.LoadCatalogOptions{CachePath: e.catalogPath, RequireCache: true})
	if err != nil {
		return &Error{Code: ErrorCatalogUnavailable, Operation: "set_selection", Message: err.Error(), Cause: err}
	}
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
	if providerID == "" {
		return &Error{Code: ErrorModelUnavailable, Operation: "set_selection", Model: modelID, Message: "eyrie engine: could not determine provider for model"}
	}
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	if cfg == nil {
		cfg = &config.ProviderConfig{}
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
