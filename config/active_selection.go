package config

import (
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
)

// ActiveModel returns the user's selected model from provider.json (canonical when possible).
func ActiveModel(cfg *ProviderConfig) string {
	if cfg == nil {
		return ""
	}
	if m := AsNonEmptyString(cfg.ActiveModel); m != "" {
		return m
	}
	provider := DefaultProviderFromConfig(cfg)
	if provider == "" {
		return ""
	}
	return GetProviderActiveModel(cfg, provider)
}

// ActiveProvider returns the configured active provider id.
func ActiveProvider(cfg *ProviderConfig) string {
	if cfg == nil {
		return ""
	}
	if p := AsNonEmptyString(cfg.ActiveProvider); p != "" {
		return catalog.CanonicalProviderID(p)
	}
	return catalog.CanonicalProviderID(DefaultProviderFromConfig(cfg))
}

// SetActiveProvider updates active_provider in provider config.
func SetActiveProvider(cfg *ProviderConfig, provider string) {
	if cfg == nil {
		return
	}
	provider = catalog.CanonicalProviderID(provider)
	if provider == "" {
		return
	}
	cfg.ActiveProvider = provider
}

// SetProviderModel sets active_model and the provider-scoped model field.
func SetProviderModel(cfg *ProviderConfig, provider, model string) {
	if cfg == nil {
		return
	}
	model = strings.TrimSpace(model)
	provider = catalog.CanonicalProviderID(provider)
	if model != "" {
		cfg.ActiveModel = model
	}
	if provider != "" {
		cfg.ActiveProvider = provider
	}
	if model == "" || provider == "" {
		return
	}
	switch provider {
	case ProviderAnthropic:
		cfg.AnthropicModel = model
	case ProviderOpenAI:
		cfg.OpenAIModel = model
	case ProviderCanopyWave:
		cfg.CanopyWaveModel = model
	case ProviderOpenRouter:
		cfg.OpenRouterModel = model
	case ProviderGrok:
		cfg.GrokModel = model
		cfg.XAIModel = model
	case ProviderGemini:
		cfg.GeminiModel = model
	case ProviderOllama:
		cfg.OllamaModel = model
	case ProviderOpenCodeGo:
		cfg.OpenCodeGoModel = model
	default:
		// Unknown/custom provider: active_model + active_provider are enough.
	}
}
