// Package setup wires catalog-backed deployment routing for hawk and eyrie CLIs.
package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
	"github.com/GrayCodeAI/eyrie/router"
)

func storeSecret(envKeys ...string) string {
	ctx := context.Background()
	for _, k := range envKeys {
		if v := credentials.LookupSecret(ctx, k); v != "" {
			return v
		}
	}
	return ""
}

// UseDeploymentRouting mirrors eyrie CLI behavior: env override, then provider.json shape.
func UseDeploymentRouting(cfg *config.ProviderConfig) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("EYRIE_DEPLOYMENT_ROUTING"))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return cfg != nil && (cfg.ConfigVersion >= 2 || len(cfg.Deployments) > 0 || cfg.Routing != nil)
}

// DeploymentProvider builds a catalog-aware router over configured deployments.
func DeploymentProvider(ctx context.Context, cfg *config.ProviderConfig) (client.Provider, error) {
	cfg = config.EnsureDeploymentConfigV2(cfg)
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, ".eyrie", "model_catalog.json")
	compiled, err := catalog.LoadCatalogV1(ctx, catalog.LoadCatalogV1Options{
		CachePath:     cachePath,
		RefreshRemote: strings.EqualFold(os.Getenv("EYRIE_MODEL_CATALOG_REFRESH"), "true"),
	})
	if err != nil {
		return nil, err
	}
	deployments := ConfiguredDeploymentAdapters(cfg)
	if len(deployments) == 0 {
		return nil, fmt.Errorf("no deployment credentials configured")
	}
	return router.NewDeploymentRouter(router.DeploymentRouterOptions{
		Catalog:     compiled,
		Deployments: deployments,
		Routing:     RouterRoutingPolicy(cfg.Routing),
	})
}

// ConfiguredDeploymentAdapters maps deployment IDs to live provider clients.
func ConfiguredDeploymentAdapters(cfg *config.ProviderConfig) map[string]router.DeploymentAdapter {
	out := map[string]router.DeploymentAdapter{}
	for id, deployment := range ConfiguredDeployments(cfg) {
		provider, ok := ProviderForDeployment(id, deployment)
		if !ok {
			continue
		}
		out[id] = router.DeploymentAdapter{
			DeploymentID:  id,
			Provider:      provider,
			ModelMappings: CloneStringMap(deployment.ModelMappings),
		}
	}
	return out
}

// ConfiguredDeployments merges explicit deployments with legacy single-provider config.
func ConfiguredDeployments(cfg *config.ProviderConfig) map[string]config.DeploymentConfig {
	out := map[string]config.DeploymentConfig{}
	if cfg != nil {
		for id, deployment := range cfg.Deployments {
			out[id] = deployment
		}
	}
	if len(out) > 0 {
		return out
	}
	provider := client.DetectProvider()
	if cfg != nil {
		if configured := config.DefaultProviderFromConfig(cfg); configured != "" {
			provider = configured
		}
	}
	if id := DefaultDeploymentForProvider(provider); id != "" {
		out[id] = LegacyDeploymentConfig(cfg, provider)
	}
	return out
}

// ProviderForDeployment constructs the API client for a catalog deployment ID.
func ProviderForDeployment(id string, deployment config.DeploymentConfig) (client.Provider, bool) {
	switch id {
	case "anthropic-direct":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("ANTHROPIC_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewAnthropicClient(apiKey, FirstNonEmpty(deployment.BaseURL, os.Getenv("ANTHROPIC_BASE_URL"))), true
	case "anthropic-vertex":
		projectID := FirstNonEmpty(deployment.ProjectID, os.Getenv("VERTEX_PROJECT_ID"))
		region := FirstNonEmpty(deployment.Region, os.Getenv("VERTEX_REGION"))
		token := FirstNonEmpty(deployment.Token, deployment.APIKey, storeSecret("VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"))
		if projectID == "" || region == "" || token == "" {
			return nil, false
		}
		return client.NewVertexClient(projectID, region, token), true
	case "anthropic-bedrock":
		region := FirstNonEmpty(deployment.Region, os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"))
		accessKeyID := FirstNonEmpty(deployment.AccessKeyID, deployment.APIKey, storeSecret("AWS_ACCESS_KEY_ID"))
		secretAccessKey := FirstNonEmpty(deployment.SecretAccessKey, deployment.Token, storeSecret("AWS_SECRET_ACCESS_KEY"))
		sessionToken := FirstNonEmpty(deployment.SessionToken, storeSecret("AWS_SESSION_TOKEN"))
		if region == "" || accessKeyID == "" || secretAccessKey == "" {
			return nil, false
		}
		return client.NewBedrockClient(accessKeyID, secretAccessKey, sessionToken, region), true
	case "openai-direct":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("OPENAI_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultOpenAIBaseURL), &client.OpenAICompat), true
	case "openai-azure":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("AZURE_OPENAI_API_KEY"))
		endpoint := FirstNonEmpty(deployment.Endpoint, os.Getenv("AZURE_OPENAI_ENDPOINT"))
		apiVersion := FirstNonEmpty(deployment.APIVersion, os.Getenv("AZURE_OPENAI_API_VERSION"))
		if apiKey == "" || endpoint == "" {
			return nil, false
		}
		return client.NewAzureClient(apiKey, endpoint, apiVersion), true
	case "grok-direct":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("XAI_API_KEY", "GROK_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultGrokOpenAIBaseURL), &client.GrokCompat), true
	case "gemini-direct":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("GEMINI_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultGeminiOpenAIBaseURL), &client.GeminiCompat), true
	case "openrouter":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("OPENROUTER_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultOpenRouterOpenAIBaseURL), &client.OpenRouterCompat), true
	case "canopywave":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("CANOPYWAVE_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultCanopyWaveOpenAIBaseURL), &client.CanopyWaveCompat), true
	case "z-ai-direct":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("ZAI_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultZAIOpenAIBaseURL), &client.ZAICompat), true
	case "ollama-local":
		baseURL := config.NormalizeOllamaOpenAIBaseURL(FirstNonEmpty(deployment.BaseURL, os.Getenv("OLLAMA_BASE_URL"), config.OllamaDefaultBaseURL))
		return client.NewOpenAIClient(FirstNonEmpty(deployment.APIKey, storeSecret("OLLAMA_API_KEY")), baseURL, &client.OllamaCompat), true
	case "opencodego":
		apiKey := FirstNonEmpty(deployment.APIKey, storeSecret("OPENCODEGO_API_KEY"))
		if apiKey == "" {
			return nil, false
		}
		return client.NewOpenAIClient(apiKey, FirstNonEmpty(deployment.BaseURL, config.DefaultOpenCodeGoBaseURL), &client.OpenCodeGoCompat), true
	default:
		return nil, false
	}
}

// DefaultDeploymentForProvider maps a logical provider name to a deployment ID.
func DefaultDeploymentForProvider(provider string) string {
	switch provider {
	case config.ProviderAnthropic:
		return "anthropic-direct"
	case config.ProviderOpenAI:
		return "openai-direct"
	case config.ProviderGrok:
		return "grok-direct"
	case config.ProviderGemini:
		return "gemini-direct"
	case config.ProviderOpenRouter:
		return "openrouter"
	case config.ProviderCanopyWave:
		return "canopywave"
	case config.ProviderZAI:
		return "z-ai-direct"
	case config.ProviderOllama:
		return "ollama-local"
	case config.ProviderOpenCodeGo:
		return "opencodego"
	default:
		return ""
	}
}

// LegacyDeploymentConfig reads API keys from flat provider.json fields.
func LegacyDeploymentConfig(cfg *config.ProviderConfig, provider string) config.DeploymentConfig {
	if cfg == nil {
		return config.DeploymentConfig{}
	}
	switch provider {
	case config.ProviderAnthropic:
		return config.DeploymentConfig{APIKey: cfg.AnthropicAPIKey, BaseURL: cfg.AnthropicBaseURL}
	case config.ProviderOpenAI:
		return config.DeploymentConfig{APIKey: cfg.OpenAIAPIKey, BaseURL: cfg.OpenAIBaseURL}
	case config.ProviderGrok:
		return config.DeploymentConfig{APIKey: FirstNonEmpty(cfg.GrokAPIKey, cfg.XAIAPIKey), BaseURL: FirstNonEmpty(cfg.GrokBaseURL, cfg.XAIBaseURL)}
	case config.ProviderGemini:
		return config.DeploymentConfig{APIKey: cfg.GeminiAPIKey, BaseURL: cfg.GeminiBaseURL}
	case config.ProviderOpenRouter:
		return config.DeploymentConfig{APIKey: cfg.OpenRouterAPIKey, BaseURL: cfg.OpenRouterBaseURL}
	case config.ProviderCanopyWave:
		return config.DeploymentConfig{APIKey: cfg.CanopyWaveAPIKey, BaseURL: cfg.CanopyWaveBaseURL}
	case config.ProviderZAI:
		return config.DeploymentConfig{APIKey: cfg.ZAIAPIKey, BaseURL: cfg.ZAIBaseURL}
	case config.ProviderOllama:
		return config.DeploymentConfig{BaseURL: cfg.OllamaBaseURL}
	case config.ProviderOpenCodeGo:
		return config.DeploymentConfig{APIKey: cfg.OpenCodeGoAPIKey, BaseURL: cfg.OpenCodeGoBaseURL}
	default:
		return config.DeploymentConfig{}
	}
}

// RouterRoutingPolicy converts config routing JSON into router policy.
func RouterRoutingPolicy(policy *config.RoutingPolicy) router.RoutingPolicy {
	if policy == nil {
		return router.RoutingPolicy{}
	}
	return router.RoutingPolicy{
		Default:   RouterRoutingStages(policy.Default),
		Providers: RouterRoutingStageMap(policy.Providers),
		Models:    RouterRoutingStageMap(policy.Models),
	}
}

// RouterRoutingStageMap converts per-key routing stages.
func RouterRoutingStageMap(in map[string][]config.RoutingStage) map[string][]router.RoutingStage {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]router.RoutingStage, len(in))
	for key, stages := range in {
		out[key] = RouterRoutingStages(stages)
	}
	return out
}

// RouterRoutingStages converts config stages to router stages.
func RouterRoutingStages(stages []config.RoutingStage) []router.RoutingStage {
	out := make([]router.RoutingStage, len(stages))
	for i, stage := range stages {
		out[i].Retries = stage.Retries
		out[i].Deployments = make([]router.DeploymentChoice, len(stage.Deployments))
		for j, choice := range stage.Deployments {
			out[i].Deployments[j] = router.DeploymentChoice{DeploymentID: choice.DeploymentID, Weight: choice.Weight}
		}
	}
	return out
}

// FirstNonEmpty returns the first non-empty trimmed string.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// CloneStringMap returns a shallow copy of m.
func CloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
