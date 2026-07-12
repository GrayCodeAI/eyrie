package engine

import (
	"context"
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
)

func (e *Engine) policyCatalog(ctx context.Context) (*catalog.CompiledCatalog, error) {
	compiled, err := catalog.LoadCatalog(nonNilContext(ctx), catalog.LoadCatalogOptions{CachePath: e.catalogPath})
	if err != nil {
		return nil, &Error{Code: ErrorCatalogUnavailable, Operation: "model_policy", Message: err.Error(), Cause: err}
	}
	return compiled, nil
}

// ModelInfo returns normalized metadata for a model id or alias.
func (e *Engine) ModelInfo(ctx context.Context, modelID string) (Model, bool, error) {
	compiled, err := e.policyCatalog(ctx)
	if err != nil {
		return Model{}, false, err
	}
	canonical, ok := compiled.CanonicalModelForAliasOrID(strings.TrimSpace(modelID))
	if !ok {
		return Model{}, false, nil
	}
	for _, model := range snapshotFromCompiled(compiled).Models {
		if model.ID == canonical {
			return model, true, nil
		}
	}
	return Model{}, false, nil
}

// ModelProviders lists canonical catalog model owners.
func (e *Engine) ModelProviders(ctx context.Context) ([]string, error) {
	compiled, err := e.policyCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return catalog.AllModelProviders(compiled), nil
}

// DefaultModel returns the catalog default for a provider.
func (e *Engine) DefaultModel(ctx context.Context, provider, fallback string) string {
	compiled, err := e.policyCatalog(ctx)
	if err != nil {
		return fallback
	}
	return catalog.ProviderDefaultModel(compiled, provider, fallback)
}

// PreferredModel returns a provider model in the requested relative class.
func (e *Engine) PreferredModel(ctx context.Context, provider string, class ModelClass, fallback string) string {
	compiled, err := e.policyCatalog(ctx)
	if err != nil {
		return fallback
	}
	return catalog.PreferredProviderModel(compiled, provider, catalogTier(class), fallback)
}

// PreferredModels returns unique cross-provider candidates in preference order.
func (e *Engine) PreferredModels(ctx context.Context, primaryProvider string, class ModelClass, limit int) []string {
	compiled, err := e.policyCatalog(ctx)
	if err != nil {
		return nil
	}
	return catalog.PreferredModelsForTier(compiled, primaryProvider, catalogTier(class), limit)
}

// ModelClassOf resolves a model's relative catalog cost band.
func (e *Engine) ModelClassOf(ctx context.Context, modelID string) ModelClass {
	compiled, _ := e.policyCatalog(ctx)
	switch catalog.ModelCostTierOf(compiled, modelID) {
	case catalog.CostTierCheap:
		return ModelClassEconomical
	case catalog.CostTierExpensive:
		return ModelClassPremium
	default:
		return ModelClassBalanced
	}
}

// ProviderForModel returns the canonical catalog owner for a model.
func (e *Engine) ProviderForModel(ctx context.Context, modelID string) string {
	compiled, _ := e.policyCatalog(ctx)
	return catalog.ProviderForModel(compiled, modelID)
}

// PrimaryModel returns Eyrie's stable best-effort catalog primary model.
func (e *Engine) PrimaryModel(ctx context.Context) string {
	compiled, _ := e.policyCatalog(ctx)
	return catalog.PrimaryModel(compiled)
}

// ModelNames returns canonical ids, display names, and aliases for completion.
func (e *Engine) ModelNames(ctx context.Context) []string {
	compiled, err := e.policyCatalog(ctx)
	if err != nil || compiled == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	for id, model := range compiled.ModelsByID {
		add(id)
		add(model.Name)
	}
	if compiled.Catalog != nil {
		for alias, canonical := range compiled.Catalog.Aliases {
			add(alias)
			add(canonical)
		}
	}
	sort.Strings(out)
	return out
}

func catalogTier(class ModelClass) catalog.ModelTier {
	switch class {
	case ModelClassEconomical:
		return catalog.TierHaiku
	case ModelClassPremium:
		return catalog.TierOpus
	default:
		return catalog.TierSonnet
	}
}
