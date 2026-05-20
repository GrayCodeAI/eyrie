package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
)

// ActiveModel returns the selected model from provider.json.
func ActiveModel(ctx context.Context) string {
	cfg := config.LoadProviderConfig("")
	return config.ActiveModel(cfg)
}

// ActiveProvider returns the selected provider from provider.json.
func ActiveProvider(ctx context.Context) string {
	_ = ctx
	cfg := config.LoadProviderConfig("")
	return config.ActiveProvider(cfg)
}

// SetActiveModel persists the user's model choice to provider.json.
func SetActiveModel(ctx context.Context, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("runtime: model id required")
	}
	cfg := config.LoadProviderConfig("")
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	provider := inferProviderForModel(ctx, modelID)
	if provider == "" {
		provider = config.ActiveProvider(cfg)
	}
	if provider == "" {
		provider = config.DefaultProviderFromConfig(cfg)
	}
	config.SetProviderModel(cfg, provider, modelID)
	path := config.GetProviderConfigPath()
	return config.SaveProviderConfig(cfg, path)
}

// SetActiveProvider persists active_provider to provider.json.
func SetActiveProvider(ctx context.Context, provider string) error {
	_ = ctx
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return fmt.Errorf("runtime: provider required")
	}
	cfg := config.LoadProviderConfig("")
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	config.SetActiveProvider(cfg, provider)
	return config.SaveProviderConfig(cfg, config.GetProviderConfigPath())
}

func inferProviderForModel(ctx context.Context, modelID string) string {
	rt, err := Load(ctx)
	if err != nil || rt == nil || rt.Catalog == nil {
		return ""
	}
	if canonical, ok := rt.Catalog.CanonicalModelForAliasOrID(modelID); ok {
		if m, ok := rt.Catalog.ModelsByID[canonical]; ok {
			return catalog.CanonicalProviderID(m.ProviderID)
		}
	}
	return ""
}
