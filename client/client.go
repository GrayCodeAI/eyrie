// Package client provides LLM provider clients for Anthropic, OpenAI,
// and OpenAI-compatible APIs with streaming, retry, and provider detection.
package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
)

// Version is set by the root package and used in User-Agent headers.
var Version = "0.0.1"

// userAgent returns the User-Agent string for HTTP requests.
func userAgent() string { return "eyrie/" + Version }

// Provider is the core interface for LLM providers.
// Implementations must be safe for concurrent use.
type Provider interface {
	// Chat sends a non-streaming chat request.
	Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error)
	// StreamChat sends a streaming chat request.
	// The caller must call Close() on the returned StreamResult when done.
	StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error)
	// Ping checks connectivity and authentication.
	Ping(ctx context.Context) error
	// Name returns the provider name (e.g. "anthropic", "openai").
	Name() string
}

// EyrieConfig holds client configuration.
type EyrieConfig struct {
	Provider   string `json:"provider,omitempty"`
	APIKey     string `json:"api_key,omitempty"`
	BaseURL    string `json:"base_url,omitempty"`
	Model      string `json:"model,omitempty"`
	MaxRetries int    `json:"max_retries,omitempty"`
}

// EyrieMessage represents a chat message.
type EyrieMessage struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	Images     []string    `json:"images,omitempty"`
	ToolUse    []ToolCall  `json:"tool_use,omitempty"`    // assistant message with tool calls
	ToolResult *ToolResult `json:"tool_result,omitempty"` // user message with tool result
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// EyrieTool represents a tool definition.
type EyrieTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// EyrieUsage tracks token usage.
type EyrieUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EyrieResponse is the response from a chat call.
type EyrieResponse struct {
	Content      string      `json:"content"`
	Usage        *EyrieUsage `json:"usage,omitempty"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	FinishReason string      `json:"finish_reason"`
	RequestID    string      `json:"request_id,omitempty"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// EyrieStreamEvent is a streaming event.
type EyrieStreamEvent struct {
	Type       string      `json:"type"` // content, tool_call, tool_input_delta, thinking, done, error
	Content    string      `json:"content,omitempty"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	Thinking   string      `json:"thinking,omitempty"`
	Error      string      `json:"error,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
	Usage      *EyrieUsage `json:"usage,omitempty"`
	StopReason string      `json:"stop_reason,omitempty"`
}

// StreamResult wraps a streaming response with cleanup.
// Callers must call Close() when done reading events, or cancel the context.
type StreamResult struct {
	Events    <-chan EyrieStreamEvent
	RequestID string
	cancel    context.CancelFunc
}

// Close stops the stream and releases resources.
func (sr *StreamResult) Close() {
	if sr.cancel != nil {
		sr.cancel()
	}
}

// ProviderType classifies providers.
type ProviderType string

const (
	// ProviderTypeAnthropic uses the Anthropic Messages API.
	ProviderTypeAnthropic ProviderType = "anthropic"
	// ProviderTypeOpenAI uses the OpenAI Chat Completions API.
	ProviderTypeOpenAI ProviderType = "openai"
	// ProviderTypeOpenAICompatible uses OpenAI-compatible APIs with custom base URLs.
	ProviderTypeOpenAICompatible ProviderType = "openai-compatible"
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
var CoreProviders = map[string]ProviderRegistryConfig{
	"anthropic": {Name: "anthropic", Type: ProviderTypeAnthropic, EnvKey: "ANTHROPIC_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"openai":    {Name: "openai", Type: ProviderTypeOpenAI, BaseURL: "https://api.openai.com/v1", EnvKey: "OPENAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
}

// OpenAICompatibleProviders use the OpenAI SDK with custom baseUrl.
var OpenAICompatibleProviders = map[string]ProviderRegistryConfig{
	"grok":       {Name: "grok", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.x.ai/v1", EnvKey: "XAI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"openrouter": {Name: "openrouter", Type: ProviderTypeOpenAICompatible, BaseURL: "https://openrouter.ai/api/v1", EnvKey: "OPENROUTER_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"canopywave": {Name: "canopywave", Type: ProviderTypeOpenAICompatible, BaseURL: "https://inference.canopywave.io/v1", EnvKey: "CANOPYWAVE_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"gemini":     {Name: "gemini", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.gemini.google.com/v1/forward", EnvKey: "GEMINI_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
	"ollama":     {Name: "ollama", Type: ProviderTypeOpenAICompatible, BaseURL: "http://localhost:11434/v1", EnvKey: "OLLAMA_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: false},
	"opencodego": {Name: "opencodego", Type: ProviderTypeOpenAICompatible, BaseURL: config.DefaultOpenCodeGoBaseURL, EnvKey: "OPENCODEGO_API_KEY", SupportsStreaming: true, SupportsTools: true, SupportsReasoning: true},
}

// EyrieClient is the universal LLM client.
// It is safe for concurrent use.
type EyrieClient struct {
	mu              sync.RWMutex
	defaultProvider string
	apiKeys         map[string]string
	providers       map[string]Provider // cached provider clients
}

// NewEyrieClient creates a new EyrieClient.
func NewEyrieClient(cfg *EyrieConfig) *EyrieClient {
	c := &EyrieClient{
		defaultProvider: "openai",
		apiKeys:         make(map[string]string),
		providers:       make(map[string]Provider),
	}
	if cfg != nil {
		if cfg.Provider != "" {
			c.defaultProvider = cfg.Provider
		}
		if cfg.APIKey != "" {
			c.apiKeys[c.defaultProvider] = cfg.APIKey
		}
	}
	return c
}

// SetAPIKey sets an API key for a provider.
func (c *EyrieClient) SetAPIKey(provider, apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKeys[provider] = apiKey
	delete(c.providers, provider) // invalidate cached client
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
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if p, ok := c.providers[providerName]; ok {
		return p, nil
	}

	apiKey := c.apiKeys[providerName]
	if apiKey == "" {
		info := c.GetProviderInfo(providerName)
		if info == nil {
			// Fallback: if OPENAI_API_BASE or OPENAI_BASE_URL is set, register
			// an ad-hoc OpenAI-compatible provider so unknown names still work.
			if fallbackURL := openaiBaseFallbackURL(); fallbackURL != "" {
				_ = RegisterDynamicProvider(providerName, fallbackURL, "OPENAI_API_KEY")
			} else {
				return nil, fmt.Errorf("eyrie: unknown provider: %s", providerName)
			}
		}
		// Re-check after potential dynamic registration.
		info = c.GetProviderInfo(providerName)
		if info == nil {
			return nil, fmt.Errorf("eyrie: unknown provider: %s", providerName)
		}
		apiKey = os.Getenv(info.EnvKey)
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
	default:
		p = NewOpenAIClient(apiKey, info.BaseURL, info.Compat)
	}

	c.providers[providerName] = p
	return p, nil
}

// Chat sends a chat request to the specified (or default) provider.
func (c *EyrieClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("eyrie: messages must not be empty")
	}
	provider := opts.Provider
	if provider == "" {
		provider = c.defaultProvider
	}
	p, err := c.getOrCreateProvider(provider)
	if err != nil {
		return nil, err
	}
	if opts.Model == "" {
		opts.Model = ResolveDefaultModel(provider)
	}
	return p.Chat(ctx, messages, opts)
}

// StreamChat sends a streaming chat request.
func (c *EyrieClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("eyrie: messages must not be empty")
	}
	provider := opts.Provider
	if provider == "" {
		provider = c.defaultProvider
	}
	p, err := c.getOrCreateProvider(provider)
	if err != nil {
		return nil, err
	}
	if opts.Model == "" {
		opts.Model = ResolveDefaultModel(provider)
	}
	return p.StreamChat(ctx, messages, opts)
}

// Ping checks connectivity to the specified (or default) provider.
func (c *EyrieClient) Ping(ctx context.Context, provider string) error {
	if provider == "" {
		provider = c.defaultProvider
	}
	p, err := c.getOrCreateProvider(provider)
	if err != nil {
		return err
	}
	return p.Ping(ctx)
}

// AnthropicClientConfig holds config for creating an Anthropic client.
type AnthropicClientConfig struct {
	APIKey         string            `json:"api_key,omitempty"`
	DefaultHeaders map[string]string `json:"default_headers,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"max_retries,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
}

// DetectProvider detects the active provider from env vars.
func DetectProvider() string {
	checks := map[string]func() bool{
		"anthropic":  func() bool { return os.Getenv("ANTHROPIC_API_KEY") != "" },
		"openrouter": func() bool { return os.Getenv("OPENROUTER_API_KEY") != "" },
		"grok":       func() bool { return os.Getenv("GROK_API_KEY") != "" || os.Getenv("XAI_API_KEY") != "" },
		"gemini":     func() bool { return os.Getenv("GEMINI_API_KEY") != "" },
		"canopywave": func() bool { return os.Getenv("CANOPYWAVE_API_KEY") != "" },
		"openai":     func() bool { return os.Getenv("OPENAI_API_KEY") != "" },
		"opencodego": func() bool { return os.Getenv("OPENCODEGO_API_KEY") != "" },
		"ollama":     func() bool { return os.Getenv("OLLAMA_BASE_URL") != "" },
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
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// ParseCustomHeaders parses GRAYCODE_CUSTOM_HEADERS env var into a map.
func ParseCustomHeaders() map[string]string {
	result := make(map[string]string)
	raw := os.Getenv("GRAYCODE_CUSTOM_HEADERS")
	if raw == "" {
		return result
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ":"); idx > 0 {
			result[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+1:])
		}
	}
	return result
}

// ResolveDefaultModel resolves the default model for a provider from the catalog.
func ResolveDefaultModel(provider string) string {
	cat := catalog.LoadModelCatalogSync("")
	models := catalog.ModelsForProvider(&cat, provider)
	if len(models) > 0 {
		return models[0].ID
	}
	return ""
}
