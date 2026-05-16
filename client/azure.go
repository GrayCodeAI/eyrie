package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type AzureClient struct {
	apiKey     string
	endpoint   string
	apiVersion string
	httpClient *http.Client
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
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *AzureClient) Name() string { return "openai-azure" }

func (c *AzureClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for azure")
	}
	oai := NewOpenAIClient(c.apiKey, c.endpoint, nil)
	oai.httpClient = c.httpClient

	body, err := c.buildRequest(messages, opts, false)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", c.endpoint, opts.Model, c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("eyrie: azure API error (status %d): %s", resp.StatusCode, parseErrorBody(resp.Body))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("eyrie: azure decode failed: %w", err)
	}

	result := &EyrieResponse{
		Usage: &EyrieUsage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}
	if len(chatResp.Choices) > 0 {
		choice := chatResp.Choices[0]
		result.Content = choice.Message.Content
		result.FinishReason = choice.FinishReason
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			result.ToolCalls = append(result.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: args})
		}
	}
	return result, nil
}

func (c *AzureClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for azure")
	}

	body, err := c.buildRequest(messages, opts, true)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", c.endpoint, opts.Model, c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure stream request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyrie: azure stream request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		errMsg := parseErrorBody(resp.Body)
		return nil, fmt.Errorf("eyrie: azure API error (status %d): %s", resp.StatusCode, errMsg)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, nil)
	events := processOpenAIStream(streamCtx, sseEvents, nil)

	return &StreamResult{Events: events, cancel: cancel}, nil
}

func (c *AzureClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/openai/models?api-version=%s", c.endpoint, c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("api-key", c.apiKey)
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

func (c *AzureClient) buildRequest(messages []EyrieMessage, opts ChatOptions, stream bool) ([]byte, error) {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type reqBody struct {
		Messages    []msg  `json:"messages"`
		MaxTokens   int    `json:"max_tokens,omitempty"`
		Temperature *float64 `json:"temperature,omitempty"`
		Stream      bool   `json:"stream,omitempty"`
	}
	msgs := make([]msg, 0, len(messages)+1)
	if opts.System != "" {
		msgs = append(msgs, msg{Role: "system", Content: opts.System})
	}
	for _, m := range messages {
		msgs = append(msgs, msg{Role: m.Role, Content: m.Content})
	}
	maxTok := opts.MaxTokens
	if maxTok == 0 {
		maxTok = 4096
	}
	body := reqBody{Messages: msgs, MaxTokens: maxTok, Temperature: opts.Temperature, Stream: stream}
	return json.Marshal(body)
}

