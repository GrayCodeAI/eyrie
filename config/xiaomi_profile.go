package config

import (
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
)

const (
	EnvXiaomiPaygAPIKey       = "XIAOMI_MIMO_PAYG_API_KEY"
	EnvXiaomiTokenPlanAPIKey  = "XIAOMI_MIMO_TOKEN_PLAN_API_KEY"
	EnvXiaomiPaygBaseURL      = "XIAOMI_MIMO_PAYG_BASE_URL"
	EnvXiaomiTokenPlanBaseURL = "XIAOMI_MIMO_TOKEN_PLAN_BASE_URL"
	EnvXiaomiTokenPlanRegion  = "XIAOMI_MIMO_TOKEN_PLAN_REGION"
)

// XiaomiTokenPlanRegionFromConfig reads persisted Token Plan cluster from provider.json.
func XiaomiTokenPlanRegionFromConfig(cfg *ProviderConfig) xiaomi.Region {
	if cfg == nil {
		return ""
	}
	r, _ := xiaomi.NormalizeRegion(cfg.XiaomiMimoTokenPlanRegion)
	return r
}

// ResolveXiaomiOpenAIBase resolves the OpenAI-compat base for a MiMo gateway id.
func ResolveXiaomiOpenAIBase(providerID string, cfg *ProviderConfig) (string, error) {
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
			region = XiaomiTokenPlanRegionFromConfig(cfg)
		}
	}
	return xiaomi.ResolveOpenAIBasePreferRegion(billing, region, override)
}

// ResolveXiaomiAnthropicBase resolves the Anthropic-compat base for a MiMo gateway id.
func ResolveXiaomiAnthropicBase(providerID string, cfg *ProviderConfig) (string, error) {
	billing, ok := xiaomi.BillingForProvider(providerID)
	if !ok {
		return "", nil
	}
	region := XiaomiTokenPlanRegionFromConfig(cfg)
	return xiaomi.ResolveAnthropicBase(billing, region)
}

// IsXiaomiMimoProvider reports whether id is a MiMo setup gateway (payg or token plan).
func IsXiaomiMimoProvider(providerID string) bool {
	_, ok := xiaomi.BillingForProvider(providerID)
	return ok
}

// MigrateLegacyXiaomiProvider rewrites deprecated xiaomi_mimo ids and env to payg.
func MigrateLegacyXiaomiProvider(cfg *ProviderConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.ActiveProvider) == "xiaomi_mimo" {
		cfg.ActiveProvider = xiaomi.ProviderPayAsYouGo
	}
	if key := strings.TrimSpace(cfg.XiaomiAPIKey); key != "" && strings.TrimSpace(cfg.XiaomiMimoPaygAPIKey) == "" {
		cfg.XiaomiMimoPaygAPIKey = key
	}
	if base := strings.TrimSpace(cfg.XiaomiBaseURL); base != "" && strings.TrimSpace(cfg.XiaomiMimoPaygBaseURL) == "" {
		cfg.XiaomiMimoPaygBaseURL = base
	}
	if cfg.Deployments != nil {
		if dep, ok := cfg.Deployments["xiaomi_mimo-direct"]; ok {
			if _, exists := cfg.Deployments["xiaomi_mimo_payg-direct"]; !exists {
				cfg.Deployments["xiaomi_mimo_payg-direct"] = dep
			}
			delete(cfg.Deployments, "xiaomi_mimo-direct")
		}
	}
}
