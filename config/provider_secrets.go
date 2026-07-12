package config

import "strings"

// ProviderConfigContainsSecrets reports whether legacy provider state contains
// credential material. It never returns or formats credential values.
func ProviderConfigContainsSecrets(cfg ProviderConfig) bool {
	for _, secret := range providerConfigSecrets(cfg) {
		if strings.TrimSpace(secret) != "" {
			return true
		}
	}
	for _, deployment := range cfg.Deployments {
		if deploymentContainsSecrets(deployment) {
			return true
		}
	}
	return false
}

// SanitizeProviderConfigForDisk removes every historical credential field
// while preserving provider selection, routing, endpoints, and model metadata.
func SanitizeProviderConfigForDisk(cfg ProviderConfig) ProviderConfig {
	cfg.AnthropicAPIKey = ""
	cfg.GrokAPIKey = ""
	cfg.XAIAPIKey = ""
	cfg.OpenAIAPIKey = ""
	cfg.CanopyWaveAPIKey = ""
	cfg.DeepSeekAPIKey = ""
	cfg.ZAIAPIKey = ""
	cfg.ZAICodingAPIKey = ""
	cfg.OpenRouterAPIKey = ""
	cfg.GeminiAPIKey = ""
	cfg.OpenCodeGoAPIKey = ""
	cfg.MoonshotAPIKey = ""
	cfg.XiaomiMimoPaygAPIKey = ""
	cfg.XiaomiMimoTokenPlanAPIKey = ""
	cfg.MiniMaxTokenPlanAPIKey = ""
	cfg.MiniMaxPaygAPIKey = ""
	cfg.PoolsideAPIKey = ""
	cfg.GroqAPIKey = ""
	cfg.ClinePassAPIKey = ""
	if cfg.Deployments != nil {
		deployments := make(map[string]DeploymentConfig, len(cfg.Deployments))
		for id, deployment := range cfg.Deployments {
			deployments[id] = SanitizeDeploymentConfigForDisk(deployment)
		}
		cfg.Deployments = deployments
	}
	return cfg
}

func providerConfigSecrets(cfg ProviderConfig) []string {
	return []string{
		cfg.AnthropicAPIKey, cfg.GrokAPIKey, cfg.XAIAPIKey, cfg.OpenAIAPIKey,
		cfg.CanopyWaveAPIKey, cfg.DeepSeekAPIKey, cfg.ZAIAPIKey, cfg.ZAICodingAPIKey,
		cfg.OpenRouterAPIKey, cfg.GeminiAPIKey, cfg.OpenCodeGoAPIKey, cfg.MoonshotAPIKey,
		cfg.XiaomiMimoPaygAPIKey, cfg.XiaomiMimoTokenPlanAPIKey,
		cfg.MiniMaxTokenPlanAPIKey, cfg.MiniMaxPaygAPIKey,
		cfg.PoolsideAPIKey, cfg.GroqAPIKey, cfg.ClinePassAPIKey,
	}
}

func deploymentContainsSecrets(deployment DeploymentConfig) bool {
	return strings.TrimSpace(deployment.APIKey) != "" || strings.TrimSpace(deployment.Token) != "" ||
		strings.TrimSpace(deployment.SecretAccessKey) != "" || strings.TrimSpace(deployment.AccessKeyID) != "" ||
		strings.TrimSpace(deployment.SessionToken) != ""
}
