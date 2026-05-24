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
	if opts.Model == "" {
		return nil, fmt.Errorf("eyrie: model is required for bedrock")
	}
	body, err := c.buildBody(messages, opts)
	if err != nil {
		return nil, err
	}
	// Set stream: true for Bedrock invoke-with-response-stream
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(body, &bodyMap); err != nil {
		return nil, err
	}
	bodyMap["stream"] = true
	streamBody, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	streamURL := strings.Replace(c.modelURL(opts.Model), "/invoke", "/invoke-with-response-stream", 1)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, streamURL, bytes.NewReader(streamBody))
	if err != nil {
		return nil, fmt.Errorf("eyrie: bedrock stream request creation failed: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.amazon.eventstream")
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(streamBody)), nil }
	if err := c.sign(req, streamBody, time.Now().UTC()); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req) //nolint:bodyclose // closed inside goroutine for streaming
	if err != nil {
		return nil, fmt.Errorf("eyrie: bedrock stream request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eyrie: bedrock stream error (status %d): %s", resp.StatusCode, parseErrorBody(resp.Body))
	}

	streamCtx, cancel := context.WithCancel(ctx)
	ch := make(chan EyrieStreamEvent, 64)

	go func() {
		defer close(ch)
		defer cancel()
		defer resp.Body.Close() //nolint:errcheck // closed in goroutine for streaming

		var contentBuf strings.Builder
		var toolCalls []ToolCall
		var usage *EyrieUsage
		var finishReason string

		// Bedrock uses Amazon EventStream format. Parse it.
		reader := newEventStreamReader(resp.Body)
		for {
			event, err := reader.ReadEvent()
			if err != nil {
				if err != io.EOF {
					select {
					case ch <- EyrieStreamEvent{Type: "error", Content: err.Error()}:
					case <-streamCtx.Done():
					}
				}
				break
			}

			// Parse the chunk payload
			var chunk anthropicStreamChunk
			if err := json.Unmarshal(event.Payload, &chunk); err != nil {
				continue
			}

			switch chunk.Type {
			case "content_block_delta":
				if chunk.Delta != nil && chunk.Delta.Text != "" {
					contentBuf.WriteString(chunk.Delta.Text)
					select {
					case ch <- EyrieStreamEvent{Type: "content", Content: chunk.Delta.Text}:
					case <-streamCtx.Done():
						return
					}
				}
				if chunk.Delta != nil && chunk.Delta.Type == "input_json_delta" && chunk.Delta.PartialJSON != "" {
					// Accumulate tool input JSON
					select {
					case ch <- EyrieStreamEvent{Type: "tool_input_delta", Content: chunk.Delta.PartialJSON}:
					case <-streamCtx.Done():
						return
					}
				}
			case "content_block_start":
				if chunk.ContentBlock != nil && chunk.ContentBlock.Type == "tool_use" {
					var args map[string]interface{}
					_ = json.Unmarshal(chunk.ContentBlock.Input, &args)
					tc := ToolCall{ID: chunk.ContentBlock.ID, Name: chunk.ContentBlock.Name, Arguments: args}
					toolCalls = append(toolCalls, tc)
					select {
					case ch <- EyrieStreamEvent{Type: "tool_call", ToolCall: &tc}:
					case <-streamCtx.Done():
						return
					}
				}
			case "message_delta":
				if chunk.Delta != nil && chunk.Delta.StopReason != "" {
					finishReason = chunk.Delta.StopReason
				}
			case "message_start":
				if chunk.Message != nil && chunk.Message.Usage != nil {
					usage = chunk.Message.Usage
				}
			}
		}

		// Send final usage event
		if usage != nil {
			select {
			case ch <- EyrieStreamEvent{Type: "usage", Usage: usage}:
			case <-streamCtx.Done():
				return
			}
		}
		select {
		case ch <- EyrieStreamEvent{Type: "done", StopReason: finishReason}:
		case <-streamCtx.Done():
		}
	}()

	return &StreamResult{Events: ch, cancel: cancel, RequestID: resp.Header.Get("X-Amzn-Requestid")}, nil
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

// anthropicStreamChunk represents a chunk from Bedrock's streaming response.
type anthropicStreamChunk struct {
	Type         string `json:"type"`
	Index        int    `json:"index,omitempty"`
	ContentBlock *struct {
		Type  string          `json:"type"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content_block,omitempty"`
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
		StopReason  string `json:"stop_reason,omitempty"`
	} `json:"delta,omitempty"`
	Message *struct {
		Usage *EyrieUsage `json:"usage,omitempty"`
	} `json:"message,omitempty"`
}

// eventStreamReader parses Amazon EventStream binary frames from a reader.
// This is a minimal implementation for Bedrock's invoke-with-response-stream.
type eventStreamReader struct {
	r io.Reader
}

func newEventStreamReader(r io.Reader) *eventStreamReader {
	return &eventStreamReader{r: r}
}

type eventStreamFrame struct {
	Headers map[string]string
	Payload []byte
}

// ReadEvent reads one EventStream frame. Returns io.EOF when the stream ends.
func (es *eventStreamReader) ReadEvent() (*eventStreamFrame, error) {
	// Read prelude: total_length(4) + headers_length(4) + prelude_crc(4)
	var prelude [12]byte
	if _, err := io.ReadFull(es.r, prelude[:]); err != nil {
		return nil, err
	}
	totalLen := int(prelude[0])<<24 | int(prelude[1])<<16 | int(prelude[2])<<8 | int(prelude[3])
	headersLen := int(prelude[4])<<24 | int(prelude[5])<<16 | int(prelude[6])<<8 | int(prelude[7])
	if totalLen < 12 || totalLen > 10*1024*1024 {
		return nil, fmt.Errorf("eventstream: invalid total length %d", totalLen)
	}
	if headersLen > totalLen-12 {
		return nil, fmt.Errorf("eventstream: headers length %d exceeds total %d", headersLen, totalLen)
	}

	// Read remaining bytes: headers + payload + message_crc(4)
	remaining := totalLen - 12
	buf := make([]byte, remaining)
	if _, err := io.ReadFull(es.r, buf); err != nil {
		return nil, err
	}

	// Parse headers (skip for now, we only need the payload)
	payloadStart := headersLen
	if payloadStart > len(buf)-4 {
		return nil, fmt.Errorf("eventstream: invalid headers length")
	}
	payload := buf[payloadStart : len(buf)-4] // exclude trailing CRC

	// Decode headers for content-type
	headers := make(map[string]string)
	headerBuf := buf[:headersLen]
	for len(headerBuf) > 0 {
		if len(headerBuf) < 2 {
			break
		}
		nameLen := int(headerBuf[0])
		if len(headerBuf) < 1+nameLen+1 {
			break
		}
		name := string(headerBuf[1 : 1+nameLen])
		headerBuf = headerBuf[1+nameLen:]
		valueType := headerBuf[0]
		headerBuf = headerBuf[1:]
		switch valueType {
		case 7: // string
			if len(headerBuf) < 2 {
				break
			}
			strLen := int(headerBuf[0])<<8 | int(headerBuf[1])
			if len(headerBuf) < 2+strLen {
				break
			}
			headers[name] = string(headerBuf[2 : 2+strLen])
			headerBuf = headerBuf[2+strLen:]
		case 0: // bool true
			headers[name] = "true"
		case 1: // bool false
			headers[name] = "false"
		default:
			// Skip other types
			return nil, fmt.Errorf("eventstream: unsupported header value type %d", valueType)
		}
	}

	return &eventStreamFrame{Headers: headers, Payload: payload}, nil
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
