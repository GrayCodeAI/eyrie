package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// OpenAIClient implements Provider for OpenAI and OpenAI-compatible APIs.
type OpenAIClient struct {
	apiKey             string
	baseURL            string
	providerName       string
	compat             *OpenAICompatConfig
	httpClient         *http.Client
	retry              RetryConfig
	logger             *slog.Logger
	defaultModel       string
	defaultMaxTokens   int
	defaultTemperature *float64
	guardrails         *Guardrails
	useMimoAuth        bool
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
		httpClient:   NewPooledHTTPClient(defaultTimeout),
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
	if c.useMimoAuth {
		mimoAuthHeaders(req, c.apiKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("User-Agent", userAgent())
}

func (c *OpenAIClient) setBearerHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Del("api-key")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent())
}

// doRequestWithMimoAuthRetry runs the HTTP request; on 401 with MiMo api-key auth, retries once with Bearer.
func (c *OpenAIClient) doRequestWithMimoAuthRetry(ctx context.Context, req *http.Request, body []byte) (*http.Response, error) {
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
	c.setBearerHeaders(req2)
	if req.Header.Get("Accept") != "" {
		req2.Header.Set("Accept", req.Header.Get("Accept"))
	}
	req2.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	return doWithRetry(ctx, c.httpClient, req2, c.retry, c.logger)
}

type openaiRequest struct {
	Model               string                   `json:"model"`
	Messages            []map[string]interface{} `json:"messages"`
	MaxTokens           *int                     `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                     `json:"max_completion_tokens,omitempty"`
	Temperature         *float64                 `json:"temperature,omitempty"`
	TopP                *float64                 `json:"top_p,omitempty"`
	Stream              bool                     `json:"stream,omitempty"`
	StreamOptions       *streamOptions           `json:"stream_options,omitempty"`
	Tools               []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice          interface{}              `json:"tool_choice,omitempty"`
	ParallelToolCalls   *bool                    `json:"parallel_tool_calls,omitempty"`
	ResponseFormat      map[string]interface{}   `json:"response_format,omitempty"`
	ReasoningEffort     string                   `json:"reasoning_effort,omitempty"`
	Thinking            map[string]interface{}   `json:"thinking,omitempty"`
	Stop                interface{}              `json:"stop,omitempty"`
	ServiceTier         string                   `json:"service_tier,omitempty"`
	User                string                   `json:"user,omitempty"`
	PresencePenalty     *float64                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float64                 `json:"frequency_penalty,omitempty"`
	N                   *int                     `json:"n,omitempty"`
	LogProbs            *bool                    `json:"logprobs,omitempty"`
	TopLogProbs         *int                     `json:"top_logprobs,omitempty"`
	Seed                *int                     `json:"seed,omitempty"`
	Store               *bool                    `json:"store,omitempty"`
	Metadata            map[string]string        `json:"metadata,omitempty"`
	Modalities          []string                 `json:"modalities,omitempty"`
	Audio               map[string]interface{}   `json:"audio,omitempty"`
	Prediction          map[string]interface{}   `json:"prediction,omitempty"`
	WebSearchOptions    map[string]interface{}   `json:"web_search_options,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openaiResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
			ToolCalls        []struct {
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

// openAIToolChoice converts an Anthropic-style ToolChoiceOption to OpenAI wire format.
// Anthropic: {type: "auto"|"any"|"tool"|"none", name: "X"}
// OpenAI:    "auto" | "none" | "required" | {type: "function", function: {name: "X"}}
func openAIToolChoice(tc *ToolChoiceOption) interface{} {
	if tc == nil {
		return nil
	}
	switch tc.Type {
	case "auto":
		return "auto"
	case "none":
		return "none"
	case "any":
		return "required" // Anthropic "any" = OpenAI "required"
	case "tool":
		// Anthropic: {type: "tool", name: "X"}
		// OpenAI:    {type: "function", function: {name: "X"}}
		if tc.Name != "" {
			return map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name": tc.Name,
				},
			}
		}
		return "required"
	default:
		return tc.Type // pass through unknown values
	}
}

// buildRequestBase builds an OpenAI-compatible request body.
// When compat is non-nil, MaxTokensField and SupportsUsageInStreaming overrides are applied.
// EyrieMessage.Thinking (reasoning_content from prior responses) is never forwarded into
// the wire format — providers like DeepSeek return HTTP 400 if it appears in input messages.
func buildRequestBase(messages []EyrieMessage, opts ChatOptions, stream bool, compat *OpenAICompatConfig) openaiRequest {
	var msgs []map[string]interface{}
	for _, m := range messages {
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				toolMsg := map[string]interface{}{
					"role":         "tool",
					"content":      tr.Content,
					"tool_call_id": tr.ToolUseID,
				}
				if tr.IsError {
					toolMsg["is_error"] = true
				}
				msgs = append(msgs, toolMsg)
			}
			continue
		}
		msg := map[string]interface{}{"role": m.Role, "content": m.Content}
		// Handle ContentParts (multi-modal): takes precedence over Content/Images
		if len(m.ContentParts) > 0 {
			content := make([]map[string]interface{}, 0, len(m.ContentParts))
			for _, part := range m.ContentParts {
				switch part.Type {
				case "text":
					content = append(content, map[string]interface{}{"type": "text", "text": part.Text})
				case "image_url":
					if part.ImageURL != nil {
						urlVal := part.ImageURL.URL
						detail := part.ImageURL.Detail
						entry := map[string]interface{}{
							"type":      "image_url",
							"image_url": map[string]interface{}{"url": urlVal},
						}
						if detail != "" {
							entry["image_url"].(map[string]interface{})["detail"] = detail //nolint:errcheck
						}
						content = append(content, entry)
					}
				case "input_audio":
					if part.InputAudio != nil {
						content = append(content, map[string]interface{}{
							"type": "input_audio",
							"input_audio": map[string]interface{}{
								"data":   part.InputAudio.Data,
								"format": part.InputAudio.Format,
							},
						})
					}
				}
			}
			msg["content"] = content
		} else if len(m.Images) > 0 {
			// Legacy Images field: build multi-part content array
			content := make([]map[string]interface{}, 0, 1+len(m.Images))
			if m.Content != "" {
				content = append(content, map[string]interface{}{"type": "text", "text": m.Content})
			}
			for _, img := range m.Images {
				content = append(content, map[string]interface{}{
					"type":      "image_url",
					"image_url": map[string]interface{}{"url": openAIImageURL(img)},
				})
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
	// Kimi/Moonshot context-cache injection (MoonshotAI-Cookbook pattern):
	// prepend a {"role":"cache","content":<id>} message so the provider
	// attaches a previously created context cache to this request.
	// Guarded by SupportsCacheRole so other providers are never affected.
	if compat != nil && compat.SupportsCacheRole && opts.KimiContextCacheID != "" {
		cacheMsg := map[string]interface{}{
			"role":    "cache",
			"content": opts.KimiContextCacheID,
		}
		if opts.KimiCacheResetTTL {
			cacheMsg["reset_ttl"] = true
		}
		msgs = append([]map[string]interface{}{cacheMsg}, msgs...)
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
	if compat != nil && compat.SupportsReasoningEffort && opts.ReasoningEffort != "" {
		req.ReasoningEffort = opts.ReasoningEffort
	}
	// Map ChatOptions fields that apply to all OpenAI-compatible providers
	if opts.TopP != nil {
		req.TopP = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		req.Stop = opts.StopSequences
	}
	if opts.ServiceTier != "" {
		req.ServiceTier = opts.ServiceTier
	}
	if opts.MetadataUserID != "" {
		req.User = opts.MetadataUserID
	}
	if opts.ToolChoice != nil {
		req.ToolChoice = openAIToolChoice(opts.ToolChoice)
	}
	if opts.PresencePenalty != nil {
		req.PresencePenalty = opts.PresencePenalty
	}
	if opts.FrequencyPenalty != nil {
		req.FrequencyPenalty = opts.FrequencyPenalty
	}
	if opts.WebSearchOptions != "" {
		var wso map[string]interface{}
		if json.Unmarshal([]byte(opts.WebSearchOptions), &wso) == nil {
			req.WebSearchOptions = wso
		}
	}
	if opts.Prediction != "" {
		var pred map[string]interface{}
		if json.Unmarshal([]byte(opts.Prediction), &pred) == nil {
			req.Prediction = pred
		}
	}
	if len(opts.Modalities) > 0 {
		req.Modalities = opts.Modalities
	}
	if opts.AudioConfig != "" {
		var audio map[string]interface{}
		if json.Unmarshal([]byte(opts.AudioConfig), &audio) == nil {
			req.Audio = audio
		}
	}
	if opts.N != nil {
		req.N = opts.N
	}
	if opts.LogProbs != nil {
		req.LogProbs = opts.LogProbs
	}
	if opts.TopLogProbs != nil {
		req.TopLogProbs = opts.TopLogProbs
	}
	if opts.Seed != nil {
		req.Seed = opts.Seed
	}
	if opts.Store != nil {
		req.Store = opts.Store
	}
	if len(opts.Metadata) > 0 {
		req.Metadata = opts.Metadata
	}
	if compat != nil && compat.ThinkingFormat == "zai" && opts.GLMThinkingEnabled != nil {
		thinkingType := "disabled"
		if *opts.GLMThinkingEnabled {
			thinkingType = "enabled"
		}
		req.Thinking = map[string]interface{}{"type": thinkingType}
	}
	return req
}

func (c *OpenAIClient) buildRequest(messages []EyrieMessage, opts ChatOptions, stream bool) openaiRequest {
	return buildRequestBase(messages, opts, stream, c.compat)
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

	resp, err := c.doRequestWithMimoAuthRetry(ctx, req, body)
	if err != nil {
		return nil, fmt.Errorf("eyrie: %s request failed: %w", c.providerName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("X-Request-Id")

	if resp.StatusCode != 200 {
		return nil, formatAPIError(c.providerName, resp.StatusCode, requestID, parseProviderError(resp.Body))
	}

	var or openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("eyrie: failed to decode %s response: %w", c.providerName, err)
	}

	result := &EyrieResponse{FinishReason: "unknown", RequestID: requestID}
	if len(or.Choices) > 0 {
		ch := or.Choices[0]
		result.Content = ch.Message.Content
		result.Thinking = ch.Message.ReasoningContent
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

	if err := applyGuardrails(ctx, result, c.guardrails); err != nil {
		return nil, err
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

	resp, err := c.doRequestWithMimoAuthRetry(ctx, req, body)
	if err != nil {
		return nil, fmt.Errorf("eyrie: %s stream request failed: %w", c.providerName, err)
	}

	requestID := resp.Header.Get("X-Request-Id")

	if resp.StatusCode != 200 {
		detail := parseProviderError(resp.Body)
		_ = resp.Body.Close()
		return nil, formatAPIError(c.providerName, resp.StatusCode, requestID, detail)
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
	if c.useMimoAuth && resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		req2, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
		if err != nil {
			return fmt.Errorf("eyrie: %s ping failed: %w", c.providerName, err)
		}
		c.setBearerHeaders(req2)
		resp, err = c.httpClient.Do(req2)
		if err != nil {
			return fmt.Errorf("eyrie: %s ping failed: %w", c.providerName, err)
		}
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("eyrie: %s: invalid API key", c.providerName)
	}
	return nil
}
