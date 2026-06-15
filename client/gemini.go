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

// GeminiClient implements Provider for the Google Gemini API.
type GeminiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	retry      RetryConfig
	logger     *slog.Logger
}

var _ Provider = (*GeminiClient)(nil)

func NewGeminiClient(apiKey, baseURL string) *GeminiClient {
	c := &GeminiClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: NewPooledHTTPClient(defaultTimeout),
		retry:      DefaultRetryConfig(),
		logger:     slog.Default().With("component", "gemini"),
	}
	if c.baseURL == "" {
		c.baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	return c
}

func (c *GeminiClient) Name() string { return "gemini" }

func (c *GeminiClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent())
	if c.isVertex() {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else {
		req.Header.Set("x-goog-api-key", c.apiKey)
	}
}

func (c *GeminiClient) isVertex() bool {
	return strings.Contains(c.baseURL, "aiplatform.googleapis.com")
}

func (c *GeminiClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for gemini")
	}
	body, err := c.buildBody(messages, opts)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, opts.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: gemini request creation failed: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: gemini request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("eyrie: gemini returned %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eyrie: gemini response read failed: %w", err)
	}

	return c.parseResponse(respBody)
}

func (c *GeminiClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for gemini")
	}
	body, err := c.buildBody(messages, opts)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", c.baseURL, opts.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: gemini stream request creation failed: %w", err)
	}
	c.setHeaders(req)
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := doWithRetry(ctx, c.httpClient, req, c.retry, c.logger)
	if err != nil {
		return nil, fmt.Errorf("eyrie: gemini stream request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("eyrie: gemini stream returned %d: %s", resp.StatusCode, string(respBody))
	}

	streamCtx, cancel := context.WithCancel(ctx)
	events := make(chan EyrieStreamEvent, 64)
	go c.streamLoop(streamCtx, resp.Body, events)

	return NewStreamResult(events, cancel), nil
}

func (c *GeminiClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/models", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("eyrie: gemini ping request creation failed: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyrie: gemini ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("eyrie: gemini ping returned %d", resp.StatusCode)
	}
	return nil
}

// --- Request building ---

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	ToolConfig        *geminiToolConfig       `json:"toolConfig,omitempty"`
	SafetySettings    []geminiSafetySetting   `json:"safetySettings,omitempty"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	InlineData       *geminiInlineData       `json:"inlineData,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	ID   string                 `json:"id,omitempty"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens  int                    `json:"maxOutputTokens,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"topP,omitempty"`
	TopK             *int                   `json:"topK,omitempty"`
	StopSequences    []string               `json:"stopSequences,omitempty"`
	ResponseMimeType string                 `json:"responseMimeType,omitempty"`
	ResponseSchema   map[string]interface{} `json:"responseSchema,omitempty"`
	ThinkingConfig   *geminiThinkingConfig  `json:"thinkingConfig,omitempty"`
	PresencePenalty  *float64               `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequencyPenalty,omitempty"`
	Seed             *int                   `json:"seed,omitempty"`
	ResponseLogprobs *bool                  `json:"responseLogprobs,omitempty"`
	Logprobs         *int                   `json:"logprobs,omitempty"`
	CandidateCount   *int                   `json:"candidateCount,omitempty"`
}

type geminiThinkingConfig struct {
	ThinkingLevel string `json:"thinkingLevel,omitempty"` // "low", "medium", "high"
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// geminiToolConfig controls function calling behavior.
type geminiToolConfig struct {
	FunctionCallingConfig *geminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type geminiFunctionCallingConfig struct {
	Mode string `json:"mode"` // "AUTO", "ANY", "NONE"
}

// geminiSafetySetting configures content safety filtering.
type geminiSafetySetting struct {
	Category  string `json:"category"`  // e.g., "HARM_CATEGORY_HARASSMENT"
	Threshold string `json:"threshold"` // e.g., "BLOCK_NONE", "BLOCK_LOW_AND_ABOVE", etc.
}

func (c *GeminiClient) buildBody(messages []EyrieMessage, opts ChatOptions) ([]byte, error) {
	contents := make([]geminiContent, 0, len(messages))
	var systemInstruction *geminiContent

	// Build a lookup of tool call ID → function name from conversation history
	toolNameByID := map[string]string{}
	for _, msg := range messages {
		for _, tc := range msg.ToolUse {
			if tc.ID != "" && tc.Name != "" {
				toolNameByID[tc.ID] = tc.Name
			}
		}
	}

	for _, msg := range messages {
		gc := geminiContent{Parts: make([]geminiPart, 0, 2)}
		switch msg.Role {
		case "user":
			gc.Role = "user"
		case "assistant":
			gc.Role = "model"
		case "system":
			systemInstruction = &geminiContent{Parts: []geminiPart{{Text: msg.Content}}}
			continue
		default:
			gc.Role = "user"
		}
		if msg.Content != "" {
			gc.Parts = append(gc.Parts, geminiPart{Text: msg.Content})
		}
		// Handle ContentParts (multi-modal)
		if len(msg.ContentParts) > 0 {
			for _, part := range msg.ContentParts {
				switch part.Type {
				case "text":
					gc.Parts = append(gc.Parts, geminiPart{Text: part.Text})
				case "image_url":
					if part.ImageURL != nil {
						mimeType, data, isBase64 := parseImageString(part.ImageURL.URL)
						if !isBase64 {
							mimeType = "image/png"
							data = part.ImageURL.URL
						}
						gc.Parts = append(gc.Parts, geminiPart{
							InlineData: &geminiInlineData{MimeType: mimeType, Data: data},
						})
					}
				case "input_audio":
					if part.InputAudio != nil {
						mediaType := audioFormatToMediaType(part.InputAudio.Format)
						gc.Parts = append(gc.Parts, geminiPart{
							InlineData: &geminiInlineData{MimeType: mediaType, Data: part.InputAudio.Data},
						})
					}
				}
			}
		} else {
			for _, img := range msg.Images {
				gc.Parts = append(gc.Parts, geminiPart{
					InlineData: &geminiInlineData{MimeType: "image/png", Data: img},
				})
			}
		}
		// Tool results — use function name from conversation history, not tool_use_id
		if len(msg.ToolResults) > 0 {
			gc.Role = "user"
			for _, tr := range msg.ToolResults {
				funcName := toolNameByID[tr.ToolUseID]
				if funcName == "" {
					funcName = tr.ToolUseID // fallback for legacy format
				}
				fr := geminiFunctionResponse{
					Name:     funcName,
					Response: map[string]string{"content": tr.Content},
				}
				// Gemini 3+ requires the tool call ID
				if tr.ToolUseID != "" {
					fr.ID = tr.ToolUseID
				}
				gc.Parts = append(gc.Parts, geminiPart{FunctionResponse: &fr})
			}
		}
		// Tool calls — include ID for Gemini 3+
		for _, tc := range msg.ToolUse {
			fc := geminiFunctionCall{Name: tc.Name, Args: tc.Arguments}
			if tc.ID != "" {
				fc.ID = tc.ID
			}
			gc.Parts = append(gc.Parts, geminiPart{FunctionCall: &fc})
		}
		contents = append(contents, gc)
	}

	req := geminiRequest{Contents: contents, SystemInstruction: systemInstruction}

	// Generation config
	if opts.MaxTokens > 0 || opts.Temperature != nil || opts.TopP != nil || len(opts.StopSequences) > 0 || opts.TopK != nil {
		req.GenerationConfig = &geminiGenerationConfig{
			MaxOutputTokens: opts.MaxTokens,
			Temperature:     opts.Temperature,
			TopP:            opts.TopP,
			TopK:            opts.TopK,
			StopSequences:   opts.StopSequences,
		}
		// Structured output support
		if opts.ResponseFormat != nil && opts.ResponseFormat.Type == "json_schema" && opts.ResponseFormat.Schema != "" {
			req.GenerationConfig.ResponseMimeType = "application/json"
			var schema map[string]interface{}
			if json.Unmarshal([]byte(opts.ResponseFormat.Schema), &schema) == nil {
				req.GenerationConfig.ResponseSchema = schema
			}
		}
	}

	// Tools
	if len(opts.Tools) > 0 {
		decls := make([]geminiFunctionDecl, len(opts.Tools))
		for i, t := range opts.Tools {
			decls[i] = geminiFunctionDecl(t)
		}
		req.Tools = []geminiTool{{FunctionDeclarations: decls}}
	}

	// Tool config — map Anthropic-style tool_choice to Gemini function calling mode
	if opts.ToolChoice != nil {
		mode := "AUTO"
		switch opts.ToolChoice.Type {
		case "any":
			mode = "ANY"
		case "none":
			mode = "NONE"
		case "auto":
			mode = "AUTO"
		}
		req.ToolConfig = &geminiToolConfig{
			FunctionCallingConfig: &geminiFunctionCallingConfig{Mode: mode},
		}
	}

	// Safety settings — disable all by default for unrestricted use
	req.SafetySettings = []geminiSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "BLOCK_NONE"},
	}

	// Map additional ChatOptions to generationConfig
	if req.GenerationConfig == nil && (opts.PresencePenalty != nil || opts.FrequencyPenalty != nil || opts.Seed != nil || opts.N != nil || opts.LogProbs != nil || opts.TopLogProbs != nil) {
		req.GenerationConfig = &geminiGenerationConfig{}
	}
	if req.GenerationConfig != nil {
		if opts.PresencePenalty != nil {
			req.GenerationConfig.PresencePenalty = opts.PresencePenalty
		}
		if opts.FrequencyPenalty != nil {
			req.GenerationConfig.FrequencyPenalty = opts.FrequencyPenalty
		}
		if opts.Seed != nil {
			req.GenerationConfig.Seed = opts.Seed
		}
		if opts.N != nil {
			req.GenerationConfig.CandidateCount = opts.N
		}
		if opts.LogProbs != nil {
			req.GenerationConfig.ResponseLogprobs = opts.LogProbs
		}
		if opts.TopLogProbs != nil {
			req.GenerationConfig.Logprobs = opts.TopLogProbs
		}
	}

	return json.Marshal(req)
}

// --- Response parsing ---

type geminiResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	Usage          *geminiUsage          `json:"usageMetadata,omitempty"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsage struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount,omitempty"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
	ToolUsePromptTokenCount int `json:"toolUsePromptTokenCount,omitempty"`
}

type geminiPromptFeedback struct {
	BlockReason        string `json:"blockReason,omitempty"`
	BlockReasonMessage string `json:"blockReasonMessage,omitempty"`
}

func (c *GeminiClient) parseResponse(data []byte) (*EyrieResponse, error) {
	var gr geminiResponse
	if err := json.Unmarshal(data, &gr); err != nil {
		return nil, fmt.Errorf("eyrie: gemini response parse failed: %w", err)
	}
	if len(gr.Candidates) == 0 {
		// Check if the prompt was blocked
		if gr.PromptFeedback != nil && gr.PromptFeedback.BlockReason != "" {
			msg := gr.PromptFeedback.BlockReason
			if gr.PromptFeedback.BlockReasonMessage != "" {
				msg = gr.PromptFeedback.BlockReasonMessage
			}
			return nil, fmt.Errorf("eyrie: gemini blocked prompt: %s", msg)
		}
		return nil, fmt.Errorf("eyrie: gemini returned no candidates")
	}

	candidate := gr.Candidates[0]
	resp := &EyrieResponse{
		FinishReason: mapGeminiFinishReason(candidate.FinishReason),
	}

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			resp.Content += part.Text
		}
		if part.FunctionCall != nil {
			tc := ToolCall{
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			}
			if part.FunctionCall.ID != "" {
				tc.ID = part.FunctionCall.ID
			}
			resp.ToolCalls = append(resp.ToolCalls, tc)
		}
	}

	if gr.Usage != nil {
		resp.Usage = &EyrieUsage{
			PromptTokens:     gr.Usage.PromptTokenCount,
			CompletionTokens: gr.Usage.CandidatesTokenCount,
			TotalTokens:      gr.Usage.TotalTokenCount,
			ThinkingTokens:   gr.Usage.ThoughtsTokenCount,
			CacheReadTokens:  gr.Usage.CachedContentTokenCount,
		}
	}

	return resp, nil
}

func mapGeminiFinishReason(reason string) string {
	switch strings.ToUpper(reason) {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	case "SAFETY":
		return "content_filter"
	default:
		return reason
	}
}

// --- Streaming ---

func (c *GeminiClient) streamLoop(ctx context.Context, body io.ReadCloser, events chan<- EyrieStreamEvent) {
	defer close(events)
	defer func() { _ = body.Close() }()

	doneSent := false
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 4096)
	for {
		n, readErr := body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					break
				}
				line := string(buf[:idx])
				buf = buf[idx+1:]
				line = strings.TrimSpace(line)
				if line == "" || !strings.HasPrefix(line, "data: ") {
					continue
				}
				data := strings.TrimPrefix(line, "data: ")
				if c.processStreamChunk(ctx, data, events) {
					doneSent = true
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	if !doneSent {
		select {
		case events <- EyrieStreamEvent{Type: "done"}:
		case <-ctx.Done():
		}
	}
}

func (c *GeminiClient) processStreamChunk(ctx context.Context, data string, events chan<- EyrieStreamEvent) bool {
	var chunk geminiResponse
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return false
	}
	if len(chunk.Candidates) == 0 {
		return false
	}
	candidate := chunk.Candidates[0]
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			select {
			case events <- EyrieStreamEvent{Type: "content", Content: part.Text}:
			case <-ctx.Done():
				return false
			}
		}
		if part.FunctionCall != nil {
			tc := &ToolCall{
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			}
			if part.FunctionCall.ID != "" {
				tc.ID = part.FunctionCall.ID
			}
			select {
			case events <- EyrieStreamEvent{
				Type:     "tool_call",
				ToolCall: tc,
			}:
			case <-ctx.Done():
				return false
			}
		}
	}
	if chunk.Usage != nil {
		select {
		case events <- EyrieStreamEvent{
			Type: "done",
			Usage: &EyrieUsage{
				PromptTokens:     chunk.Usage.PromptTokenCount,
				CompletionTokens: chunk.Usage.CandidatesTokenCount,
				TotalTokens:      chunk.Usage.TotalTokenCount,
				ThinkingTokens:   chunk.Usage.ThoughtsTokenCount,
				CacheReadTokens:  chunk.Usage.CachedContentTokenCount,
			},
		}:
			return true
		case <-ctx.Done():
		}
	}
	return false
}
