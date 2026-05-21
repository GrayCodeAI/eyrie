package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type BedrockClient struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	region          string
	httpClient      *http.Client
}

var _ Provider = (*BedrockClient)(nil)

func NewBedrockClient(accessKeyID, secretAccessKey, sessionToken, region string) *BedrockClient {
	return &BedrockClient{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		sessionToken:    sessionToken,
		region:          region,
		httpClient:      &http.Client{Timeout: defaultTimeout},
	}
}

func (c *BedrockClient) Name() string { return "anthropic-bedrock" }

func (c *BedrockClient) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for bedrock")
	}
	body, err := c.buildBody(messages, opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.modelURL(opts.Model), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("eyrie: bedrock request creation failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	if err := c.sign(req, body, time.Now().UTC()); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyrie: bedrock request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eyrie: bedrock API error (status %d): %s", resp.StatusCode, parseErrorBody(resp.Body))
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("eyrie: bedrock decode failed: %w", err)
	}
	return responseFromAnthropic(ar, resp.Header.Get("X-Amzn-Requestid")), nil
}

func (c *BedrockClient) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	resp, err := c.Chat(ctx, messages, opts)
	if err != nil {
		return nil, err
	}
	out := make(chan EyrieStreamEvent, 3+len(resp.ToolCalls))
	out <- EyrieStreamEvent{Type: "content", Content: resp.Content}
	for i := range resp.ToolCalls {
		toolCall := resp.ToolCalls[i]
		out <- EyrieStreamEvent{Type: "tool_call", ToolCall: &toolCall}
	}
	if resp.Usage != nil {
		out <- EyrieStreamEvent{Type: "usage", Usage: resp.Usage}
	}
	out <- EyrieStreamEvent{Type: "done", StopReason: resp.FinishReason}
	close(out)
	return &StreamResult{Events: out, RequestID: resp.RequestID}, nil
}

func (c *BedrockClient) Ping(ctx context.Context) error {
	if c.region == "" || c.accessKeyID == "" || c.secretAccessKey == "" {
		return fmt.Errorf("eyrie: bedrock credentials are incomplete")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://bedrock.%s.amazonaws.com/foundation-models", c.region), nil)
	if err != nil {
		return err
	}
	if err := c.sign(req, nil, time.Now().UTC()); err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("eyrie: bedrock ping failed: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("eyrie: bedrock: invalid credentials")
	}
	return nil
}

func (c *BedrockClient) buildBody(messages []EyrieMessage, opts ChatOptions) ([]byte, error) {
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
	reqBody := anthropicRequest{
		Model:       opts.Model,
		MaxTokens:   maxTokens,
		Messages:    msgs,
		System:      system,
		Temperature: opts.Temperature,
		Tools:       convertToAnthropicTools(opts.Tools),
	}
	return json.Marshal(reqBody)
}

func (c *BedrockClient) modelURL(model string) string {
	return fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", c.region, url.PathEscape(model))
}

func (c *BedrockClient) sign(req *http.Request, body []byte, now time.Time) error {
	if c.region == "" || c.accessKeyID == "" || c.secretAccessKey == "" {
		return fmt.Errorf("eyrie: bedrock credentials are incomplete")
	}
	service := "bedrock"
	if strings.HasPrefix(req.URL.Host, "bedrock-runtime.") {
		service = "bedrock"
	}
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(body)
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if c.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", c.sessionToken)
	}
	canonicalHeaders, signedHeaders := canonicalAWSHeaders(req.Header)
	canonicalRequest := strings.Join([]string{
		req.Method,
		awsCanonicalURI(req.URL.EscapedPath()),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := dateStamp + "/" + c.region + "/" + service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signingKey := awsSigningKey(c.secretAccessKey, dateStamp, c.region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", c.accessKeyID, scope, signedHeaders, signature))
	return nil
}

func responseFromAnthropic(ar anthropicResponse, requestID string) *EyrieResponse {
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
		Content: content, FinishReason: ar.StopReason, ToolCalls: toolCalls, RequestID: requestID,
		Usage: &EyrieUsage{
			PromptTokens:        ar.Usage.InputTokens,
			CompletionTokens:    ar.Usage.OutputTokens,
			TotalTokens:         ar.Usage.InputTokens + ar.Usage.OutputTokens,
			CacheCreationTokens: ar.Usage.CacheCreationInputTokens,
			CacheReadTokens:     ar.Usage.CacheReadInputTokens,
		},
	}
}

func canonicalAWSHeaders(headers http.Header) (string, string) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for _, lower := range keys {
		values := headers.Values(lower)
		if len(values) == 0 {
			values = headers.Values(http.CanonicalHeaderKey(lower))
		}
		for i := range values {
			values[i] = strings.Join(strings.Fields(values[i]), " ")
		}
		canonical.WriteString(lower)
		canonical.WriteByte(':')
		canonical.WriteString(strings.Join(values, ","))
		canonical.WriteByte('\n')
	}
	return canonical.String(), strings.Join(keys, ";")
}

func awsCanonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func awsSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}
