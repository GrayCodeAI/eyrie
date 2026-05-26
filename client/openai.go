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

// OpenAIClient implements Provider for OpenAI and OpenAI-compatible APIs.
type OpenAIClient struct {
	apiKey       string
	baseURL      string
	providerName string
	compat       *OpenAICompatConfig
	httpClient   *http.Client
	retry        RetryConfig
	logger       *slog.Logger
}

// Compile-time check that OpenAIClient implements Provider.
var _ Provider = (*OpenAIClient)(nil)

// NewOpenAIClient creates a configured OpenAI/compatible client.
func NewOpenAIClient(apiKey, baseURL string, compat *OpenAICompatConfig, opts ...ClientOption) *OpenAIClient {
	c := &OpenAIClient{
		apiKey:       apiKey,
		baseURL:      baseURL,
		providerName: "openai",
		compat:       compat,
		// TODO: Use a shared http.Transport with MaxIdleConnsPerHost and
		// IdleConnTimeout to enable connection pooling across providers,
		// reducing latency and connection overhead.
		httpClient:   &http.Client{Timeout: defaultTimeout},
		retry:        DefaultRetryConfig(),
		logger:       slog.Default(),
	}
	if c.baseURL == "" {
		c.baseURL = "https://api.openai.com/v1"
	}
	if c.compat == nil {
		c.compat = &OpenAICompat
	}
	for _, opt := range opts {
		opt.applyOpenAI(c)
	}
	return c
}

// Name returns the provider name.
func (c *OpenAIClient) Name() string { return c.providerName }

func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent())
}

type openaiRequest struct {
	Model               string                   `json:"model"`
	Messages            []map[string]interface{} `json:"messages"`
	MaxTokens           *int                     `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                     `json:"max_completion_tokens,omitempty"`
	Temperature         *float64                 `json:"temperature,omitempty"`
	Stream              bool                     `json:"stream,omitempty"`
	StreamOptions       *streamOptions           `json:"stream_options,omitempty"`
	Tools               []map[string]interface{} `json:"tools,omitempty"`
	ResponseFormat      map[string]interface{}   `json:"response_format,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openaiResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details,omitempty"`
	} `json:"usage"`
}

// buildRequestBase builds an OpenAI-compatible request body.
// When compat is non-nil, MaxTokensField and SupportsUsageInStreaming overrides are applied.
func buildRequestBase(messages []EyrieMessage, opts ChatOptions, stream bool, compat *OpenAICompatConfig) openaiRequest {
	var msgs []map[string]interface{}
	for _, m := range messages {
		if m.ToolResult != nil {
			msgs = append(msgs, map[string]interface{}{
				"role":         "tool",
				"content":      m.ToolResult.Content,
				"tool_call_id": m.ToolResult.ToolUseID,
			})
			continue
		}
		msg := map[string]interface{}{"role": m.Role, "content": m.Content}
		// Handle messages with images: build multi-part content array
		if len(m.Images) > 0 {
			content := make([]map[string]interface{}, 0, 1+len(m.Images))
			if m.Content != "" {
				content = append(content, map[string]interface{}{"type": "text", "text": m.Content})
			}
			for _, img := range m.Images {
				switch {
				case strings.HasPrefix(img, "data:"):
					// Already a data URI, pass directly
					content = append(content, map[string]interface{}{
						"type":      "image_url",
						"image_url": map[string]interface{}{"url": img},
					})
				case strings.HasPrefix(img, "http"):
					// Plain URL
					content = append(content, map[string]interface{}{
						"type":      "image_url",
						"image_url": map[string]interface{}{"url": img},
					})
				default:
					// Assume raw base64 data, default to image/png
					content = append(content, map[string]interface{}{
						"type":      "image_url",
						"image_url": map[string]interface{}{"url": "data:image/png;base64," + img},
					})
				}
			}
			msg["content"] = content
		}
		if len(m.ToolUse) > 0 {
			toolCalls := make([]map[string]interface{}, len(m.ToolUse))
			for i, tc := range m.ToolUse {
				args, _ := json.Marshal(tc.Arguments)
				toolCalls[i] = map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": string(args),
					},
				}
			}
			msg["tool_calls"] = toolCalls
		}
		msgs = append(msgs, msg)
	}
	req := openaiRequest{Model: opts.Model, Messages: msgs, Temperature: opts.Temperature, Stream: stream}
	if len(opts.Tools) > 0 {
		tools := make([]map[string]interface{}, len(opts.Tools))
		for i, t := range opts.Tools {
			tools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		req.Tools = tools
	}
	maxTok := opts.MaxTokens
	if maxTok == 0 {
		maxTok = 4096
	}
	if compat != nil && compat.MaxTokensField == "max_completion_tokens" {
		req.MaxCompletionTokens = &maxTok
	} else {
		req.MaxTokens = &maxTok
	}
	if stream {
		if compat == nil || compat.SupportsUsageInStreaming {
			req.StreamOptions = &streamOptions{IncludeUsage: true}
		}
	}
	if opts.ResponseFormat != nil {
		rf := map[string]interface{}{"type": opts.ResponseFormat.Type}
		if opts.ResponseFormat.Schema != "" && opts.ResponseFormat.Type == "json_schema" {
			var schema map[string]interface{}
			if json.Unmarshal([]byte(opts.ResponseFormat.Schema), &schema) == nil {
				rf["json_schema"] = schema
			}
		}
		req.ResponseFormat = rf
	}
	return req
}

func (c *OpenAIClient) buildRequest(messages []EyrieMessage, opts ChatOptions, stream bool) openaiRequest {
	return buildRequestBase(messages, opts, stream, c.compat)
}

// buildOpenAIRequest builds the shared OpenAI-compatible request body.
func buildOpenAIRequest(messages []EyrieMessage, opts ChatOptions, stream bool) openaiRequest {
	return buildRequestBase(messages, opts, stream, nil)
}

// Chat sends a non-streaming request.
func (c *OpenAIClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for %s", c.providerName)
	}

	reqBody := c.buildRequest(messages, opts, false)
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: failed to create request: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	c.logger.Debug("openai chat", "provider", c.providerName, "model", opts.Model, "base_url", c.baseURL)

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: %s request failed: %w", c.providerName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("X-Request-Id")

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("eyrie: %s API error (request_id=%s): %s", c.providerName, requestID, parseErrorBody(resp.Body))
	}

	var or openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("eyrie: failed to decode %s response: %w", c.providerName, err)
	}

	result := &EyrieResponse{FinishReason: "unknown", RequestID: requestID}
	if len(or.Choices) > 0 {
		ch := or.Choices[0]
		result.Content = ch.Message.Content
		result.FinishReason = ch.FinishReason
		for _, tc := range ch.Message.ToolCalls {
			var args map[string]interface{}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			result.ToolCalls = append(result.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: args})
		}
	}
	if or.Usage != nil {
		result.Usage = &EyrieUsage{
			PromptTokens:     or.Usage.PromptTokens,
			CompletionTokens: or.Usage.CompletionTokens,
			TotalTokens:      or.Usage.TotalTokens,
		}
		if or.Usage.PromptTokensDetails != nil {
			result.Usage.CacheReadTokens = or.Usage.PromptTokensDetails.CachedTokens
		}
	}
	return result, nil
}

// StreamChat sends a streaming request.
func (c *OpenAIClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for %s", c.providerName)
	}

	reqBody := c.buildRequest(messages, opts, true)
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: failed to create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: %s stream request failed: %w", c.providerName, err)
	}

	requestID := resp.Header.Get("X-Request-Id")

	if resp.StatusCode != 200 {
		errMsg := parseErrorBody(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("eyrie: %s API error (request_id=%s): %s", c.providerName, requestID, errMsg)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, c.logger)
	events := processOpenAIStream(streamCtx, sseEvents, c.logger)

	return &StreamResult{Events: events, RequestID: requestID, cancel: cancel}, nil
}

// Ping checks connectivity.
func (c *OpenAIClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("eyrie: %s ping failed: %w", c.providerName, err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyrie: %s ping failed: %w", c.providerName, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("eyrie: %s: invalid API key", c.providerName)
	}
	return nil
}
