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

// maxVertexRequestSize is the maximum request body size for Vertex AI (30 MB).
const maxVertexRequestSize = 30 * 1024 * 1024

type VertexClient struct {
	projectID  string
	region     string
	token      string
	httpClient *http.Client
	retry      RetryConfig
	logger     *slog.Logger
	guardrails *Guardrails
}

var _ Provider = (*VertexClient)(nil)

func NewVertexClient(projectID, region, token string) *VertexClient {
	return &VertexClient{
		projectID:  projectID,
		region:     region,
		token:      token,
		httpClient: NewPooledHTTPClient(defaultTimeout),
		retry:      DefaultRetryConfig(),
		logger:     slog.Default().With("component", "vertex"),
	}
}

func (c *VertexClient) Name() string { return "anthropic-vertex" }

func (c *VertexClient) baseURL() string {
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models", c.region, c.projectID, c.region)
}

func (c *VertexClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for vertex")
	}
	body, err := c.buildBody(messages, opts, false)
	if err != nil {
		return nil, err
	}

	// Check request size (30 MB Vertex limit)
	if len(body) > maxVertexRequestSize {
		return nil, fmt.Errorf("eyrie: request size %d bytes exceeds Vertex limit of %d bytes", len(body), maxVertexRequestSize)
	}

	url := fmt.Sprintf("%s/%s:rawPredict", c.baseURL(), opts.Model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex request creation failed: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("X-Goog-Request-Id")

	if resp.StatusCode != 200 {
		return nil, formatAPIError("vertex", resp.StatusCode, requestID, parseProviderError(resp.Body))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("eyrie: vertex decode failed: %w", err)
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
			continue
		case "tool_use":
			var args map[string]interface{}
			_ = json.Unmarshal(block.Input, &args)
			toolCalls = append(toolCalls, ToolCall{ID: block.ID, Name: block.Name, Arguments: args})
		}
	}

	eyrieResp := &EyrieResponse{
		Content: content, Thinking: thinkingContent, FinishReason: ar.StopReason, ToolCalls: toolCalls,
		RequestID: requestID,
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

func (c *VertexClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for vertex")
	}
	body, err := c.buildBody(messages, opts, true)
	if err != nil {
		return nil, err
	}

	// Check request size (30 MB Vertex limit)
	if len(body) > maxVertexRequestSize {
		return nil, fmt.Errorf("eyrie: request size %d bytes exceeds Vertex limit of %d bytes", len(body), maxVertexRequestSize)
	}

	url := fmt.Sprintf("%s/%s:streamRawPredict", c.baseURL(), opts.Model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex stream request failed: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex stream request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		detail := parseProviderError(resp.Body)
		_ = resp.Body.Close()
		return nil, formatAPIError("vertex", resp.StatusCode, resp.Header.Get("X-Goog-Request-Id"), detail)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, c.logger)
	events := processAnthropicStream(streamCtx, sseEvents, c.logger)

	return &StreamResult{Events: events, cancel: cancel}, nil
}

func (c *VertexClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models", c.region, c.projectID, c.region)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyrie: vertex ping failed: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("eyrie: vertex: invalid credentials")
	}
	return nil
}

func (c *VertexClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", userAgent())
}

func (c *VertexClient) buildBody(messages []EyrieMessage, opts ChatOptions, stream bool) ([]byte, error) {
	msgs, system := buildAnthropicMessages(messages)
	if opts.System != "" {
		if system != "" {
			system = opts.System + "\n\n" + system
		} else {
			system = opts.System
		}
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	thinking := resolveThinking(opts)

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

	// Marshal and inject anthropic_version (Vertex-specific)
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	// Add anthropic_version field which is required for Vertex
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	m["anthropic_version"] = "vertex-2023-10-16"
	return json.Marshal(m)
}
