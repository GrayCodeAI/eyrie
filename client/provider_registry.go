package client

import (
	"context"
	"fmt"

	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// ProviderType classifies providers.
type ProviderType string

const (
	// ProviderTypeAnthropic uses the Anthropic Messages API.
	ProviderTypeAnthropic ProviderType = "anthropic"
	// ProviderTypeOpenAI uses the OpenAI Chat Completions API.
	ProviderTypeOpenAI ProviderType = "openai"
	// ProviderTypeOpenAICompatible uses OpenAI-compatible APIs with custom base URLs.
	ProviderTypeOpenAICompatible ProviderType = "openai-compatible"
	// ProviderTypeAzure uses Azure OpenAI.
	ProviderTypeAzure ProviderType = "azure"
	// ProviderTypeBedrock uses AWS Bedrock.
	ProviderTypeBedrock ProviderType = "bedrock"
	// ProviderTypeVertex uses Google Vertex AI.
	ProviderTypeVertex ProviderType = "vertex"
)

// ProviderRegistryConfig holds provider registry info.
type ProviderRegistryConfig struct {
	Name              string              `json:"name"`
	Type              ProviderType        `json:"type"`
	BaseURL           string              `json:"base_url,omitempty"`
	EnvKey            string              `json:"env_key"`
	SupportsStreaming bool                `json:"supports_streaming"`
	SupportsTools     bool                `json:"supports_tools"`
	SupportsReasoning bool                `json:"supports_reasoning"`
	Compat            *OpenAICompatConfig `json:"compat,omitempty"`
}

// CoreProviders are providers with dedicated SDKs.
// This map is populated at package init time and must not be written to
// afterward. All reads are safe for concurrent use without locking.
var CoreProviders = map[string]ProviderRegistryConfig{
	"anthropic": {Name: "anthropic", Type: ProviderTypeAnthropic, EnvKey: "ANTHROPIC_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"openai":    {Name: "openai", Type: ProviderTypeOpenAI, BaseURL: "https://api.openai.com/v1", EnvKey: "OPENAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"azure":     {Name: "azure", Type: ProviderTypeAzure, EnvKey: "AZURE_OPENAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"bedrock":   {Name: "bedrock", Type: ProviderTypeBedrock, EnvKey: "AWS_SECRET_ACCESS_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"vertex":    {Name: "vertex", Type: ProviderTypeVertex, EnvKey: "VERTEX_ACCESS_TOKEN", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
}

// OpenAICompatibleProviders use the OpenAI SDK with custom baseUrl.
var OpenAICompatibleProviders = map[string]ProviderRegistryConfig{
	"grok":       {Name: "grok", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.x.ai/v1", EnvKey: "XAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"openrouter": {Name: "openrouter", Type: ProviderTypeOpenAICompatible, BaseURL: "https://openrouter.ai/api/v1", EnvKey: "OPENROUTER_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"z-ai":       {Name: "z-ai", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.z.ai/api/paas/v4", EnvKey: "ZAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"canopywave": {Name: "canopywave", Type: ProviderTypeOpenAICompatible, BaseURL: "https://inference.canopywave.io/v1", EnvKey: "CANOPYWAVE_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"gemini":     {Name: "gemini", Type: ProviderTypeOpenAICompatible, BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", EnvKey: "GEMINI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"ollama":     {Name: "ollama", Type: ProviderTypeOpenAICompatible, BaseURL: "http://localhost:11434/v1", EnvKey: "OLLAMA_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: false},
	"opencodego": {Name: "opencodego", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultOpenCodeGoBaseURL, EnvKey: "OPENCODEGO_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"kimi":       {Name: "kimi", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultKimiOpenAIBaseURL, EnvKey: "MOONSHOT_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"xiaomi":     {Name: "xiaomi", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultXiaomiOpenAIBaseURL, EnvKey: "XIAOMI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
}

// GetProviders lists all available providers.
func (c *EyrieClient) GetProviders() []string {
	var providers []string
	for k := range CoreProviders {
		providers = append(providers, k)
	}
	dynamicMu.RLock()
	for k := range OpenAICompatibleProviders {
		providers = append(providers, k)
	}
	dynamicMu.RUnlock()
	return providers
}

// GetProviderInfo returns config for a provider.
func (c *EyrieClient) GetProviderInfo(provider string) *ProviderRegistryConfig {
	if p, ok := CoreProviders[provider]; ok {
		return &p
	}
	dynamicMu.RLock()
	p, ok := OpenAICompatibleProviders[provider]
	dynamicMu.RUnlock()
	if ok {
		return &p
	}
	return nil
}

func (c *EyrieClient) getOrCreateProvider(providerName string) (Provider, error) {
	c.mu.RLock()
	if p, ok := c.providers[providerName]; ok {
		c.mu.RUnlock()
		return p, nil
	}
	hasKey := c.apiKeys[providerName] != ""
	needsRegistration := !hasKey && c.GetProviderInfo(providerName) == nil
	c.mu.RUnlock()

	// Register fallback provider BEFORE acquiring c.mu to avoid lock ordering issues.
	if needsRegistration {
		if fallbackURL := openaiBaseFallbackURL(); fallbackURL != "" {
			_ = RegisterDynamicProvider(providerName, fallbackURL, "OPENAI_API_KEY")
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if p, ok := c.providers[providerName]; ok {
		return p, nil
	}

	// Re-read apiKeys under write lock to avoid TOCTOU: another goroutine
	// may have added a key between our RUnlock and Lock.
	apiKey := c.apiKeys[providerName]
	if apiKey == "" {
		info := c.GetProviderInfo(providerName)
		if info == nil {
			return nil, fmt.Errorf("eyrie: unknown provider: %s", providerName)
		}
		apiKey = resolveEnvSecret(info.EnvKey)
		if apiKey == "" && providerName == "grok" {
			apiKey = resolveEnvSecret("GROK_API_KEY")
		}
	}

	info := c.GetProviderInfo(providerName)
	if info == nil {
		return nil, fmt.Errorf("eyrie: unknown provider: %s", providerName)
	}

	if apiKey == "" && providerName != "ollama" {
		return nil, fmt.Errorf("eyrie: no API key for %s; set %s or call SetAPIKey()", providerName, info.EnvKey)
	}

	var p Provider
	switch info.Type {
	case ProviderTypeAnthropic:
		p = NewAnthropicClient(apiKey, info.BaseURL)
	case ProviderTypeAzure:
		endpoint := resolveEnvSecret("AZURE_OPENAI_ENDPOINT")
		if endpoint == "" {
			endpoint = info.BaseURL
		}
		apiVersion := resolveEnvSecret("AZURE_OPENAI_API_VERSION")
		p = NewAzureClient(apiKey, endpoint, apiVersion)
	case ProviderTypeBedrock:
		region := resolveEnvSecret("AWS_REGION")
		if region == "" {
			region = resolveEnvSecret("AWS_DEFAULT_REGION")
		}
		if region == "" {
			region = "us-east-1"
		}
		accessKey := resolveEnvSecret("AWS_ACCESS_KEY_ID")
		sessionToken := resolveEnvSecret("AWS_SESSION_TOKEN")
		p = NewBedrockClient(accessKey, apiKey, sessionToken, region)
	case ProviderTypeVertex:
		projectID := resolveEnvSecret("VERTEX_PROJECT_ID")
		region := resolveEnvSecret("VERTEX_REGION")
		if region == "" {
			region = "us-central1"
		}
		p = NewVertexClient(projectID, region, apiKey)
	default:
		p = NewOpenAIClient(apiKey, info.BaseURL, info.Compat)
	}

	c.providers[providerName] = p
	return p, nil
}

// DetectProvider detects the active provider from the credential store (not process env).
func DetectProvider() string {
	ctx := context.Background()
	checks := map[string]func() bool{
		"anthropic":  func() bool { return credentials.HasSecret(ctx, "ANTHROPIC_API_KEY") },
		"openrouter": func() bool { return credentials.HasSecret(ctx, "OPENROUTER_API_KEY") },
		"grok": func() bool {
			return credentials.HasSecret(ctx, "GROK_API_KEY") || credentials.HasSecret(ctx, "XAI_API_KEY")
		},
		"gemini":     func() bool { return credentials.HasSecret(ctx, "GEMINI_API_KEY") },
		"z-ai":       func() bool { return credentials.HasSecret(ctx, "ZAI_API_KEY") },
		"canopywave": func() bool { return credentials.HasSecret(ctx, "CANOPYWAVE_API_KEY") },
		"openai":     func() bool { return credentials.HasSecret(ctx, "OPENAI_API_KEY") },
		"opencodego": func() bool { return credentials.HasSecret(ctx, "OPENCODEGO_API_KEY") },
		"ollama":     func() bool { return resolveEnvSecret("OLLAMA_BASE_URL") != "" },
		"azure": func() bool {
			return credentials.HasSecret(ctx, "AZURE_OPENAI_API_KEY") && resolveEnvSecret("AZURE_OPENAI_ENDPOINT") != ""
		},
		"bedrock": func() bool {
			return credentials.HasSecret(ctx, "AWS_ACCESS_KEY_ID") && credentials.HasSecret(ctx, "AWS_SECRET_ACCESS_KEY")
		},
		"vertex": func() bool {
			return credentials.HasSecret(ctx, "VERTEX_PROJECT_ID") && credentials.HasSecret(ctx, "VERTEX_ACCESS_TOKEN")
		},
	}
	for _, p := range config.APIProviderDetectionOrder {
		if fn, ok := checks[p]; ok && fn() {
			return p
		}
	}
	return "anthropic"
}

// ResolveProviderModelEnvOverride resolves the model env override for a provider.
func ResolveProviderModelEnvOverride(provider string) string {
	if provider == "" {
		provider = DetectProvider()
	}
	for _, k := range config.ProviderModelEnvKeys[provider] {
		if v := resolveEnvSecret(k); v != "" {
			return v
		}
	}
	return ""
}

func resolveEnvSecret(envKey string) string {
	return credentials.LookupSecret(context.Background(), envKey)
}
