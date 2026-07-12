package engine

import "time"

// Intent expresses a host's semantic preference without naming a provider.
type Intent string

const (
	IntentFast       Intent = "fast"
	IntentBalanced   Intent = "balanced"
	IntentReasoning  Intent = "reasoning"
	IntentEconomical Intent = "economical"
)

// ModelClass is a provider-neutral relative model cost/capability band.
type ModelClass string

const (
	ModelClassEconomical ModelClass = "economical"
	ModelClassBalanced   ModelClass = "balanced"
	ModelClassPremium    ModelClass = "premium"
)

// Requirements describes capabilities that a resolved model must support.
type Requirements struct {
	Streaming      bool `json:"streaming,omitempty"`
	Tools          bool `json:"tools,omitempty"`
	Vision         bool `json:"vision,omitempty"`
	StructuredJSON bool `json:"structured_json,omitempty"`
	Reasoning      bool `json:"reasoning,omitempty"`
	MinimumContext int  `json:"minimum_context,omitempty"`
}

// Preference contains optional selection policy supplied by the host/user.
type Preference struct {
	Intent            Intent  `json:"intent,omitempty"`
	PreferredModelID  string  `json:"preferred_model_id,omitempty"`
	PreferredProvider string  `json:"preferred_provider,omitempty"`
	AllowFallback     bool    `json:"allow_fallback,omitempty"`
	MaximumCostUSD    float64 `json:"maximum_cost_usd,omitempty"`
}

// ContentPart is a provider-neutral multimodal message part.
type ContentPart struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	URL         string `json:"url,omitempty"`
	Detail      string `json:"detail,omitempty"`
	AudioData   string `json:"audio_data,omitempty"`
	AudioFormat string `json:"audio_format,omitempty"`
}

// ToolCall is a normalized tool invocation.
type ToolCall struct {
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult is a host-executed tool result sent back to a model.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Message is the stable conversation DTO at the host boundary.
type Message struct {
	Role         string        `json:"role"`
	Content      string        `json:"content,omitempty"`
	Thinking     string        `json:"thinking,omitempty"`
	ContentParts []ContentPart `json:"content_parts,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolResults  []ToolResult  `json:"tool_results,omitempty"`
}

// Tool is a model-visible tool definition. Eyrie emits requests for these
// tools; the host remains responsible for permission checks and execution.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolChoice controls whether and how the model may request tools.
type ToolChoice struct {
	Type                   string `json:"type"`
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

// GenerationOptions contains model-generation controls that have equivalent
// semantics across one or more provider adapters. Provider-specific wire
// formats remain internal to Eyrie.
type GenerationOptions struct {
	EnableCaching        bool              `json:"enable_caching,omitempty"`
	ReasoningEffort      string            `json:"reasoning_effort,omitempty"`
	ThinkingBudgetTokens int               `json:"thinking_budget_tokens,omitempty"`
	ThinkingMode         string            `json:"thinking_mode,omitempty"`
	ThinkingDisplay      string            `json:"thinking_display,omitempty"`
	GLMThinkingEnabled   *bool             `json:"glm_thinking_enabled,omitempty"`
	VirtualKeyID         string            `json:"virtual_key_id,omitempty"`
	KimiContextCacheID   string            `json:"kimi_context_cache_id,omitempty"`
	KimiCacheResetTTL    bool              `json:"kimi_cache_reset_ttl,omitempty"`
	TopP                 *float64          `json:"top_p,omitempty"`
	TopK                 *int              `json:"top_k,omitempty"`
	StopSequences        []string          `json:"stop_sequences,omitempty"`
	ToolChoice           *ToolChoice       `json:"tool_choice,omitempty"`
	ServiceTier          string            `json:"service_tier,omitempty"`
	OutputEffort         string            `json:"output_effort,omitempty"`
	PresencePenalty      *float64          `json:"presence_penalty,omitempty"`
	FrequencyPenalty     *float64          `json:"frequency_penalty,omitempty"`
	N                    *int              `json:"n,omitempty"`
	LogProbs             *bool             `json:"logprobs,omitempty"`
	TopLogProbs          *int              `json:"top_logprobs,omitempty"`
	Seed                 *int              `json:"seed,omitempty"`
	Store                *bool             `json:"store,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Modalities           []string          `json:"modalities,omitempty"`
	AudioConfig          string            `json:"audio_config,omitempty"`
	Prediction           string            `json:"prediction,omitempty"`
	WebSearchOptions     string            `json:"web_search_options,omitempty"`
}

// Limits bounds a generation request.
type Limits struct {
	MaxOutputTokens      int           `json:"max_output_tokens,omitempty"`
	MaxContinuations     int           `json:"max_continuations,omitempty"`
	MaxTotalOutputTokens int           `json:"max_total_output_tokens,omitempty"`
	Timeout              time.Duration `json:"timeout,omitempty"`
}

// Metadata provides stable correlation identifiers. Providers receive only
// fields that their adapter explicitly supports.
type Metadata struct {
	SessionID string `json:"session_id,omitempty"`
	TurnID    string `json:"turn_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

// GenerateRequest is the provider-neutral request accepted by Engine.
type GenerateRequest struct {
	Messages     []Message         `json:"messages"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
	Tools        []Tool            `json:"tools,omitempty"`
	Requirements Requirements      `json:"requirements,omitempty"`
	Preference   Preference        `json:"preference,omitempty"`
	Limits       Limits            `json:"limits,omitempty"`
	Metadata     Metadata          `json:"metadata,omitempty"`
	Temperature  *float64          `json:"temperature,omitempty"`
	OutputSchema string            `json:"output_schema,omitempty"`
	Options      GenerationOptions `json:"options,omitempty"`
}

// Route is the concrete model/deployment decision made by Eyrie.
type Route struct {
	Provider          string `json:"provider"`
	Model             string `json:"model"`
	DeploymentRouting bool   `json:"deployment_routing"`
}

// Usage is normalized token accounting.
type Usage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	TotalTokens         int `json:"total_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
	ThinkingTokens      int `json:"thinking_tokens,omitempty"`
}

// GenerateResponse is a normalized blocking response.
type GenerateResponse struct {
	Content      string     `json:"content"`
	Thinking     string     `json:"thinking,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
	RequestID    string     `json:"request_id,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	Route        Route      `json:"route"`
}

// EventType is a stable stream event name.
type EventType string

const (
	EventRouteSelected EventType = "route_selected"
	EventContentDelta  EventType = "content_delta"
	EventThinkingDelta EventType = "thinking_delta"
	EventToolCallStart EventType = "tool_call_start"
	EventToolCallDelta EventType = "tool_call_delta"
	EventToolCallDone  EventType = "tool_call_done"
	EventUsage         EventType = "usage"
	EventRetry         EventType = "retry"
	EventContinuation  EventType = "continuation"
	EventRouteChanged  EventType = "route_changed"
	EventWarning       EventType = "warning"
	EventTTFT          EventType = "ttft"
	EventDone          EventType = "done"
)

// Event is a normalized model stream event. New optional fields and event
// types are additive; hosts must safely ignore unknown event types.
type Event struct {
	Type       EventType `json:"type"`
	Content    string    `json:"content,omitempty"`
	Thinking   string    `json:"thinking,omitempty"`
	ToolCall   *ToolCall `json:"tool_call,omitempty"`
	Usage      *Usage    `json:"usage,omitempty"`
	Route      *Route    `json:"route,omitempty"`
	Warning    string    `json:"warning,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	StopReason string    `json:"stop_reason,omitempty"`
	TTFTMillis int       `json:"ttft_ms,omitempty"`
}

// Model is a host-facing catalog row.
type Model struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"display_name"`
	Description      string   `json:"description,omitempty"`
	Owner            string   `json:"owner,omitempty"`
	ProviderID       string   `json:"provider_id"`
	GatewayID        string   `json:"gateway_id,omitempty"`
	ContextWindow    int      `json:"context_window,omitempty"`
	MaxOutputTokens  int      `json:"max_output_tokens,omitempty"`
	InputPricePer1M  float64  `json:"input_price_per_1m,omitempty"`
	OutputPricePer1M float64  `json:"output_price_per_1m,omitempty"`
	PriceKnown       bool     `json:"price_known"`
	Capabilities     []string `json:"capabilities,omitempty"`
	Source           string   `json:"source,omitempty"`
}

// CatalogSnapshot is an immutable host-facing view of a loaded catalog.
type CatalogSnapshot struct {
	Models    []Model   `json:"models"`
	CachePath string    `json:"cache_path,omitempty"`
	Stale     bool      `json:"stale,omitempty"`
	LoadedAt  time.Time `json:"loaded_at"`
}

// CredentialStatus is safe to render or log; it never contains a secret.
type CredentialStatus struct {
	ProviderID string `json:"provider_id"`
	EnvVar     string `json:"env_var,omitempty"`
	Configured bool   `json:"configured"`
	Verified   bool   `json:"verified,omitempty"`
	Masked     string `json:"masked,omitempty"`
}

// CredentialProvider is safe setup metadata for a configurable provider.
// It contains identifiers and labels only, never credential material.
type CredentialProvider struct {
	ProviderID   string `json:"provider_id"`
	DeploymentID string `json:"deployment_id,omitempty"`
	EnvVar       string `json:"env_var,omitempty"`
	DisplayName  string `json:"display_name"`
	RequiresKey  bool   `json:"requires_key"`
	Rank         int    `json:"rank,omitempty"`
}

// CredentialResolution validates pasted input and returns provider choices.
// The input secret is never retained in this value.
type CredentialResolution struct {
	FormatOK                bool                 `json:"format_ok"`
	FormatError             string               `json:"format_error,omitempty"`
	Providers               []CredentialProvider `json:"providers"`
	ProbeDisambiguationUsed bool                 `json:"probe_disambiguation_used,omitempty"`
}

// Gateway is one host-facing provider/deployment configuration row.
type Gateway struct {
	ID                   string `json:"id"`
	DisplayName          string `json:"display_name"`
	DeploymentID         string `json:"deployment_id,omitempty"`
	CredentialEnv        string `json:"credential_env,omitempty"`
	RequiresKey          bool   `json:"requires_key"`
	CredentialConfigured bool   `json:"credential_configured"`
	DeploymentConfigured bool   `json:"deployment_configured"`
	ModelCount           int    `json:"model_count"`
	Active               bool   `json:"active"`
	RegionLabel          string `json:"region_label,omitempty"`
	RegionRequired       bool   `json:"region_required,omitempty"`
}

// Selection is the effective provider/model state supplied to a host session.
type Selection struct {
	Provider                string `json:"provider"`
	Model                   string `json:"model"`
	HasConfiguredDeployment bool   `json:"has_configured_deployment"`
	DeploymentRouting       bool   `json:"deployment_routing"`
}

// SelectionOptions contains optional user or command-line overrides.
type SelectionOptions struct {
	ProviderOverride          string `json:"provider_override,omitempty"`
	ModelOverride             string `json:"model_override,omitempty"`
	DeploymentRoutingOverride *bool  `json:"-"`
}
