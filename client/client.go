// Package client provides LLM provider clients for Anthropic, OpenAI,
// and OpenAI-compatible APIs with streaming, retry, and provider detection.
package client

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/GrayCodeAI/eyrie/catalog"
)

// Version is exported here for backwards compatibility. Callers should prefer
// the canonical eyrie.Version (which is sourced from the repo-root VERSION
// file). This variable is initialised by the root package via SetVersion to
// avoid a circular import.
var Version = "dev"

// SetVersion is called by the root eyrie package's init to wire the canonical
// version into this sub-package without creating an import cycle.
func SetVersion(v string) { Version = v }

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
	APIKey     string `json:"-"`
	BaseURL    string `json:"base_url,omitempty"`
	Model      string `json:"model,omitempty"`
	MaxRetries int    `json:"max_retries,omitempty"`
}

// ContentPart represents a piece of content in a multi-modal message.
// Use the helper types (TextPart, ImagePart, AudioPart) to construct these.
type ContentPart struct {
	Type       string          `json:"type"`                  // "text", "image_url", "input_audio"
	Text       string          `json:"text,omitempty"`        // for type="text"
	ImageURL   *ImageURLPart   `json:"image_url,omitempty"`   // for type="image_url"
	InputAudio *InputAudioPart `json:"input_audio,omitempty"` // for type="input_audio"
}

// ImageURLPart represents an image content part.
// URL can be an HTTP(S) URL or a data URI (data:image/png;base64,...).
type ImageURLPart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high" (OpenAI-specific)
}

// InputAudioPart represents an audio content part (base64 encoded).
type InputAudioPart struct {
	Data   string `json:"data"`   // base64 encoded audio
	Format string `json:"format"` // "wav", "mp3" (OpenAI) or used to derive media_type (Anthropic)
}

// EyrieMessage represents a chat message.
// For simple text messages, set Content directly.
// For multi-modal messages (images, audio), use ContentParts.
// When ContentParts is non-empty, it takes precedence over Content and Images.
// The Images field is retained for backward compatibility.
type EyrieMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content,omitempty"`
	ContentParts []ContentPart `json:"content_parts,omitempty"` // multi-modal content (takes precedence over Content/Images)
	Images       []string      `json:"images,omitempty"`
	ToolUse      []ToolCall    `json:"tool_use,omitempty"`     // assistant message with tool calls
	ToolResults  []ToolResult  `json:"tool_results,omitempty"` // user message with one or more tool results
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
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
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

// NewStreamResult creates a StreamResult with a cancel function for resource cleanup.
func NewStreamResult(events <-chan EyrieStreamEvent, cancel context.CancelFunc) *StreamResult {
	return &StreamResult{Events: events, cancel: cancel}
}

// EyrieClient is the universal LLM client.
// It is safe for concurrent use.
type EyrieClient struct {
	mu              sync.RWMutex
	defaultProvider string
	apiKeys         map[string]string
	baseURLs        map[string]string
	providers       map[string]Provider // cached provider clients
	coalescer       *Coalescer          // optional request coalescing
}

// Client creates an EyrieClient.
func Client(cfg *EyrieConfig, opts ...ClientOption) *EyrieClient {
	c := &EyrieClient{
		defaultProvider: DetectProvider(),
		apiKeys:         make(map[string]string),
		baseURLs:        make(map[string]string),
		providers:       make(map[string]Provider),
	}
	if cfg != nil {
		if cfg.Provider != "" {
			c.defaultProvider = cfg.Provider
		}
		if cfg.APIKey != "" {
			c.apiKeys[c.defaultProvider] = cfg.APIKey
		}
		if cfg.BaseURL != "" {
			c.baseURLs[c.defaultProvider] = cfg.BaseURL
		}
	}
	// Apply options (including coalescing)
	for _, opt := range opts {
		opt.applyEyrie(c)
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
	APIKey         string            `json:"-"`
	DefaultHeaders map[string]string `json:"default_headers,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	MaxRetries     int               `json:"max_retries,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	BaseURL        string            `json:"base_url,omitempty"`
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
			name := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			// Reject header names/values containing control characters to prevent injection.
			if strings.ContainsAny(name, "\r\n") || strings.ContainsAny(value, "\r\n") {
				continue
			}
			result[name] = value
		}
	}
	return result
}

var (
	cachedCatalog   *catalog.CompiledCatalogV1
	catalogLoadOnce sync.Once
)

// NewImageMessage creates a user message with an image from a URL or data URI.
// The url parameter accepts HTTP(S) URLs or data URIs (data:image/png;base64,...).
func NewImageMessage(url string) EyrieMessage {
	return EyrieMessage{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "image_url", ImageURL: &ImageURLPart{URL: url}},
		},
	}
}

// NewImageMessageWithText creates a user message with text and an image from a URL or data URI.
func NewImageMessageWithText(text, url string) EyrieMessage {
	return EyrieMessage{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "image_url", ImageURL: &ImageURLPart{URL: url}},
		},
	}
}

// NewBase64ImageMessage creates a user message with a base64-encoded image.
// mediaType should be a MIME type like "image/png" or "image/jpeg".
func NewBase64ImageMessage(data, mediaType string) EyrieMessage {
	return EyrieMessage{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "image_url", ImageURL: &ImageURLPart{
				URL: "data:" + mediaType + ";base64," + data,
			}},
		},
	}
}

// NewBase64ImageMessageWithText creates a user message with text and a base64-encoded image.
func NewBase64ImageMessageWithText(text, data, mediaType string) EyrieMessage {
	return EyrieMessage{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "image_url", ImageURL: &ImageURLPart{
				URL: "data:" + mediaType + ";base64," + data,
			}},
		},
	}
}

// NewAudioMessage creates a user message with base64-encoded audio.
// format should be "wav" or "mp3".
func NewAudioMessage(base64Data, format string) EyrieMessage {
	return EyrieMessage{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "input_audio", InputAudio: &InputAudioPart{
				Data:   base64Data,
				Format: format,
			}},
		},
	}
}

// NewAudioMessageWithText creates a user message with text and base64-encoded audio.
func NewAudioMessageWithText(text, base64Data, format string) EyrieMessage {
	return EyrieMessage{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "input_audio", InputAudio: &InputAudioPart{
				Data:   base64Data,
				Format: format,
			}},
		},
	}
}

// ResolveDefaultModel resolves the default model for a provider from the catalog.
func ResolveDefaultModel(provider string) string {
	catalogLoadOnce.Do(func() {
		cat, err := catalog.LoadCatalogV1(context.Background(), catalog.LoadCatalogV1Options{})
		if err == nil {
			cachedCatalog = cat
		}
	})
	if cachedCatalog == nil {
		return ""
	}
	models := catalog.ModelEntriesForProvider(cachedCatalog, provider)
	if len(models) > 0 {
		return models[0].ID
	}
	return ""
}
