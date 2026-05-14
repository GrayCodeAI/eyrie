package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VertexClient struct {
	projectID  string
	region     string
	token      string
	httpClient *http.Client
}

var _ Provider = (*VertexClient)(nil)

func NewVertexClient(projectID, region, token string) *VertexClient {
	return &VertexClient{
		projectID:  projectID,
		region:     region,
		token:      token,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *VertexClient) Name() string { return "anthropic-vertex" }

func (c *VertexClient) baseURL() string {
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models", c.region, c.projectID, c.region)
}

func (c *VertexClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for vertex")
	}
	body, err := c.buildBody(messages, opts, false)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s:rawPredict", c.baseURL(), opts.Model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex request creation failed: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("eyrie: vertex API error (status %d): %s", resp.StatusCode, parseErrorBody(resp.Body))
	}

	var ar struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("eyrie: vertex decode failed: %w", err)
	}

	var content string
	var toolCalls []ToolCall
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			var args map[string]interface{}
			json.Unmarshal(block.Input, &args)
			toolCalls = append(toolCalls, ToolCall{ID: block.ID, Name: block.Name, Arguments: args})
		}
	}

	return &EyrieResponse{
		Content: content, FinishReason: ar.StopReason, ToolCalls: toolCalls,
		Usage: &EyrieUsage{
			PromptTokens:     ar.Usage.InputTokens,
			CompletionTokens: ar.Usage.OutputTokens,
			TotalTokens:      ar.Usage.InputTokens + ar.Usage.OutputTokens,
		},
	}, nil
}

func (c *VertexClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for vertex")
	}
	body, err := c.buildBody(messages, opts, true)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s:streamRawPredict", c.baseURL(), opts.Model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex stream request failed: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyrie: vertex stream failed: %w", err)
	}

	if resp.StatusCode != 200 {
		errMsg := parseErrorBody(resp.Body)
		return nil, fmt.Errorf("eyrie: vertex API error (status %d): %s", resp.StatusCode, errMsg)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	sseEvents := parseSSEStream(streamCtx, resp.Body, nil)
	events := processAnthropicStream(streamCtx, sseEvents, nil)

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
	resp.Body.Close()
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
	reqBody := map[string]interface{}{
		"anthropic_version": "vertex-2023-10-16",
		"model":            opts.Model,
		"max_tokens":       maxTokens,
		"messages":         msgs,
		"stream":           stream,
	}
	if system != "" {
		reqBody["system"] = system
	}
	if opts.Temperature != nil {
		reqBody["temperature"] = *opts.Temperature
	}
	if len(opts.Tools) > 0 {
		reqBody["tools"] = convertToAnthropicTools(opts.Tools)
	}
	return json.Marshal(reqBody)
}

