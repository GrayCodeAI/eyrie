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

type AzureClient struct {
	apiKey     string
	endpoint   string
	apiVersion string
	httpClient *http.Client
	retry      RetryConfig
	logger     *slog.Logger
}

var _ Provider = (*AzureClient)(nil)

func NewAzureClient(apiKey, endpoint, apiVersion string) *AzureClient {
	if apiVersion == "" {
		apiVersion = "2024-08-01-preview"
	}
	return &AzureClient{
		apiKey:     apiKey,
		endpoint:   strings.TrimRight(endpoint, "/"),
		apiVersion: apiVersion,
		httpClient: NewPooledHTTPClient(defaultTimeout),
		retry:      DefaultRetryConfig(),
		logger:     slog.Default(),
	}
}

func (c *AzureClient) Name() string { return "azure" }

func (c *AzureClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("User-Agent", userAgent())
}

func (c *AzureClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for azure")
	}

	reqBody := c.buildRequest(messages, opts, false)
	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", c.endpoint, opts.Model, c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure request creation failed: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	c.logger.Debug("azure chat", "model", opts.Model, "endpoint", c.endpoint)

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("eyrie: azure API error (request_id=%s): %s", requestID, parseErrorBody(resp.Body))
	}

	var or openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("eyrie: azure decode failed: %w", err)
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

func (c *AzureClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	messages = SanitizeMessages(messages)
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for azure")
	}

	reqBody := c.buildRequest(messages, opts, true)
	body, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", c.endpoint, opts.Model, c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure stream request creation failed: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	c.logger.Debug("azure stream", "model", opts.Model, "endpoint", c.endpoint)

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure stream request failed: %w", err)
	}

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != 200 {
		errMsg := parseErrorBody(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("eyrie: azure API error (request_id=%s): %s", requestID, errMsg)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, c.logger)
	events := processOpenAIStream(streamCtx, sseEvents, c.logger)

	return &StreamResult{Events: events, RequestID: requestID, cancel: cancel}, nil
}

func (c *AzureClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/openai/models?api-version=%s", c.endpoint, c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyrie: azure ping failed: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("eyrie: azure: invalid API key")
	}
	return nil
}

func (c *AzureClient) buildRequest(messages []EyrieMessage, opts ChatOptions, stream bool) openaiRequest {
	oaiReq := buildOpenAIRequest(messages, opts, stream)
	return oaiReq
}
