package types

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// --- Branded string types ---

type (
	SessionId string
	AgentId   string
)

var agentIdPattern = regexp.MustCompile(`^a(?:.+-)?[0-9a-f]{16}$`)

func AsSessionId(s string) SessionId { return SessionId(s) }
func AsAgentId(s string) AgentId     { return AgentId(s) }

func ToAgentId(s string) (*AgentId, error) {
	if !agentIdPattern.MatchString(s) {
		return nil, nil
	}
	id := AgentId(s)
	return &id, nil
}

// --- Connector types ---

type ConnectorTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ConnectorTextDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func IsConnectorTextBlock(m map[string]interface{}) bool {
	t, ok := m["type"].(string)
	return ok && t == "connector_text"
}

// --- SDK content block types ---

type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type TextBlockParam struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Base64ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ImageBlockParam struct {
	Type   string            `json:"type"`
	Source Base64ImageSource `json:"source"`
}

type ToolUseBlock struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type ToolUseBlockParam struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type ToolResultBlockParam struct {
	Type      string      `json:"type"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"is_error"`
}

type ThinkingBlock struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

type ThinkingBlockParam struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

type RedactedThinkingBlock struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type RedactedThinkingBlockParam struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type (
	ContentBlock      = interface{}
	ContentBlockParam = interface{}
)

// --- Message types ---

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type Message struct {
	ID           string      `json:"id"`
	Type_        string      `json:"type"`
	Role         string      `json:"role"`
	Content      interface{} `json:"content"`
	Model        string      `json:"model"`
	StopReason   string      `json:"stop_reason"`
	StopSequence string      `json:"stop_sequence"`
	Usage        *Usage      `json:"usage"`
}

type MessageParam struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type BetaUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type BetaMessage struct {
	ID           string         `json:"id"`
	Type_        string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence"`
	Usage        BetaUsage      `json:"usage"`
}

type BetaMessageParam struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema ToolInputSchema `json:"input_schema"`
}

// --- StopReason constants ---

const (
	EndTurn      = "end_turn"
	MaxTokens    = "max_tokens"
	StopSequence = "stop_sequence"
	ToolUse      = "tool_use"
)

// --- MessageOrigin constants ---

type MessageOrigin = string

const (
	User       MessageOrigin = "user"
	API        MessageOrigin = "api"
	ToolOrigin MessageOrigin = "tool"
	System     MessageOrigin = "system"
	Compact    MessageOrigin = "compact"
	Recovery   MessageOrigin = "recovery"
)

// --- MessageSource constants ---

type MessageSource = string

const (
	UserSource   MessageSource = "user"
	Teammate     MessageSource = "teammate"
	SystemSource MessageSource = "system"
	Tick         MessageSource = "tick"
	Task         MessageSource = "task"
)

// --- Specialized message types ---

type ErrorDetails struct {
	ActualTokens *int `json:"actual_tokens,omitempty"`
	LimitTokens  *int `json:"limit_tokens,omitempty"`
}

type UserMessage struct {
	Message
}

type AssistantMessage struct {
	Message
	IsApiErrorMessage bool          `json:"is_api_error_message"`
	ErrorDetails      *ErrorDetails `json:"error_details,omitempty"`
}

type SystemMessage struct {
	Message
}

// --- Request/config types ---

type ToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type MessageCreateParams struct {
	Model         string         `json:"model"`
	MaxTokens     int            `json:"max_tokens"`
	Messages      []MessageParam `json:"messages"`
	Tools         []Tool         `json:"tools,omitempty"`
	ToolChoice    *ToolChoice    `json:"tool_choice,omitempty"`
	System        *string        `json:"system,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
	TopP          *float64       `json:"top_p,omitempty"`
	TopK          *int           `json:"top_k,omitempty"`
	StopSequences []string       `json:"stop_sequences,omitempty"`
	Stream        bool           `json:"stream"`
}

type MessageStreamEvent struct {
	Type string `json:"type"`
}

type ClientOptions struct {
	APIKey     string `json:"api_key"`
	BaseURL    string `json:"base_url"`
	Timeout    int    `json:"timeout"`
	MaxRetries int    `json:"max_retries"`
}

// --- Type guard functions ---

func IsTextBlock(v interface{}) (TextBlock, bool) {
	b, ok := v.(TextBlock)
	return b, ok
}

func IsImageBlock(v interface{}) (ImageBlockParam, bool) {
	b, ok := v.(ImageBlockParam)
	return b, ok
}

func IsToolUseBlock(v interface{}) (ToolUseBlock, bool) {
	b, ok := v.(ToolUseBlock)
	return b, ok
}

func IsToolResultBlock(v interface{}) (ToolResultBlockParam, bool) {
	b, ok := v.(ToolResultBlockParam)
	return b, ok
}

// --- Creator functions ---

func CreateUserMessage(text string) UserMessage {
	return UserMessage{
		Message: Message{
			Role:    "user",
			Content: text,
		},
	}
}

func CreateAssistantMessage(text string) AssistantMessage {
	return AssistantMessage{
		Message: Message{
			Role:    "assistant",
			Content: text,
		},
	}
}

func CreateSystemMessage(text string) SystemMessage {
	return SystemMessage{
		Message: Message{
			Role:    "system",
			Content: text,
		},
	}
}

// --- API Error types ---

type APIError struct {
	Status  int                    `json:"status"`
	Headers map[string]string      `json:"headers"`
	Err     map[string]interface{} `json:"error"`
	Message string                 `json:"message"`
}

func (e *APIError) Error() string { return e.Message }

func NewAPIError(status int, headers map[string]string, err map[string]interface{}, message string) *APIError {
	return &APIError{Status: status, Headers: headers, Err: err, Message: message}
}

type APIConnectionError struct{ APIError }

func NewAPIConnectionError(message string) *APIConnectionError {
	return &APIConnectionError{APIError{Message: message}}
}

type APIConnectionTimeoutError struct{ APIError }

func NewAPIConnectionTimeoutError(message string) *APIConnectionTimeoutError {
	return &APIConnectionTimeoutError{APIError{Message: message}}
}

type APIUserAbortError struct{ APIError }

func NewAPIUserAbortError(message string) *APIUserAbortError {
	return &APIUserAbortError{APIError{Message: message}}
}

type NotFoundError struct{ APIError }

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{APIError{Status: 404, Message: message}}
}

type AuthenticationError struct{ APIError }

func NewAuthenticationError(message string) *AuthenticationError {
	return &AuthenticationError{APIError{Status: 401, Message: message}}
}

// --- Usage types ---

type ServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests"`
	WebFetchRequests  int `json:"web_fetch_requests"`
}

type CacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
}

type NonNullableUsage struct {
	InputTokens              int           `json:"input_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	ServerToolUse            ServerToolUse `json:"server_tool_use"`
	ServiceTier              string        `json:"service_tier"`
	CacheCreation            CacheCreation `json:"cache_creation"`
	InferenceGeo             string        `json:"inference_geo"`
	Iterations               []interface{} `json:"iterations"`
	Speed                    string        `json:"speed"`
}

func EmptyUsage() NonNullableUsage {
	return NonNullableUsage{Iterations: []interface{}{}}
}

// TransientError wraps an HTTP status code and message to indicate a retriable error.
// Use errors.As to check for transient errors programmatically.
type TransientError struct {
	StatusCode int
	Message    string
}

func (e *TransientError) Error() string {
	return fmt.Sprintf("transient error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsTransient reports whether the error is likely temporary and worth retrying.
// This is the canonical retry classifier used by the retry middleware.
//
// Conservative by default: unknown errors (those that match neither retriable
// nor non-retriable patterns) are treated as NOT retriable. This avoids
// unnecessary retries for unexpected error types (e.g., malformed responses,
// serialization failures). Callers like FallbackProvider that want optimistic
// fallback on unknown errors implement their own wrapper — see client.isRetriableError.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	// Check for typed TransientError
	var te *TransientError
	if errors.As(err, &te) {
		return true
	}
	msg := strings.ToLower(err.Error())

	// Extract HTTP status code if present and check explicit lists.
	nonRetriableCodes := map[int]bool{400: true, 401: true, 403: true, 404: true, 422: true}
	retriableCodes := map[int]bool{408: true, 429: true, 500: true, 502: true, 503: true, 504: true, 529: true}
	if matches := httpStatusRe.FindStringSubmatch(err.Error()); len(matches) >= 2 {
		if code, err := strconv.Atoi(matches[1]); err == nil {
			if nonRetriableCodes[code] {
				return false
			}
			if retriableCodes[code] {
				return true
			}
		}
	}

	for _, pattern := range []string{
		"timeout", "timed out", "deadline exceeded",
		"connection refused", "connection reset", "eof", "broken pipe",
		"temporarily", "overloaded", "try again",
		"unavailable", "server error", "bad gateway", "service unavailable",
		"rate limit", "rate_limit",
	} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	if _, ok := err.(*APIConnectionTimeoutError); ok {
		return true
	}
	return false
}

var httpStatusRe = regexp.MustCompile(`\b(\d{3})\b`)

// ExtractHTTPStatus attempts to extract an HTTP status code from err's message.
// Uses the same regex as IsTransient to find 3-digit codes in error text.
// Returns the status code and true if found, or 0 and false otherwise.
func ExtractHTTPStatus(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	matches := httpStatusRe.FindStringSubmatch(err.Error())
	if len(matches) < 2 {
		return 0, false
	}
	code, convErr := strconv.Atoi(matches[1])
	if convErr != nil {
		return 0, false
	}
	return code, true
}

// ClassifyError creates a structured error from a status code and message.
// It returns the most specific error type for the given error condition.
// For retriable status codes (408, 429, 500, 502, 503, 504, 529), returns a TransientError.
func ClassifyError(provider string, statusCode int, message string) error {
	retriableCodes := map[int]bool{408: true, 429: true, 500: true, 502: true, 503: true, 504: true, 529: true}
	if retriableCodes[statusCode] {
		return &TransientError{StatusCode: statusCode, Message: message}
	}
	return &APIError{
		Status:  statusCode,
		Message: message,
	}
}
