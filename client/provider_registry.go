package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
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

// CoreProviders and OpenAICompatibleProviders are derived from the canonical
// catalog registry. Dynamic providers are added only to the compatible map.
var CoreProviders, OpenAICompatibleProviders = staticProviderMaps()

func staticProviderMaps() (map[string]ProviderRegistryConfig, map[string]ProviderRegistryConfig) {
	core := make(map[string]ProviderRegistryConfig)
	compatible := make(map[string]ProviderRegistryConfig)
	for _, spec := range registry.All() {
		providerType := ProviderTypeOpenAICompatible
		if spec.TransportKind != "" {
			providerType = ProviderType(spec.TransportKind)
		}
		baseURL := spec.RuntimeBaseURL
		if baseURL == "" {
			baseURL = spec.ProbeBaseURL
		}
		envKey := spec.RuntimeCredentialEnv
		if envKey == "" {
			envKey = spec.CredentialEnv
		}
		provider := ProviderRegistryConfig{
			Name: spec.ProviderID, Type: providerType, BaseURL: baseURL, EnvKey: envKey,
			SupportsStreaming: true, SupportsTools: true, SupportsReasoning: !spec.IsLocal,
		}
		if providerType == ProviderTypeOpenAICompatible {
			compatible[spec.ProviderID] = provider
		} else {
			core[spec.ProviderID] = provider
		}
	}
	return core, compatible
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
		"poolside":   func() bool { return credentials.HasSecret(ctx, "POOLSIDE_API_KEY") },
		"groq":       func() bool { return credentials.HasSecret(ctx, "GROQ_API_KEY") },
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
