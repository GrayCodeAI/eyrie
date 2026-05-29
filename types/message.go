package types

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
	APIKey     string `json:"-"`
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
