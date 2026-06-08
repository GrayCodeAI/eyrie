package client

import (
	"log/slog"
	"net/http"
	"time"
)

const defaultTimeout = 10 * time.Minute

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
}

// ClientOption configures clients.
type ClientOption struct {
	applyFn       func(*AnthropicClient)
	applyOpenAIFn func(*OpenAIClient)
	applyEyrieFn  func(*EyrieClient)

	// Structured output configuration (used by WithStructuredOutput)
	structuredSchema     map[string]interface{}
	structuredMaxRetries int
	structuredSchemaJSON string
}

func (o ClientOption) apply(c *AnthropicClient) {
	if o.applyFn != nil {
		o.applyFn(c)
	}
}

func (o ClientOption) applyOpenAI(c *OpenAIClient) {
	if o.applyOpenAIFn != nil {
		o.applyOpenAIFn(c)
	}
}

func (o ClientOption) applyEyrie(c *EyrieClient) {
	if o.applyEyrieFn != nil {
		o.applyEyrieFn(c)
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.httpClient.Timeout = d },
		applyOpenAIFn: func(c *OpenAIClient) { c.httpClient.Timeout = d },
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.httpClient = hc },
		applyOpenAIFn: func(c *OpenAIClient) { c.httpClient = hc },
	}
}

// WithRetry sets retry configuration.
func WithRetry(rc RetryConfig) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.retry = rc },
		applyOpenAIFn: func(c *OpenAIClient) { c.retry = rc },
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.logger = l },
		applyOpenAIFn: func(c *OpenAIClient) { c.logger = l },
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.apiKey = key },
		applyOpenAIFn: func(c *OpenAIClient) { c.apiKey = key },
	}
}

// WithBaseURL sets the base URL.
func WithBaseURL(url string) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.baseURL = url },
		applyOpenAIFn: func(c *OpenAIClient) { c.baseURL = url },
	}
}

// WithModel sets the default model for requests.
func WithModel(model string) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.defaultModel = model },
		applyOpenAIFn: func(c *OpenAIClient) { c.defaultModel = model },
	}
}

// WithMaxTokens sets the default max tokens for requests.
func WithMaxTokens(n int) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.defaultMaxTokens = n },
		applyOpenAIFn: func(c *OpenAIClient) { c.defaultMaxTokens = n },
	}
}

// WithTemperature sets the default temperature for requests.
func WithTemperature(t float64) ClientOption {
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.defaultTemperature = &t },
		applyOpenAIFn: func(c *OpenAIClient) { c.defaultTemperature = &t },
	}
}

// WithCoalescing enables request coalescing for identical concurrent requests.
// When enabled, multiple goroutines sending identical requests (same provider,
// model, messages, temperature, max_tokens) will be deduplicated into a single
// API call, with the result broadcast to all waiters.
//
// The ttl parameter controls how long completed requests remain in the coalescer
// for potential reuse. A typical value is 100-500ms.
func WithCoalescing(ttl time.Duration) ClientOption {
	return ClientOption{
		applyEyrieFn: func(c *EyrieClient) {
			c.coalescer = NewCoalescer(ttl)
		},
	}
}

// WithGuardrails attaches output guardrails to the client. Guardrails run
// after the LLM response but before returning to the caller. Blocked
// responses are replaced with an error; redacted responses have matches
// replaced with asterisks.
func WithGuardrails(rules ...GuardrailRule) ClientOption {
	g := NewGuardrails(rules...)
	return ClientOption{
		applyFn:       func(c *AnthropicClient) { c.guardrails = g },
		applyOpenAIFn: func(c *OpenAIClient) { c.guardrails = g },
	}
}

// WithGuardrailType attaches output guardrails using built-in rules for the
// specified types. For example, WithGuardrailType(GuardrailPII, GuardrailSecretLeak)
// enables PII redaction and secret leak blocking with default patterns.
func WithGuardrailType(types ...GuardrailType) ClientOption {
	var rules []GuardrailRule
	for _, t := range types {
		rules = append(rules, RulesForType(t)...)
	}
	return WithGuardrails(rules...)
}
