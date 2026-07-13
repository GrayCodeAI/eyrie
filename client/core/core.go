// Package core holds the provider contract and the data types shared by
// every layer of the eyrie client: adapters, middleware, caching, embeddings,
// and the client facade itself.
//
// core is a leaf package — it must not import any other eyrie/client
// subpackage. The public names remain available as aliases in
// github.com/GrayCodeAI/eyrie/client, which is the API consumers should keep
// importing; this package exists so subpackages can share the contract
// without an import cycle through the facade.
//
// See plans/client-package-decomposition.md for the migration plan.
package core

import "context"

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
	Thinking     string        `json:"thinking,omitempty"`      // chain-of-thought captured from a prior response (never forwarded to providers that reject it)
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
	ThinkingTokens      int `json:"thinking_tokens,omitempty"`
}

// EyrieResponse is the response from a chat call.
type EyrieResponse struct {
	Content        string      `json:"content"`
	Thinking       string      `json:"thinking,omitempty"`
	Usage          *EyrieUsage `json:"usage,omitempty"`
	ToolCalls      []ToolCall  `json:"tool_calls,omitempty"`
	FinishReason   string      `json:"finish_reason"`
	RequestID      string      `json:"request_id,omitempty"`
	OrganizationID string      `json:"organization_id,omitempty"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// EyrieStreamEvent is a streaming event.
type EyrieStreamEvent struct {
	Type       string      `json:"type"` // content, tool_call, tool_input_delta, thinking, done, error, ttft
	Content    string      `json:"content,omitempty"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	Thinking   string      `json:"thinking,omitempty"`
	Error      string      `json:"error,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
	Usage      *EyrieUsage `json:"usage,omitempty"`
	StopReason string      `json:"stop_reason,omitempty"`
	TTFTms     int         `json:"ttft_ms,omitempty"` // time-to-first-token milliseconds, set on "done" event
	// TTFT carries time-to-first-token milliseconds on Type=="ttft" events.
	// It is emitted as a dedicated event immediately before the first content
	// or tool-call delta so consumers can measure latency without waiting for
	// the stream to finish.
	TTFT int `json:"ttft,omitempty"`
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

// NewStreamResultWithRequestID is NewStreamResult carrying the provider's request ID.
func NewStreamResultWithRequestID(events <-chan EyrieStreamEvent, requestID string, cancel context.CancelFunc) *StreamResult {
	return &StreamResult{Events: events, RequestID: requestID, cancel: cancel}
}

// ResponseFormat specifies the desired output format for the model response.
type ResponseFormat struct {
	Type   string `json:"type"`             // "json_object" or "json_schema"
	Schema string `json:"schema,omitempty"` // optional JSON schema for structured output
}

// ChatOptions holds options for a chat request.
type ChatOptions struct {
	Provider       string          `json:"provider,omitempty"`
	Model          string          `json:"model,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	Tools          []EyrieTool     `json:"tools,omitempty"`
	System         string          `json:"system,omitempty"`
	EnableCaching  bool            `json:"enable_caching,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	// ReasoningEffort hints how much reasoning the model should perform.
	// Valid values are "low", "medium", or "high" (see ReasoningLow/Medium/High).
	// Only applied for OpenAI-compatible providers whose compat config sets
	// SupportsReasoningEffort.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// ThinkingBudgetTokens enables Anthropic extended thinking with the given
	// token budget when greater than zero. Ignored by other providers.
	ThinkingBudgetTokens int `json:"thinking_budget_tokens,omitempty"`
	// ThinkingMode controls Anthropic thinking behavior: "enabled", "adaptive", or "disabled".
	// When set, takes precedence over ThinkingBudgetTokens (except "enabled" which uses the budget).
	ThinkingMode string `json:"thinking_mode,omitempty"`
	// ThinkingDisplay controls how thinking is shown: "summarized" (default) or "omitted".
	ThinkingDisplay string `json:"thinking_display,omitempty"`
	// GLMThinkingEnabled toggles GLM/Z.ai extended reasoning via the provider's
	// non-OpenAI thinking={"type":"enabled"|"disabled"} request parameter. Only
	// applied for OpenAI-compatible providers whose compat config sets
	// ThinkingFormat to "zai". When nil the parameter is omitted and the model
	// uses its default (GLM defaults to enabled). Ignored by other providers.
	GLMThinkingEnabled *bool `json:"glm_thinking_enabled,omitempty"`
	// VirtualKeyID optionally attributes the request to a logical virtual key
	// for budget enforcement and cost accounting (see BudgetProvider). When
	// empty, the BudgetProvider also checks the request context.
	VirtualKeyID string `json:"virtual_key_id,omitempty"`
	// KimiContextCacheID, when set, prepends a cache-role message to the
	// request for Kimi/Moonshot context caching. Only effective when the
	// provider compat is KimiCompat (SupportsCacheRole true).
	KimiContextCacheID string `json:"kimi_context_cache_id,omitempty"`
	// KimiCacheResetTTL resets the TTL of the cache on use when true.
	// Only effective when KimiContextCacheID is also set.
	KimiCacheResetTTL bool `json:"kimi_cache_reset_ttl,omitempty"`

	// Shared parameters (Anthropic + OpenAI)
	TopP           *float64          `json:"top_p,omitempty"`            // nucleus sampling (0.0-1.0)
	TopK           *int              `json:"top_k,omitempty"`            // top-K sampling (Anthropic only)
	StopSequences  []string          `json:"stop_sequences,omitempty"`   // custom stop sequences
	ToolChoice     *ToolChoiceOption `json:"tool_choice,omitempty"`      // tool use control
	MetadataUserID string            `json:"metadata_user_id,omitempty"` // user ID for abuse detection / monitoring
	ServiceTier    string            `json:"service_tier,omitempty"`     // "auto", "default", "flex", "priority"
	OutputEffort   string            `json:"output_effort,omitempty"`    // "low","medium","high","xhigh","max" (Anthropic)
	OutputSchema   string            `json:"output_schema,omitempty"`    // JSON schema string for structured output (Anthropic)

	// OpenAI-specific parameters
	PresencePenalty  *float64          `json:"presence_penalty,omitempty"`   // -2.0 to 2.0
	FrequencyPenalty *float64          `json:"frequency_penalty,omitempty"`  // -2.0 to 2.0
	N                *int              `json:"n,omitempty"`                  // number of completions (1-128)
	LogProbs         *bool             `json:"logprobs,omitempty"`           // return log probabilities
	TopLogProbs      *int              `json:"top_logprobs,omitempty"`       // 0-20, requires logprobs=true
	Seed             *int              `json:"seed,omitempty"`               // deterministic sampling
	Store            *bool             `json:"store,omitempty"`              // store output for Responses API
	Metadata         map[string]string `json:"metadata,omitempty"`           // developer tags
	Modalities       []string          `json:"modalities,omitempty"`         // "text", "audio"
	AudioConfig      string            `json:"audio_config,omitempty"`       // JSON: {voice, format}
	Prediction       string            `json:"prediction,omitempty"`         // JSON: {type:"content", content:"..."}
	WebSearchOptions string            `json:"web_search_options,omitempty"` // JSON: {search_context_size, user_location}
}

// ToolChoiceOption controls how the model uses tools (Anthropic).
type ToolChoiceOption struct {
	Type                   string `json:"type"`           // "auto", "any", "tool", "none"
	Name                   string `json:"name,omitempty"` // required when type="tool"
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

// ContinuationConfig controls output continuation behavior.
type ContinuationConfig struct {
	// MaxContinuations is the maximum number of continuation calls (default 3).
	MaxContinuations int
	// MaxTotalTokens caps the total output tokens across all continuations (0 = unlimited).
	MaxTotalTokens int
}

// DefaultContinuationConfig returns sensible defaults.
func DefaultContinuationConfig() ContinuationConfig {
	return ContinuationConfig{MaxContinuations: 3, MaxTotalTokens: 32000}
}
