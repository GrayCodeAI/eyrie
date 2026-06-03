package config

import "strings"

// EnsureDeploymentConfigV2 upgrades legacy flat provider.json to deployment-aware v2.
func EnsureDeploymentConfigV2(cfg *ProviderConfig) *ProviderConfig {
	if cfg == nil {
		return nil
	}
	MigrateLegacyXiaomiProvider(cfg)
	if cfg.ConfigVersion >= 2 || len(cfg.Deployments) > 0 || cfg.Routing != nil {
		if cfg.ConfigVersion < 2 && (len(cfg.Deployments) > 0 || cfg.Routing != nil) {
			cfg.ConfigVersion = 2
		}
		return cfg
	}
	deployments := map[string]DeploymentConfig{}
	legacy := []struct {
		provider string
		id       string
	}{
		{ProviderAnthropic, "anthropic-direct"},
		{ProviderOpenAI, "openai-direct"},
		{ProviderGrok, "grok-direct"},
		{ProviderGemini, "gemini-direct"},
		{ProviderOpenRouter, "openrouter"},
		{ProviderZAI, "z-ai-direct"},
		{ProviderCanopyWave, "canopywave"},
		{ProviderOllama, "ollama-local"},
		{ProviderOpenCodeGo, "opencodego"},
		{ProviderKimi, "kimi-direct"},
		{ProviderXiaomiMimoPayg, "xiaomi_mimo_payg-direct"},
		{ProviderXiaomiMimoTokenPlan, "xiaomi_mimo_token_plan-direct"},
	}
	for _, item := range legacy {
		dep := legacyDeploymentConfig(cfg, item.provider)
		if legacyDeploymentConfigured(dep, item.provider) {
			deployments[item.id] = dep
		}
	}
	if len(deployments) == 0 {
		return cfg
	}
	cfg.Deployments = deployments
	cfg.ConfigVersion = 2
	if cfg.Routing == nil {
		cfg.Routing = defaultRoutingPolicyV2(deployments)
	}
	return cfg
}

func legacyDeploymentConfig(cfg *ProviderConfig, provider string) DeploymentConfig {
	if cfg == nil {
		return DeploymentConfig{}
	}
	switch provider {
	case ProviderAnthropic:
		return DeploymentConfig{APIKey: cfg.AnthropicAPIKey, BaseURL: cfg.AnthropicBaseURL}
	case ProviderOpenAI:
		return DeploymentConfig{APIKey: cfg.OpenAIAPIKey, BaseURL: cfg.OpenAIBaseURL}
	case ProviderGrok:
		return DeploymentConfig{APIKey: legacyFirstNonEmpty(cfg.GrokAPIKey, cfg.XAIAPIKey), BaseURL: legacyFirstNonEmpty(cfg.GrokBaseURL, cfg.XAIBaseURL)}
	case ProviderGemini:
		return DeploymentConfig{APIKey: cfg.GeminiAPIKey, BaseURL: cfg.GeminiBaseURL}
	case ProviderOpenRouter:
		return DeploymentConfig{APIKey: cfg.OpenRouterAPIKey, BaseURL: cfg.OpenRouterBaseURL}
	case ProviderCanopyWave:
		return DeploymentConfig{APIKey: cfg.CanopyWaveAPIKey, BaseURL: cfg.CanopyWaveBaseURL}
	case ProviderZAI:
		return DeploymentConfig{APIKey: cfg.ZAIAPIKey, BaseURL: cfg.ZAIBaseURL}
	case ProviderOllama:
		return DeploymentConfig{BaseURL: cfg.OllamaBaseURL}
	case ProviderOpenCodeGo:
		return DeploymentConfig{APIKey: cfg.OpenCodeGoAPIKey, BaseURL: cfg.OpenCodeGoBaseURL}
	case ProviderKimi:
		return DeploymentConfig{APIKey: cfg.MoonshotAPIKey, BaseURL: cfg.MoonshotBaseURL}
	case ProviderXiaomiMimoPayg:
		return DeploymentConfig{
			APIKey:  legacyFirstNonEmpty(cfg.XiaomiMimoPaygAPIKey, cfg.XiaomiAPIKey),
			BaseURL: legacyFirstNonEmpty(cfg.XiaomiMimoPaygBaseURL, cfg.XiaomiBaseURL),
		}
	case ProviderXiaomiMimoTokenPlan:
		base, _ := ResolveXiaomiOpenAIBase(ProviderXiaomiMimoTokenPlan, cfg)
		return DeploymentConfig{APIKey: cfg.XiaomiMimoTokenPlanAPIKey, BaseURL: base}
	default:
		return DeploymentConfig{}
	}
}

func legacyDeploymentConfigured(dep DeploymentConfig, provider string) bool {
	switch provider {
	case ProviderOllama:
		return strings.TrimSpace(dep.BaseURL) != ""
	default:
		return strings.TrimSpace(dep.APIKey) != "" ||
			strings.TrimSpace(dep.Token) != "" ||
			strings.TrimSpace(dep.AccessKeyID) != ""
	}
}

func defaultRoutingPolicyV2(deployments map[string]DeploymentConfig) *RoutingPolicy {
	byProvider := map[string][]RoutingStage{}
	for id := range deployments {
		switch id {
		case "anthropic-direct":
			byProvider["anthropic"] = anthropicRoutingStagesV2(deployments)
		case "openai-direct":
			byProvider["openai"] = []RoutingStage{{
				Deployments: []DeploymentChoice{{DeploymentID: "openai-direct", Weight: 100}},
				Retries:     1,
			}}
		default:
			provider := deploymentOwnerProviderID(id)
			if provider == "" {
				continue
			}
			byProvider[provider] = []RoutingStage{{
				Deployments: []DeploymentChoice{{DeploymentID: id, Weight: 100}},
				Retries:     1,
			}}
		}
	}
	return &RoutingPolicy{Providers: byProvider}
}

func anthropicRoutingStagesV2(deployments map[string]DeploymentConfig) []RoutingStage {
	var primary []DeploymentChoice
	if _, ok := deployments["anthropic-direct"]; ok {
		primary = append(primary, DeploymentChoice{DeploymentID: "anthropic-direct", Weight: 100})
	}
	if len(primary) == 0 {
		return nil
	}
	stages := []RoutingStage{{Deployments: primary, Retries: 1}}
	var fallback []DeploymentChoice
	for _, id := range []string{"anthropic-vertex", "anthropic-bedrock"} {
		if _, ok := deployments[id]; ok {
			fallback = append(fallback, DeploymentChoice{DeploymentID: id, Weight: 100})
		}
	}
	if len(fallback) > 0 {
		stages = append(stages, RoutingStage{Deployments: fallback, Retries: 1})
	}
	return stages
}

func deploymentOwnerProviderID(deploymentID string) string {
	switch deploymentID {
	case "anthropic-direct", "anthropic-bedrock", "anthropic-vertex":
		return "anthropic"
	case "openai-direct", "openai-azure":
		return "openai"
	case "gemini-direct", "gemini-vertex":
		return "google"
	case "grok-direct":
		return "xai"
	case "openrouter":
		return "openrouter"
	case "canopywave":
		return "canopywave"
	case "z-ai-direct":
		return "z-ai"
	case "ollama-local":
		return "ollama"
	case "opencodego":
		return "opencodego"
	case "kimi-direct":
		return "kimi"
	case "xiaomi_mimo_payg-direct", "xiaomi_mimo-direct":
		return "xiaomi_mimo_payg"
	case "xiaomi_mimo_token_plan-direct":
		return "xiaomi_mimo_token_plan"
	default:
		return ""
	}
}

func legacyFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
