package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// AnthropicClient implements Provider for the Anthropic Messages API.
type AnthropicClient struct {
	apiKey     string
	baseURL    string
	version    string
	httpClient *http.Client
	retry      RetryConfig
	logger     *slog.Logger
}

// Compile-time check that AnthropicClient implements Provider.
var _ Provider = (*AnthropicClient)(nil)

// NewAnthropicClient creates a configured Anthropic client.
func NewAnthropicClient(apiKey, baseURL string, opts ...ClientOption) *AnthropicClient {
	c := &AnthropicClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		version:    "2023-06-01",
		httpClient: &http.Client{Timeout: defaultTimeout},
		retry:      DefaultRetryConfig(),
		logger:     slog.Default(),
	}
	if c.baseURL == "" {
		c.baseURL = "https://api.anthropic.com"
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// Name returns the provider name.
func (c *AnthropicClient) Name() string { return "anthropic" }

func (c *AnthropicClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", c.version)
	req.Header.Set("User-Agent", userAgent())
}

type anthropicRequest struct {
	Model       string                   `json:"model"`
	MaxTokens   int                      `json:"max_tokens"`
	Messages    []map[string]interface{} `json:"messages"`
	System      string                   `json:"system,omitempty"`
	Temperature *float64                 `json:"temperature,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	Tools       []anthropicTool          `json:"tools,omitempty"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

func buildAnthropicMessages(messages []EyrieMessage) ([]map[string]interface{}, string) {
	var system string
	msgs := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		// Assistant message with tool_use blocks
		if m.Role == "assistant" && len(m.ToolUse) > 0 {
			content := make([]map[string]interface{}, 0)
			if m.Content != "" {
				content = append(content, map[string]interface{}{"type": "text", "text": m.Content})
			}
			for _, tc := range m.ToolUse {
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": tc.Arguments,
				})
			}
			msgs = append(msgs, map[string]interface{}{"role": "assistant", "content": content})
			continue
		}
		// User message with tool_result
		if m.Role == "user" && m.ToolResult != nil {
			content := []map[string]interface{}{{
				"type":        "tool_result",
				"tool_use_id": m.ToolResult.ToolUseID,
				"content":     m.ToolResult.Content,
			}}
			if m.ToolResult.IsError {
				content[0]["is_error"] = true
			}
			msgs = append(msgs, map[string]interface{}{"role": "user", "content": content})
			continue
		}
		// Handle messages with images: build multi-part content array
		if len(m.Images) > 0 {
			content := make([]map[string]interface{}, 0, 1+len(m.Images))
			if m.Content != "" {
				content = append(content, map[string]interface{}{"type": "text", "text": m.Content})
			}
			for _, img := range m.Images {
				mediaType, data, isBase64 := parseImageString(img)
				if isBase64 {
					content = append(content, map[string]interface{}{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": mediaType,
							"data":       data,
						},
					})
				} else {
					// URL-based image
					content = append(content, map[string]interface{}{
						"type": "image",
						"source": map[string]interface{}{
							"type": "url",
							"url":  img,
						},
					})
				}
			}
			msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": content})
			continue
		}
		msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": m.Content})
	}
	return msgs, system
}

// parseImageString parses an image string into media type, data, and whether it is base64.
// Accepts "data:<mediaType>;base64,<data>" or a plain URL (http/https).
func parseImageString(img string) (mediaType, data string, isBase64 bool) {
	if strings.HasPrefix(img, "data:") {
		// Format: data:<mediaType>;base64,<data>
		rest := strings.TrimPrefix(img, "data:")
		if semiIdx := strings.Index(rest, ";base64,"); semiIdx >= 0 {
			mediaType = rest[:semiIdx]
			data = rest[semiIdx+len(";base64,"):]
			return mediaType, data, true
		}
	}
	// Treat as URL
	return "", img, false
}

// Chat sends a non-streaming message to Anthropic.
// NOTE: Anthropic does not support a native JSON mode (response_format).
// Structured output with Anthropic is achieved via the tool-use pattern
// (defining a tool whose input_schema is your desired output schema).
// This is not implemented here; opts.ResponseFormat is ignored for Anthropic.
// Future work: implement tool-use-based structured output for Anthropic.
func (c *AnthropicClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for anthropic")
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	var body []byte
	if opts.EnableCaching {
		allMessages := messages
		if opts.System != "" {
			allMessages = append([]EyrieMessage{{Role: "system", Content: opts.System}}, allMessages...)
		}
		tools := convertToAnthropicTools(opts.Tools)
		cachedReq := buildAnthropicCachedRequest(allMessages, opts.Model, maxTokens, opts.Temperature, false, tools)
		body, _ = json.Marshal(cachedReq)
	} else {
		msgs, system := buildAnthropicMessages(messages)
		if opts.System != "" {
			if system != "" {
				system = opts.System + "\n\n" + system
			} else {
				system = opts.System
			}
		}
		reqBody := anthropicRequest{
			Model: opts.Model, MaxTokens: maxTokens, Messages: msgs,
			System: system, Temperature: opts.Temperature,
			Tools: convertToAnthropicTools(opts.Tools),
		}
		body, _ = json.Marshal(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: failed to create request: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	c.logger.Debug("anthropic chat", "model", opts.Model, "caching", opts.EnableCaching)

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: anthropic request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("Request-Id")

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("eyrie: anthropic API error (request_id=%s): %s", requestID, parseErrorBody(resp.Body))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("eyrie: failed to decode anthropic response: %w", err)
	}

	var content string
	var toolCalls []ToolCall
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			var args map[string]interface{}
			_ = json.Unmarshal(block.Input, &args)
			toolCalls = append(toolCalls, ToolCall{ID: block.ID, Name: block.Name, Arguments: args})
		}
	}

	return &EyrieResponse{
		Content: content, FinishReason: ar.StopReason, ToolCalls: toolCalls,
		RequestID: requestID,
		Usage: &EyrieUsage{
			PromptTokens:        ar.Usage.InputTokens,
			CompletionTokens:    ar.Usage.OutputTokens,
			TotalTokens:         ar.Usage.InputTokens + ar.Usage.OutputTokens,
			CacheCreationTokens: ar.Usage.CacheCreationInputTokens,
			CacheReadTokens:     ar.Usage.CacheReadInputTokens,
		},
	}, nil
}

// StreamChat sends a streaming message to Anthropic.
func (c *AnthropicClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for anthropic")
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	var body []byte
	if opts.EnableCaching {
		allMessages := messages
		if opts.System != "" {
			allMessages = append([]EyrieMessage{{Role: "system", Content: opts.System}}, allMessages...)
		}
		tools := convertToAnthropicTools(opts.Tools)
		cachedReq := buildAnthropicCachedRequest(allMessages, opts.Model, maxTokens, opts.Temperature, true, tools)
		body, _ = json.Marshal(cachedReq)
	} else {
		msgs, system := buildAnthropicMessages(messages)
		if opts.System != "" {
			if system != "" {
				system = opts.System + "\n\n" + system
			} else {
				system = opts.System
			}
		}
		reqBody := anthropicRequest{
			Model: opts.Model, MaxTokens: maxTokens, Messages: msgs,
			System: system, Temperature: opts.Temperature, Stream: true,
			Tools: convertToAnthropicTools(opts.Tools),
		}
		body, _ = json.Marshal(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: failed to create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: anthropic stream request failed: %w", err)
	}

	requestID := resp.Header.Get("Request-Id")

	if resp.StatusCode != 200 {
		errMsg := parseErrorBody(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("eyrie: anthropic API error (request_id=%s): %s", requestID, errMsg)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, c.logger)
	events := processAnthropicStream(streamCtx, sseEvents, c.logger)

	return &StreamResult{Events: events, RequestID: requestID, cancel: cancel}, nil
}

// Ping checks connectivity to the Anthropic API.
func (c *AnthropicClient) Ping(ctx context.Context) error {
	body := []byte(`{"model":"claude-3-5-haiku-20241022","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("eyrie: anthropic ping failed: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyrie: anthropic ping failed: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("eyrie: anthropic: invalid API key")
	}
	return nil
}

func convertToAnthropicTools(tools []EyrieTool) []anthropicTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropicTool, len(tools))
	for i, t := range tools {
		out[i] = anthropicTool{Name: t.Name, Description: t.Description, InputSchema: t.Parameters}
	}
	return out
}
