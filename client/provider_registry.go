package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	"deepseek":               {Name: "deepseek", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.deepseek.com/v1", EnvKey: "DEEPSEEK_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"grok":                   {Name: "grok", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.x.ai/v1", EnvKey: "XAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"openrouter":             {Name: "openrouter", Type: ProviderTypeOpenAICompatible, BaseURL: "https://openrouter.ai/api/v1", EnvKey: "OPENROUTER_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"zai_payg":               {Name: "zai_payg", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.z.ai/api/paas/v4", EnvKey: "ZAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"zai_coding":             {Name: "zai_coding", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.z.ai/api/coding/paas/v4", EnvKey: "ZAI_CODING_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"canopywave":             {Name: "canopywave", Type: ProviderTypeOpenAICompatible, BaseURL: "https://inference.canopywave.io/v1", EnvKey: "CANOPYWAVE_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"gemini":                 {Name: "gemini", Type: ProviderTypeOpenAICompatible, BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", EnvKey: "GEMINI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"ollama":                 {Name: "ollama", Type: ProviderTypeOpenAICompatible, BaseURL: "http://localhost:11434/v1", EnvKey: "OLLAMA_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: false},
	"opencodego":             {Name: "opencodego", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultOpenCodeGoBaseURL, EnvKey: "OPENCODEGO_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"kimi":                   {Name: "kimi", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultKimiOpenAIBaseURL, EnvKey: "MOONSHOT_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"xiaomi_mimo_payg":       {Name: "xiaomi_mimo_payg", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultXiaomiOpenAIBaseURL, EnvKey: config.EnvXiaomiPaygAPIKey, SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"xiaomi_mimo_token_plan": {Name: "xiaomi_mimo_token_plan", Type: ProviderTypeOpenAICompatible, BaseURL: "", EnvKey: config.EnvXiaomiTokenPlanAPIKey, SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"minimax_token_plan":     {Name: "minimax_token_plan", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.minimax.io/v1", EnvKey: "MINIMAX_TOKEN_PLAN_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"minimax_payg":           {Name: "minimax_payg", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.minimax.io/v1", EnvKey: "MINIMAX_PAYG_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
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
	// Gated on dynamicProviderEnvVar to prevent silent exfiltration of OPENAI_API_KEY
	// to an attacker-controlled OPENAI_API_BASE.
	if needsRegistration && dynamicProviderEnabled() {
		if fallbackURL := openaiBaseFallbackURL(); fallbackURL != "" {
			slog.Warn(
				"auto-registering OpenAI-compatible provider from OPENAI_API_BASE",
				"provider", providerName,
				"base_url", fallbackURL,
				"opt_in_env", dynamicProviderEnvVar,
			)
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
	}

	info := c.GetProviderInfo(providerName)
	if info == nil {
		return nil, fmt.Errorf("eyrie: unknown provider: %s", providerName)
	}
	baseURL := c.baseURLs[providerName]
	if baseURL == "" {
		baseURL = info.BaseURL
	}

	if apiKey == "" && providerName != "ollama" {
		return nil, fmt.Errorf("eyrie: no API key for %s; set %s or call SetAPIKey()", providerName, info.EnvKey)
	}

	var p Provider
	switch info.Type {
	case ProviderTypeAnthropic:
		p = NewAnthropicClient(apiKey, baseURL)
	case ProviderTypeAzure:
		endpoint := resolveEnvSecret("AZURE_OPENAI_ENDPOINT")
		if endpoint == "" {
			endpoint = baseURL
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
		if projectID == "" {
			return nil, fmt.Errorf("eyrie: vertex requires VERTEX_PROJECT_ID")
		}
		region := resolveEnvSecret("VERTEX_REGION")
		if region == "" {
			region = "us-central1"
		}
		p = NewVertexClient(projectID, region, apiKey)
	default:
		if config.IsZAIProvider(providerName) {
			providerCfg := config.LoadProviderConfig("")
			openAIBase, err := config.ResolveZAIOpenAIBase(providerName, providerCfg)
			if err != nil {
				return nil, err
			}
			anthropicBase := config.ResolveZAIAnthropicBase(providerCfg)
			p = NewZAIClient(apiKey, openAIBase, anthropicBase, info.Compat, providerName)
			break
		}
		if config.IsXiaomiMimoProvider(providerName) {
			providerCfg := config.LoadProviderConfig("")
			openAIBase, err := config.ResolveXiaomiOpenAIBase(providerName, providerCfg)
			if err != nil {
				return nil, err
			}
			anthropicBase, err := config.ResolveXiaomiAnthropicBase(providerName, providerCfg)
			if err != nil {
				return nil, err
			}
			p = NewMiMoClient(apiKey, openAIBase, anthropicBase, info.Compat, providerName)
			break
		}
		if providerName == "opencodego" {
			p = NewOpenCodeGoClient(apiKey, baseURL)
			break
		}
		p = NewOpenAIClient(apiKey, baseURL, info.Compat)
	}

	c.providers[providerName] = p
	return p, nil
}

// DetectProvider detects the active provider from the credential store (not process env).
func DetectProvider() string {
	ctx := context.Background()
	checks := map[string]func() bool{
		"anthropic":  func() bool { return credentials.HasSecret(ctx, "ANTHROPIC_API_KEY") },
		"deepseek":   func() bool { return credentials.HasSecret(ctx, "DEEPSEEK_API_KEY") },
		"openrouter": func() bool { return credentials.HasSecret(ctx, "OPENROUTER_API_KEY") },
		"grok":       func() bool { return credentials.HasSecret(ctx, "XAI_API_KEY") },
		"gemini":     func() bool { return credentials.HasSecret(ctx, "GEMINI_API_KEY") },
		"zai_payg":   func() bool { return credentials.HasSecret(ctx, "ZAI_API_KEY") },
		"zai_coding": func() bool { return credentials.HasSecret(ctx, "ZAI_CODING_API_KEY") },
		"canopywave": func() bool { return credentials.HasSecret(ctx, "CANOPYWAVE_API_KEY") },
		"openai":     func() bool { return credentials.HasSecret(ctx, "OPENAI_API_KEY") },
		"opencodego": func() bool { return credentials.HasSecret(ctx, "OPENCODEGO_API_KEY") },
		"kimi":       func() bool { return credentials.HasSecret(ctx, "MOONSHOT_API_KEY") },
		"xiaomi_mimo_payg": func() bool {
			return credentials.HasSecret(ctx, config.EnvXiaomiPaygAPIKey) || credentials.HasSecret(ctx, "XIAOMI_MIMO_API_KEY")
		},
		"xiaomi_mimo_token_plan": func() bool {
			return credentials.HasSecret(ctx, config.EnvXiaomiTokenPlanAPIKey)
		},
		"minimax_token_plan": func() bool {
			return credentials.HasSecret(ctx, "MINIMAX_TOKEN_PLAN_API_KEY")
		},
		"minimax_payg": func() bool {
			return credentials.HasSecret(ctx, "MINIMAX_PAYG_API_KEY")
		},
		"ollama": func() bool { return resolveEnvSecret("OLLAMA_BASE_URL") != "" },
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
	// Bound the lookup to prevent indefinite stalls when the OS keyring
	// is unresponsive (e.g., locked keychain on macOS, D-Bus failure on
	// Linux). The keyring itself has a 30s timeout, but resolveEnvSecret
	// is called multiple times in sequence during provider construction
	// (up to 6 calls for AWS Bedrock), so a per-call cap keeps the total
	// stall bounded.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return credentials.LookupSecret(ctx, envKey)
}
