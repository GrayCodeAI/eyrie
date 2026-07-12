package engine

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
)

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
