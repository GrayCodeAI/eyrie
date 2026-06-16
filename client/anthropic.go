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

// maxAnthropicRequestSize is the maximum request body size for the Messages API (32 MB).
const maxAnthropicRequestSize = 32 * 1024 * 1024

// AnthropicClient implements Provider for the Anthropic Messages API.
type AnthropicClient struct {
	apiKey             string
	baseURL            string
	version            string
	httpClient         *http.Client
	retry              RetryConfig
	logger             *slog.Logger
	defaultModel       string
	defaultMaxTokens   int
	defaultTemperature *float64
	guardrails         *Guardrails
	useMimoAuth        bool
}

// Compile-time check that AnthropicClient implements Provider.
var _ Provider = (*AnthropicClient)(nil)

// NewAnthropicClient creates a configured Anthropic client.
func NewAnthropicClient(apiKey, baseURL string, opts ...ClientOption) *AnthropicClient {
	c := &AnthropicClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		version:    "2023-06-01",
		httpClient: NewPooledHTTPClient(defaultTimeout),
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
	if c.useMimoAuth {
		req.Header.Set("api-key", c.apiKey)
	} else {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Anthropic-Version", c.version)
	req.Header.Set("User-Agent", userAgent())
}

type anthropicRequest struct {
	Model         string                   `json:"model"`
	MaxTokens     int                      `json:"max_tokens"`
	Messages      []map[string]interface{} `json:"messages"`
	System        string                   `json:"system,omitempty"`
	Temperature   *float64                 `json:"temperature,omitempty"`
	TopP          *float64                 `json:"top_p,omitempty"`
	TopK          *int                     `json:"top_k,omitempty"`
	Stream        bool                     `json:"stream,omitempty"`
	StopSequences []string                 `json:"stop_sequences,omitempty"`
	Tools         []anthropicTool          `json:"tools,omitempty"`
	ToolChoice    *anthropicToolChoice     `json:"tool_choice,omitempty"`
	Thinking      *anthropicThinking       `json:"thinking,omitempty"`
	Metadata      *anthropicMetadata       `json:"metadata,omitempty"`
	ServiceTier   string                   `json:"service_tier,omitempty"`
	OutputConfig  *anthropicOutputConfig   `json:"output_config,omitempty"`
}

// anthropicThinking enables Anthropic extended thinking.
// Type is "enabled" (with budget_tokens), "adaptive" (model decides), or "disabled".
type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
	Display      string `json:"display,omitempty"` // "summarized" or "omitted"
}

// anthropicToolChoice controls how the model uses tools.
type anthropicToolChoice struct {
	Type                   string `json:"type"`           // "auto", "any", "tool", "none"
	Name                   string `json:"name,omitempty"` // required when type="tool"
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

// anthropicMetadata carries request-level metadata.
type anthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// anthropicOutputConfig controls output format and effort.
type anthropicOutputConfig struct {
	Effort string                 `json:"effort,omitempty"` // "low","medium","high","xhigh","max"
	Format *anthropicOutputFormat `json:"format,omitempty"`
}

// anthropicOutputFormat specifies structured output format.
type anthropicOutputFormat struct {
	Type   string                 `json:"type"` // "json_schema"
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// thinkingForBudget returns an enabled thinking config when budget > 0, else nil.
func thinkingForBudget(budget int) *anthropicThinking {
	if budget <= 0 {
		return nil
	}
	return &anthropicThinking{Type: "enabled", BudgetTokens: budget}
}

// thinkingAdaptive returns an adaptive thinking config.
func thinkingAdaptive() *anthropicThinking {
	return &anthropicThinking{Type: "adaptive"}
}

// thinkingDisabled returns a disabled thinking config.
func thinkingDisabled() *anthropicThinking {
	return &anthropicThinking{Type: "disabled"}
}

// resolveThinking builds the thinking config from ChatOptions.
func resolveThinking(opts ChatOptions) *anthropicThinking {
	switch opts.ThinkingMode {
	case "adaptive":
		return thinkingAdaptive()
	case "disabled":
		return thinkingDisabled()
	case "enabled":
		thinking := thinkingForBudget(opts.ThinkingBudgetTokens)
		if thinking != nil && opts.ThinkingDisplay != "" {
			thinking.Display = opts.ThinkingDisplay
		}
		return thinking
	default:
		// Legacy behavior: if budget > 0, enable with budget
		return thinkingForBudget(opts.ThinkingBudgetTokens)
	}
}

// resolveToolChoice converts ChatOptions.ToolChoice to wire format.
func resolveToolChoice(tc *ToolChoiceOption) *anthropicToolChoice {
	if tc == nil {
		return nil
	}
	return &anthropicToolChoice{
		Type:                   tc.Type,
		Name:                   tc.Name,
		DisableParallelToolUse: tc.DisableParallelToolUse,
	}
}

// resolveMetadata builds metadata from ChatOptions.
func resolveMetadata(opts ChatOptions) *anthropicMetadata {
	if opts.MetadataUserID == "" {
		return nil
	}
	return &anthropicMetadata{UserID: opts.MetadataUserID}
}

// resolveOutputConfig builds output config from ChatOptions.
func resolveOutputConfig(opts ChatOptions) *anthropicOutputConfig {
	if opts.OutputEffort == "" && opts.OutputSchema == "" {
		return nil
	}
	cfg := &anthropicOutputConfig{Effort: opts.OutputEffort}
	if opts.OutputSchema != "" {
		var schema map[string]interface{}
		if json.Unmarshal([]byte(opts.OutputSchema), &schema) == nil {
			cfg.Format = &anthropicOutputFormat{Type: "json_schema", Schema: schema}
		}
	}
	return cfg
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type      string          `json:"type"`
		Text      string          `json:"text,omitempty"`
		Thinking  string          `json:"thinking,omitempty"`  // thinking block content
		Signature string          `json:"signature,omitempty"` // thinking signature for multi-turn
		Data      string          `json:"data,omitempty"`      // redacted_thinking data
		ID        string          `json:"id,omitempty"`
		Name      string          `json:"name,omitempty"`
		Input     json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		OutputTokensDetails      struct {
			ThinkingTokens int `json:"thinking_tokens"`
		} `json:"output_tokens_details"`
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
		// User message with tool results
		if m.Role == "user" && len(m.ToolResults) > 0 {
			content := make([]map[string]interface{}, 0, len(m.ToolResults))
			for _, tr := range m.ToolResults {
				block := map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": tr.ToolUseID,
					"content":     tr.Content,
				}
				if tr.IsError {
					block["is_error"] = true
				}
				content = append(content, block)
			}
			msgs = append(msgs, map[string]interface{}{"role": "user", "content": content})
			continue
		}
		// Handle ContentParts (multi-modal): takes precedence over Content/Images
		if len(m.ContentParts) > 0 {
			content := make([]map[string]interface{}, 0, len(m.ContentParts))
			for _, part := range m.ContentParts {
				switch part.Type {
				case "text":
					content = append(content, map[string]interface{}{"type": "text", "text": part.Text})
				case "image_url":
					if part.ImageURL != nil {
						mediaType, data, isBase64 := parseImageString(part.ImageURL.URL)
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
							content = append(content, map[string]interface{}{
								"type": "image",
								"source": map[string]interface{}{
									"type": "url",
									"url":  part.ImageURL.URL,
								},
							})
						}
					}
				case "input_audio":
					if part.InputAudio != nil {
						// Anthropic expects audio as an "audio" content block with base64 source
						mediaType := audioFormatToMediaType(part.InputAudio.Format)
						content = append(content, map[string]interface{}{
							"type": "audio",
							"source": map[string]interface{}{
								"type":       "base64",
								"media_type": mediaType,
								"data":       part.InputAudio.Data,
							},
						})
					}
				}
			}
			msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": content})
			continue
		}
		// Handle legacy Images field: build multi-part content array
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

// audioFormatToMediaType converts a short audio format string to a full MIME type.
func audioFormatToMediaType(format string) string {
	switch strings.ToLower(format) {
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "ogg":
		return "audio/ogg"
	case "aac":
		return "audio/aac"
	case "webm":
		return "audio/webm"
	default:
		// If it looks like a full MIME type already, pass through
		if strings.Contains(format, "/") {
			return format
		}
		return "audio/" + format
	}
}

// Chat sends a non-streaming message to Anthropic.
// buildAnthropicRequest constructs the request body and http.Request
// for both Chat and StreamChat. If stream is true, the body sets
// `stream: true` and the request gets the `Accept: text/event-stream`
// header. Returns the http.Request (with GetBody set for retry) and
// the raw body bytes (needed by doRequestWithMimoAuthRetry for the
// MiMo 401 retry path).
//
// This helper removes ~120 lines of duplication between Chat and
// StreamChat (lines 375-446 and 496-565 in the previous version):
// every field — System, Temperature, TopP, TopK, StopSequences,
// EnableCaching, tools, thinking, metadata, serviceTier,
// outputConfig — was previously re-applied in both methods.
func (c *AnthropicClient) buildAnthropicRequest(ctx context.Context, messages []EyrieMessage, opts ChatOptions, stream bool) (*http.Request, []byte, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, nil, fmt.Errorf("eyrie: model is required for anthropic")
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	thinking := resolveThinking(opts)

	var body []byte
	if opts.EnableCaching {
		allMessages := messages
		if opts.System != "" {
			allMessages = append([]EyrieMessage{{Role: "system", Content: opts.System}}, allMessages...)
		}
		tools := convertToAnthropicTools(opts.Tools)
		cachedReq := buildAnthropicCachedRequest(allMessages, opts.Model, maxTokens, opts.Temperature, stream, tools,
			thinking, resolveToolChoice(opts.ToolChoice), opts.TopP, opts.TopK, opts.StopSequences)
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
			Model:         opts.Model,
			MaxTokens:     maxTokens,
			Messages:      msgs,
			System:        system,
			Temperature:   opts.Temperature,
			TopP:          opts.TopP,
			TopK:          opts.TopK,
			StopSequences: opts.StopSequences,
			Stream:        stream,
			Tools:         convertToAnthropicTools(opts.Tools),
			ToolChoice:    resolveToolChoice(opts.ToolChoice),
			Thinking:      thinking,
			Metadata:      resolveMetadata(opts),
			ServiceTier:   opts.ServiceTier,
			OutputConfig:  resolveOutputConfig(opts),
		}
		body, _ = json.Marshal(reqBody)
	}

	// Check request size (32 MB limit for Messages API)
	if len(body) > maxAnthropicRequestSize {
		return nil, nil, fmt.Errorf("eyrie: request size %d bytes exceeds Anthropic limit of %d bytes", len(body), maxAnthropicRequestSize)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("eyrie: failed to create request: %w", err)
	}
	c.setHeaders(req)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	return req, body, nil
}

// NOTE: Anthropic does not support a native JSON mode (response_format).
// Structured output with Anthropic is achieved via the tool-use pattern
// (defining a tool whose input_schema is your desired output schema).
// This is not implemented here; opts.ResponseFormat is ignored for Anthropic.
// Future work: implement tool-use-based structured output for Anthropic.
func (c *AnthropicClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	req, body, err := c.buildAnthropicRequest(ctx, messages, opts, false)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("anthropic chat", "model", opts.Model, "caching", opts.EnableCaching)

	resp, err := c.doRequestWithMimoAuthRetry(ctx, req, body)
	if err != nil {
		return nil, fmt.Errorf("eyrie: anthropic request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("Request-Id")
	orgID := resp.Header.Get("Anthropic-Organization-Id")

	if resp.StatusCode != 200 {
		return nil, formatAPIError("anthropic", resp.StatusCode, requestID, parseProviderError(resp.Body))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("eyrie: failed to decode anthropic response: %w", err)
	}

	var content, thinkingContent string
	var toolCalls []ToolCall
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "thinking":
			thinkingContent += block.Thinking
		case "redacted_thinking":
			// Safety-sensitive reasoning — skip silently
			continue
		case "tool_use":
			var args map[string]interface{}
			_ = json.Unmarshal(block.Input, &args)
			toolCalls = append(toolCalls, ToolCall{ID: block.ID, Name: block.Name, Arguments: args})
		}
	}

	eyrieResp := &EyrieResponse{
		Content: content, Thinking: thinkingContent, FinishReason: ar.StopReason, ToolCalls: toolCalls,
		RequestID: requestID, OrganizationID: orgID,
		Usage: &EyrieUsage{
			PromptTokens:        ar.Usage.InputTokens,
			CompletionTokens:    ar.Usage.OutputTokens,
			TotalTokens:         ar.Usage.InputTokens + ar.Usage.OutputTokens,
			CacheCreationTokens: ar.Usage.CacheCreationInputTokens,
			CacheReadTokens:     ar.Usage.CacheReadInputTokens,
			ThinkingTokens:      ar.Usage.OutputTokensDetails.ThinkingTokens,
		},
	}

	if err := applyGuardrails(ctx, eyrieResp, c.guardrails); err != nil {
		return nil, err
	}

	return eyrieResp, nil
}

// StreamChat sends a streaming message to Anthropic.
func (c *AnthropicClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	req, body, err := c.buildAnthropicRequest(ctx, messages, opts, true)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestWithMimoAuthRetry(ctx, req, body)
	if err != nil {
		return nil, fmt.Errorf("eyrie: anthropic stream request failed: %w", err)
	}

	requestID := resp.Header.Get("Request-Id")

	if resp.StatusCode != 200 {
		detail := parseProviderError(resp.Body)
		_ = resp.Body.Close()
		return nil, formatAPIError("anthropic", resp.StatusCode, requestID, detail)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, c.logger)
	events := processAnthropicStream(streamCtx, sseEvents, c.logger)

	return &StreamResult{Events: events, RequestID: requestID, cancel: cancel}, nil
}

func (c *AnthropicClient) doRequestWithMimoAuthRetry(ctx context.Context, req *http.Request, body []byte) (*http.Response, error) {
	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, err
	}
	if !c.useMimoAuth || resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	_ = resp.Body.Close()
	req2, err := http.NewRequestWithContext(ctx, req.Method, req.URL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+c.apiKey)
	req2.Header.Set("Anthropic-Version", c.version)
	req2.Header.Set("User-Agent", userAgent())
	if req.Header.Get("Accept") != "" {
		req2.Header.Set("Accept", req.Header.Get("Accept"))
	}
	req2.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	return doWithRetry(ctx, c.httpClient, req2, c.retry, c.logger)
}

// Ping checks connectivity to the Anthropic API using a lightweight GET request.
func (c *AnthropicClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/v1/models", nil)
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

// TokenCountResult holds the result of a token count request.
type TokenCountResult struct {
	InputTokens int `json:"input_tokens"`
}

// CountTokens counts the number of tokens in a message without generating a response.
// Uses the same request format as Chat but hits the /v1/messages/count_tokens endpoint.
// The request body includes model, messages, system, and tools (but not max_tokens or stream).
func (c *AnthropicClient) CountTokens(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*TokenCountResult, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for anthropic count_tokens")
	}

	msgs, system := buildAnthropicMessages(messages)
	if opts.System != "" {
		if system != "" {
			system = opts.System + "\n\n" + system
		} else {
			system = opts.System
		}
	}

	// Build a minimal request for token counting (no max_tokens, no stream)
	reqBody := map[string]interface{}{
		"model":    opts.Model,
		"messages": msgs,
	}
	if system != "" {
		reqBody["system"] = system
	}
	if len(opts.Tools) > 0 {
		reqBody["tools"] = convertToAnthropicTools(opts.Tools)
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("eyrie: failed to marshal count_tokens request: %w", err)
	}

	// Check request size (32 MB limit for Messages API)
	if len(body) > maxAnthropicRequestSize {
		return nil, fmt.Errorf("eyrie: request size %d bytes exceeds Anthropic limit of %d bytes", len(body), maxAnthropicRequestSize)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages/count_tokens", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: failed to create count_tokens request: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := c.doRequestWithMimoAuthRetry(ctx, req, body)
	if err != nil {
		return nil, fmt.Errorf("eyrie: anthropic count_tokens failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, formatAPIError("anthropic", resp.StatusCode, resp.Header.Get("Request-Id"), parseProviderError(resp.Body))
	}

	var result TokenCountResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eyrie: failed to decode count_tokens response: %w", err)
	}
	return &result, nil
}
