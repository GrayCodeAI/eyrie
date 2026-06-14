package config

import (
	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
)

const (
	EnvXiaomiMimoPaygAPIKey       = "XIAOMI_MIMO_PAYG_API_KEY"
	EnvXiaomiMimoTokenPlanAPIKey  = "XIAOMI_MIMO_TOKEN_PLAN_API_KEY"
	EnvXiaomiMimoPaygBaseURL      = "XIAOMI_MIMO_PAYG_BASE_URL"
	EnvXiaomiMimoTokenPlanBaseURL = "XIAOMI_MIMO_TOKEN_PLAN_BASE_URL"
	EnvXiaomiMimoTokenPlanRegion  = "XIAOMI_MIMO_TOKEN_PLAN_REGION"
)

// XiaomiMimoTokenPlanRegionFromConfig reads persisted Token Plan cluster from provider.json.
func XiaomiMimoTokenPlanRegionFromConfig(cfg *ProviderConfig) xiaomi.Region {
	if cfg == nil {
		return ""
	}
	r, _ := xiaomi.NormalizeRegion(cfg.XiaomiMimoTokenPlanRegion)
	return r
}

// ResolveXiaomiMimoOpenAIBase resolves the OpenAI-compat base for a MiMo gateway id.
func ResolveXiaomiMimoOpenAIBase(providerID string, cfg *ProviderConfig) (string, error) {
	billing, ok := xiaomi.BillingForProvider(providerID)
	if !ok {
		return "", nil
	}
	var override string
	var region xiaomi.Region
	switch billing {
	case xiaomi.BillingPayAsYouGo:
		if cfg != nil {
			override = cfg.XiaomiMimoPaygBaseURL
		}
	case xiaomi.BillingTokenPlan:
		if cfg != nil {
			override = cfg.XiaomiMimoTokenPlanBaseURL
			region = XiaomiMimoTokenPlanRegionFromConfig(cfg)
		}
	}
	return xiaomi.ResolveOpenAIBasePreferRegion(billing, region, override)
}

// ResolveXiaomiMimoAnthropicBase resolves the Anthropic-compat base for a MiMo gateway id.
func ResolveXiaomiMimoAnthropicBase(providerID string, cfg *ProviderConfig) (string, error) {
	billing, ok := xiaomi.BillingForProvider(providerID)
	if !ok {
		return "", nil
	}
	region := XiaomiMimoTokenPlanRegionFromConfig(cfg)
	return xiaomi.ResolveAnthropicBase(billing, region)
}

// IsXiaomiMimoProvider reports whether id is a MiMo setup gateway (payg or token plan).
func IsXiaomiMimoProvider(providerID string) bool {
	_, ok := xiaomi.BillingForProvider(providerID)
	return ok
}
