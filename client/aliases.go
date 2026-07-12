package client

import (
	"context"
	"net/http"
	"time"

	"github.com/GrayCodeAI/eyrie/client/core"
	"github.com/GrayCodeAI/eyrie/client/embeddings"
)

// The provider contract and request/response data types live in
// client/core so subpackages can share them without importing this facade
// (see plans/client-package-decomposition.md). The aliases below keep the
// long-standing client.* names as the public API — they are the same types,
// not copies, so existing code and type assertions are unaffected.

type (
	// Provider is the core interface for LLM providers.
	Provider = core.Provider
	// EyrieConfig holds client configuration.
	EyrieConfig = core.EyrieConfig
	// ContentPart represents a piece of content in a multi-modal message.
	ContentPart = core.ContentPart
	// ImageURLPart represents an image content part.
	ImageURLPart = core.ImageURLPart
	// InputAudioPart represents an audio content part (base64 encoded).
	InputAudioPart = core.InputAudioPart
	// EyrieMessage represents a chat message.
	EyrieMessage = core.EyrieMessage
	// ToolResult represents the result of a tool execution.
	ToolResult = core.ToolResult
	// EyrieTool represents a tool definition.
	EyrieTool = core.EyrieTool
	// EyrieUsage tracks token usage.
	EyrieUsage = core.EyrieUsage
	// EyrieResponse is the response from a chat call.
	EyrieResponse = core.EyrieResponse
	// ToolCall represents a tool invocation.
	ToolCall = core.ToolCall
	// EyrieStreamEvent is a streaming event.
	EyrieStreamEvent = core.EyrieStreamEvent
	// StreamResult wraps a streaming response with cleanup.
	StreamResult = core.StreamResult
	// ResponseFormat specifies the desired output format for the model response.
	ResponseFormat = core.ResponseFormat
	// ChatOptions holds options for a chat request.
	ChatOptions = core.ChatOptions
	// ToolChoiceOption controls how the model uses tools (Anthropic).
	ToolChoiceOption = core.ToolChoiceOption
	// ContinuationConfig controls output continuation behavior.
	ContinuationConfig = core.ContinuationConfig
	// EyrieError is a structured error that preserves provider context,
	// HTTP metadata, and request identification for debugging.
	EyrieError = core.EyrieError
	// RetryConfig controls retry behavior for HTTP clients.
	RetryConfig = core.RetryConfig
	// SSEEvent is one server-sent event from a streaming response body.
	SSEEvent = core.SSEEvent
	// RepeatDetector detects degenerate repeating output in streamed text.
	RepeatDetector = core.RepeatDetector
	// ResponseHealth classifies whether a provider response carried usable output.
	ResponseHealth = core.ResponseHealth
	// ResponseSignals are the observations needed to classify response health.
	ResponseSignals = core.ResponseSignals
	// GuardrailType classifies a guardrail rule.
	GuardrailType = core.GuardrailType
	// GuardrailAction is what happens when a guardrail rule matches.
	GuardrailAction = core.GuardrailAction
	// GuardrailSeverity ranks how serious a violation is.
	GuardrailSeverity = core.GuardrailSeverity
	// GuardrailRule is one output-filtering rule.
	GuardrailRule = core.GuardrailRule
	// GuardrailViolation records a rule match in a response.
	GuardrailViolation = core.GuardrailViolation
	// GuardrailError is returned when a blocking rule matches.
	GuardrailError = core.GuardrailError
	// Guardrails is a compiled set of output-filtering rules.
	Guardrails = core.Guardrails
	// StreamGuardrailConfig configures incremental guardrail scanning.
	StreamGuardrailConfig = core.StreamGuardrailConfig
	// StreamGuardrailResult is the outcome of scanning one stream chunk.
	StreamGuardrailResult = core.StreamGuardrailResult
	// StreamGuardrails applies guardrail rules to a response stream chunk by chunk.
	StreamGuardrails = core.StreamGuardrails
)

// NewStreamGuardrails builds an incremental guardrail scanner over a rule set.
func NewStreamGuardrails(g *Guardrails, config StreamGuardrailConfig) *StreamGuardrails {
	return core.NewStreamGuardrails(g, config)
}

// Guardrail rule types, actions, and severities (see core).
const (
	GuardrailPII             = core.GuardrailPII
	GuardrailPromptInjection = core.GuardrailPromptInjection
	GuardrailHarmfulContent  = core.GuardrailHarmfulContent
	GuardrailSecretLeak      = core.GuardrailSecretLeak
	GuardrailCustom          = core.GuardrailCustom
	GuardrailBlock           = core.GuardrailBlock
	GuardrailRedact          = core.GuardrailRedact
	GuardrailWarn            = core.GuardrailWarn
	SeverityLow              = core.SeverityLow
	SeverityMedium           = core.SeverityMedium
	SeverityHigh             = core.SeverityHigh
	SeverityCritical         = core.SeverityCritical
)

// NewGuardrails compiles a guardrail rule set (panics on invalid patterns).
func NewGuardrails(rules ...GuardrailRule) *Guardrails { return core.NewGuardrails(rules...) }

// NewGuardrailsSafe compiles a guardrail rule set, returning pattern errors.
func NewGuardrailsSafe(rules ...GuardrailRule) (*Guardrails, error) {
	return core.NewGuardrailsSafe(rules...)
}

// ApplyRedactions scrubs matched content from a response.
func ApplyRedactions(response string, violations []GuardrailViolation) string {
	return core.ApplyRedactions(response, violations)
}

// DefaultPIIRules returns the built-in PII redaction rules.
func DefaultPIIRules() []GuardrailRule { return core.DefaultPIIRules() }

// DefaultSecretLeakRules returns the built-in secret-leak blocking rules.
func DefaultSecretLeakRules() []GuardrailRule { return core.DefaultSecretLeakRules() }

// DefaultPromptInjectionRules returns the built-in prompt-injection rules.
func DefaultPromptInjectionRules() []GuardrailRule { return core.DefaultPromptInjectionRules() }

// DefaultHarmfulContentRules returns the built-in harmful-content rules.
func DefaultHarmfulContentRules() []GuardrailRule { return core.DefaultHarmfulContentRules() }

// AllDefaultRules returns every built-in guardrail rule.
func AllDefaultRules() []GuardrailRule { return core.AllDefaultRules() }

// RulesForType returns the built-in rules for one guardrail type.
func RulesForType(t GuardrailType) []GuardrailRule { return core.RulesForType(t) }

// Response health classifications (see core.DetectResponseHealth).
const (
	ResponseOK                 = core.ResponseOK
	ResponseErrorOnlyReasoning = core.ResponseErrorOnlyReasoning
	ResponseEmpty              = core.ResponseEmpty
	ResponseMalformedStream    = core.ResponseMalformedStream
)

// streamChannelBuffer bridges the buffer-size constant that moved to core.
const streamChannelBuffer = core.StreamChannelBuffer

// DetectResponseHealth classifies a response from stream/response signals.
func DetectResponseHealth(sig ResponseSignals) ResponseHealth {
	return core.DetectResponseHealth(sig)
}

// ResponseHasContent reports whether a response carries content or tool calls.
func ResponseHasContent(resp *EyrieResponse) bool {
	return core.ResponseHasContent(resp)
}

// DefaultRepeatDetector returns a RepeatDetector with production thresholds.
func DefaultRepeatDetector() *RepeatDetector {
	return core.DefaultRepeatDetector()
}

// NewPooledHTTPClient returns an *http.Client sharing the pooled transport.
func NewPooledHTTPClient(timeout time.Duration) *http.Client {
	return core.NewPooledHTTPClient(timeout)
}

// CloseIdleConnections closes idle pooled connections across all provider clients.
func CloseIdleConnections() {
	core.CloseIdleConnections()
}

// ParseInlineToolCalls extracts inline/Hermes-style tool calls from text.
func ParseInlineToolCalls(text string) (string, []ToolCall) {
	return core.ParseInlineToolCalls(text)
}

// NewRetryConfig constructs a RetryConfig from core fields and optional
// HTTP status codes to retry on.
func NewRetryConfig(maxRetries int, baseDelay, maxDelay time.Duration, retryOn ...int) RetryConfig {
	return core.NewRetryConfig(maxRetries, baseDelay, maxDelay, retryOn...)
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return core.DefaultRetryConfig()
}

// Embedding API moved to client/embeddings; aliased here for compatibility.
type (
	// Embedder is the interface for creating embeddings.
	Embedder = embeddings.Embedder
	// EmbeddingParams holds asymmetric params for indexing vs query.
	EmbeddingParams = embeddings.EmbeddingParams
	// EmbeddingRequest represents an embedding API call.
	EmbeddingRequest = embeddings.EmbeddingRequest
	// EmbeddingResponse holds embedding results.
	EmbeddingResponse = embeddings.EmbeddingResponse
	// SemanticCacheConfig configures the embedding-based semantic cache.
	SemanticCacheConfig = embeddings.SemanticCacheConfig
	// SemanticCacheStats reports semantic cache effectiveness.
	SemanticCacheStats = embeddings.SemanticCacheStats
	// EmbeddingCachedProvider caches chat responses keyed by embedding similarity.
	EmbeddingCachedProvider = embeddings.EmbeddingCachedProvider
)

// DefaultEmbeddingParams returns known-good asymmetric params for common embedding models.
func DefaultEmbeddingParams(model string) EmbeddingParams {
	return embeddings.DefaultEmbeddingParams(model)
}

// DefaultSemanticCacheConfig returns sensible semantic cache defaults.
func DefaultSemanticCacheConfig() SemanticCacheConfig {
	return embeddings.DefaultSemanticCacheConfig()
}

// NewEmbeddingCachedProvider wraps a provider with an embedding-similarity cache.
func NewEmbeddingCachedProvider(inner Provider, embedder Embedder, cfg SemanticCacheConfig) *EmbeddingCachedProvider {
	return embeddings.NewEmbeddingCachedProvider(inner, embedder, cfg)
}

// Package-local bridges for helpers that moved to client/core. Declared as
// unexported names so the many in-package call sites read unchanged while
// the single implementation lives in core.
type providerErrorDetail = core.ProviderErrorDetail

var (
	doWithRetry            = core.DoWithRetry
	parseProviderError     = core.ParseProviderError
	formatAPIError         = core.FormatAPIError
	copyResponse           = core.CopyResponse
	parseSSEStream         = core.ParseSSEStream
	processAnthropicStream = core.ProcessAnthropicStream
	processOpenAIStream    = core.ProcessOpenAIStream
	emit                   = core.Emit
	openAIImageURL         = core.OpenAIImageURL
	parseImageString       = core.ParseImageString
	userAgent              = core.UserAgent
	applyGuardrails        = core.ApplyGuardrails
)

// NewStreamResult creates a StreamResult with a cancel function for resource cleanup.
func NewStreamResult(events <-chan EyrieStreamEvent, cancel context.CancelFunc) *StreamResult {
	return core.NewStreamResult(events, cancel)
}

// NewStreamResultWithRequestID is NewStreamResult carrying the provider's request ID.
func NewStreamResultWithRequestID(events <-chan EyrieStreamEvent, requestID string, cancel context.CancelFunc) *StreamResult {
	return core.NewStreamResultWithRequestID(events, requestID, cancel)
}

// DefaultContinuationConfig returns sensible defaults.
func DefaultContinuationConfig() ContinuationConfig {
	return core.DefaultContinuationConfig()
}
