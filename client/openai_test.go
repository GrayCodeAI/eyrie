package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Helpers ---

func newTestOpenAIClient(url string, compat *OpenAICompatConfig) *OpenAIClient {
	rc := RetryConfig{MaxRetries: 0, BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond, RetryOn: []int{}}
	return NewOpenAIClient("test-key", url, compat, WithRetry(rc))
}

func defaultChatOpts() ChatOptions {
	return ChatOptions{Model: "gpt-4o"}
}

func basicMessages() []EyrieMessage {
	return []EyrieMessage{{Role: "user", Content: "Hello"}}
}

// --- TestOpenAIChat ---

func TestOpenAIChat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected content-type: %s", ct)
		}

		// Verify request body
		var reqBody openaiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if reqBody.Model != "gpt-4o" {
			t.Errorf("unexpected model: %s", reqBody.Model)
		}
		if reqBody.Stream {
			t.Error("expected stream to be false")
		}

		w.Header().Set("X-Request-Id", "req-123")
		json.NewEncoder(w).Encode(openaiResponse{
			ID: "chatcmpl-abc",
			Choices: []struct {
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
			}{
				{
					Message: struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							ID       string `json:"id"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls,omitempty"`
					}{Content: "Hello! How can I help you?"},
					FinishReason: "stop",
				},
			},
			Usage: &struct {
				PromptTokens        int `json:"prompt_tokens"`
				CompletionTokens    int `json:"completion_tokens"`
				TotalTokens         int `json:"total_tokens"`
				PromptTokensDetails *struct {
					CachedTokens int `json:"cached_tokens"`
				} `json:"prompt_tokens_details,omitempty"`
			}{
				PromptTokens: 10, CompletionTokens: 8, TotalTokens: 18,
			},
		})
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("unexpected finish_reason: %s", resp.FinishReason)
	}
	if resp.RequestID != "req-123" {
		t.Errorf("unexpected request_id: %s", resp.RequestID)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 8 || resp.Usage.TotalTokens != 18 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestOpenAIChat_MissingModel(t *testing.T) {
	c := newTestOpenAIClient("http://localhost", nil)
	_, err := c.Chat(context.Background(), basicMessages(), ChatOptions{})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenAIChat_WithUsageCacheDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-cached")
		resp := map[string]interface{}{
			"id": "chatcmpl-cached",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "ok"}, "finish_reason": "stop"},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     100,
				"completion_tokens": 20,
				"total_tokens":      120,
				"prompt_tokens_details": map[string]interface{}{
					"cached_tokens": 50,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage")
	}
	if resp.Usage.CacheReadTokens != 50 {
		t.Errorf("expected CacheReadTokens=50, got %d", resp.Usage.CacheReadTokens)
	}
}

// --- TestOpenAIChat_ToolCalls ---

func TestOpenAIChat_ToolCallsInResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tools are sent in request
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		tools, ok := reqBody["tools"].([]interface{})
		if !ok || len(tools) != 1 {
			t.Errorf("expected 1 tool in request, got %v", reqBody["tools"])
		}

		w.Header().Set("X-Request-Id", "req-tools")
		resp := map[string]interface{}{
			"id": "chatcmpl-tools",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_abc123",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_weather",
									"arguments": `{"location":"San Francisco","unit":"celsius"}`,
								},
							},
							{
								"id":   "call_def456",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_time",
									"arguments": `{"timezone":"PST"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 50, "completion_tokens": 30, "total_tokens": 80,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	opts := ChatOptions{
		Model: "gpt-4o",
		Tools: []EyrieTool{
			{
				Name:        "get_weather",
				Description: "Get the current weather",
				Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{"location": map[string]interface{}{"type": "string"}}},
			},
		},
	}
	resp, err := c.Chat(context.Background(), basicMessages(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("expected finish_reason=tool_calls, got %s", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_abc123" {
		t.Errorf("unexpected tool call id: %s", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("unexpected tool call name: %s", tc.Name)
	}
	if tc.Arguments["location"] != "San Francisco" {
		t.Errorf("unexpected arguments: %v", tc.Arguments)
	}
	tc2 := resp.ToolCalls[1]
	if tc2.Name != "get_time" {
		t.Errorf("unexpected second tool call name: %s", tc2.Name)
	}
}

// --- TestOpenAIChat_ResponseFormat ---

func TestOpenAIChat_ResponseFormatJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		rf, ok := reqBody["response_format"].(map[string]interface{})
		if !ok {
			t.Fatal("expected response_format in request")
		}
		if rf["type"] != "json_object" {
			t.Errorf("expected type=json_object, got %v", rf["type"])
		}

		w.Header().Set("X-Request-Id", "req-json")
		resp := map[string]interface{}{
			"id": "chatcmpl-json",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": `{"name":"test"}`}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	opts := ChatOptions{
		Model:          "gpt-4o",
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}
	resp, err := c.Chat(context.Background(), basicMessages(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != `{"name":"test"}` {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

func TestOpenAIChat_ResponseFormatJSONSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		rf, ok := reqBody["response_format"].(map[string]interface{})
		if !ok {
			t.Fatal("expected response_format in request")
		}
		if rf["type"] != "json_schema" {
			t.Errorf("expected type=json_schema, got %v", rf["type"])
		}
		if _, ok := rf["json_schema"]; !ok {
			t.Error("expected json_schema field in response_format")
		}

		w.Header().Set("X-Request-Id", "req-schema")
		resp := map[string]interface{}{
			"id": "chatcmpl-schema",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": `{"result":42}`}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	opts := ChatOptions{
		Model: "gpt-4o",
		ResponseFormat: &ResponseFormat{
			Type:   "json_schema",
			Schema: `{"type":"object","properties":{"result":{"type":"integer"}}}`,
		},
	}
	resp, err := c.Chat(context.Background(), basicMessages(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != `{"result":42}` {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

// --- TestOpenAIChat_ErrorHandling ---

func TestOpenAIChat_Error401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-401")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Incorrect API key provided",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	_, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected API error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Incorrect API key") {
		t.Errorf("expected error message about API key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "req-401") {
		t.Errorf("expected request_id in error, got: %v", err)
	}
}

func TestOpenAIChat_Error429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-429")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	}))
	defer srv.Close()

	// Use no-retry config so 429 returns immediately as error
	c := newTestOpenAIClient(srv.URL, nil)
	_, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "Rate limit exceeded") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestOpenAIChat_Error500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-500")
		w.WriteHeader(500)
		fmt.Fprint(w, `{"error":{"message":"Internal server error","type":"server_error"}}`)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	_, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "Internal server error") {
		t.Errorf("expected server error, got: %v", err)
	}
}

func TestOpenAIChat_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := c.Chat(ctx, basicMessages(), defaultChatOpts())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- TestOpenAIStreamChat ---

func TestOpenAIStreamChat_Success(t *testing.T) {
	sseData := []string{
		`data: {"id":"chatcmpl-stream","choices":[{"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-stream","choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-stream","choices":[{"delta":{"content":" world"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-stream","choices":[{"delta":{},"finish_reason":"stop"}]}`,
		"",
		"data: [DONE]",
		"",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["stream"] != true {
			t.Error("expected stream=true in request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-Id", "req-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for _, line := range sseData {
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	sr, err := c.StreamChat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	if sr.RequestID != "req-stream" {
		t.Errorf("unexpected request_id: %s", sr.RequestID)
	}

	var content strings.Builder
	var gotDone bool
	for evt := range sr.Events {
		switch evt.Type {
		case "content":
			content.WriteString(evt.Content)
		case "done":
			gotDone = true
			if evt.StopReason != "stop" {
				t.Errorf("expected stop_reason=stop, got %s", evt.StopReason)
			}
		case "error":
			t.Errorf("unexpected error event: %s", evt.Error)
		}
	}
	if !gotDone {
		t.Error("expected done event")
	}
	if content.String() != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", content.String())
	}
}

func TestOpenAIStreamChat_ToolCalls(t *testing.T) {
	sseData := []string{
		`data: {"id":"chatcmpl-tc","choices":[{"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-tc","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-tc","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\""}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-tc","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"main.go\"}"}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-tc","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		"",
		"data: [DONE]",
		"",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-Id", "req-stream-tc")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for _, line := range sseData {
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	sr, err := c.StreamChat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	var toolCalls []ToolCall
	var gotDone bool
	for evt := range sr.Events {
		switch evt.Type {
		case "tool_call":
			if evt.ToolCall != nil {
				toolCalls = append(toolCalls, *evt.ToolCall)
			}
		case "done":
			gotDone = true
		case "error":
			t.Errorf("unexpected error event: %s", evt.Error)
		}
	}
	if !gotDone {
		t.Error("expected done event")
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	tc := toolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("unexpected tool call id: %s", tc.ID)
	}
	if tc.Name != "read_file" {
		t.Errorf("unexpected tool call name: %s", tc.Name)
	}
	if tc.Arguments["path"] != "main.go" {
		t.Errorf("unexpected arguments: %v", tc.Arguments)
	}
}

func TestOpenAIStreamChat_MultipleToolCalls(t *testing.T) {
	sseData := []string{
		`data: {"id":"chatcmpl-mtc","choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-mtc","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"tool_a","arguments":""}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-mtc","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"x\":1}"}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-mtc","choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_2","function":{"name":"tool_b","arguments":""}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-mtc","choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"y\":2}"}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-mtc","choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		"",
		"data: [DONE]",
		"",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-Id", "req-multi-tc")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for _, line := range sseData {
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	sr, err := c.StreamChat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	var toolCalls []ToolCall
	for evt := range sr.Events {
		if evt.Type == "tool_call" && evt.ToolCall != nil {
			toolCalls = append(toolCalls, *evt.ToolCall)
		}
	}
	if len(toolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(toolCalls))
	}
	if toolCalls[0].Name != "tool_a" || toolCalls[1].Name != "tool_b" {
		t.Errorf("unexpected tool names: %s, %s", toolCalls[0].Name, toolCalls[1].Name)
	}
}

func TestOpenAIStreamChat_WithUsage(t *testing.T) {
	sseData := []string{
		`data: {"id":"chatcmpl-u","choices":[{"delta":{"content":"Hi"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl-u","choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`,
		"",
		"data: [DONE]",
		"",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify stream_options.include_usage is set when compat supports it
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		so, ok := reqBody["stream_options"]
		if !ok {
			t.Error("expected stream_options in request")
		} else {
			soMap := so.(map[string]interface{})
			if soMap["include_usage"] != true {
				t.Error("expected include_usage=true")
			}
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-Id", "req-usage")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for _, line := range sseData {
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	// Use OpenAICompat which supports usage in streaming
	c := newTestOpenAIClient(srv.URL, &OpenAICompat)
	sr, err := c.StreamChat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	var gotUsage bool
	for evt := range sr.Events {
		if evt.Type == "usage" && evt.Usage != nil {
			gotUsage = true
			if evt.Usage.PromptTokens != 5 || evt.Usage.CompletionTokens != 1 {
				t.Errorf("unexpected usage: %+v", evt.Usage)
			}
		}
	}
	if !gotUsage {
		t.Error("expected usage event")
	}
}

func TestOpenAIStreamChat_MissingModel(t *testing.T) {
	c := newTestOpenAIClient("http://localhost", nil)
	_, err := c.StreamChat(context.Background(), basicMessages(), ChatOptions{})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenAIStreamChat_Error401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-stream-401")
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"message":"Invalid auth","type":"auth_error"}}`)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	_, err := c.StreamChat(context.Background(), basicMessages(), defaultChatOpts())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "Invalid auth") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

// --- TestOpenAIPing ---

func TestOpenAIPing_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("expected /models, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", auth)
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIPing_InvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"message":"invalid key"}}`)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	err := c.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenAIPing_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	// 500 != 401, so Ping should succeed (it only checks for 401)
	err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error (500 should pass ping): %v", err)
	}
}

// --- TestOpenAI_CompatOverrides ---

func TestOpenAICompat_MaxTokensField(t *testing.T) {
	tests := []struct {
		name       string
		compat     *OpenAICompatConfig
		wantKey    string
		notWantKey string
	}{
		{
			name:       "openai uses max_completion_tokens",
			compat:     &OpenAICompat,
			wantKey:    "max_completion_tokens",
			notWantKey: "max_tokens",
		},
		{
			name:       "grok uses max_tokens",
			compat:     &GrokCompat,
			wantKey:    "max_tokens",
			notWantKey: "max_completion_tokens",
		},
		{
			name:       "ollama uses max_tokens",
			compat:     &OllamaCompat,
			wantKey:    "max_tokens",
			notWantKey: "max_completion_tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var reqBody map[string]interface{}
				json.Unmarshal(body, &reqBody)

				if _, ok := reqBody[tt.wantKey]; !ok {
					t.Errorf("expected %s in request body", tt.wantKey)
				}
				if _, ok := reqBody[tt.notWantKey]; ok {
					t.Errorf("unexpected %s in request body", tt.notWantKey)
				}

				w.Header().Set("X-Request-Id", "req-compat")
				resp := map[string]interface{}{
					"id": "chatcmpl-compat",
					"choices": []map[string]interface{}{
						{"message": map[string]interface{}{"content": "ok"}, "finish_reason": "stop"},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()

			c := newTestOpenAIClient(srv.URL, tt.compat)
			_, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOpenAICompat_StreamOptionsNotSentWhenUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		if _, ok := reqBody["stream_options"]; ok {
			t.Error("stream_options should not be sent when SupportsUsageInStreaming is false")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-Id", "req-no-so")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"id\":\"x\",\"choices\":[{\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	// GrokCompat has SupportsUsageInStreaming=false
	c := newTestOpenAIClient(srv.URL, &GrokCompat)
	sr, err := c.StreamChat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()
	// Drain events
	for range sr.Events {
	}
}

// --- TestOpenAI_ImageContent ---

func TestOpenAIChat_ImageContent_DataURI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		msgs := reqBody["messages"].([]interface{})
		msg := msgs[0].(map[string]interface{})
		content := msg["content"].([]interface{})

		if len(content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(content))
		}
		textPart := content[0].(map[string]interface{})
		if textPart["type"] != "text" || textPart["text"] != "Describe this image" {
			t.Errorf("unexpected text part: %v", textPart)
		}
		imgPart := content[1].(map[string]interface{})
		if imgPart["type"] != "image_url" {
			t.Errorf("expected image_url type, got %v", imgPart["type"])
		}
		imgURL := imgPart["image_url"].(map[string]interface{})
		if imgURL["url"] != "data:image/png;base64,iVBORw0KGgoAAAA" {
			t.Errorf("unexpected image url: %v", imgURL["url"])
		}

		w.Header().Set("X-Request-Id", "req-img")
		resp := map[string]interface{}{
			"id": "chatcmpl-img",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "A cat"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	msgs := []EyrieMessage{
		{Role: "user", Content: "Describe this image", Images: []string{"data:image/png;base64,iVBORw0KGgoAAAA"}},
	}
	resp, err := c.Chat(context.Background(), msgs, defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "A cat" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

func TestOpenAIChat_ImageContent_HTTPUrl(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		msgs := reqBody["messages"].([]interface{})
		msg := msgs[0].(map[string]interface{})
		content := msg["content"].([]interface{})

		imgPart := content[0].(map[string]interface{})
		imgURL := imgPart["image_url"].(map[string]interface{})
		if imgURL["url"] != "https://example.com/image.png" {
			t.Errorf("unexpected image url: %v", imgURL["url"])
		}

		w.Header().Set("X-Request-Id", "req-img-url")
		resp := map[string]interface{}{
			"id": "chatcmpl-img2",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "An image"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	msgs := []EyrieMessage{
		{Role: "user", Images: []string{"https://example.com/image.png"}},
	}
	resp, err := c.Chat(context.Background(), msgs, defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "An image" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

func TestOpenAIChat_ImageContent_RawBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		msgs := reqBody["messages"].([]interface{})
		msg := msgs[0].(map[string]interface{})
		content := msg["content"].([]interface{})

		imgPart := content[0].(map[string]interface{})
		imgURL := imgPart["image_url"].(map[string]interface{})
		expected := "data:image/png;base64,AAAA"
		if imgURL["url"] != expected {
			t.Errorf("expected %q, got %v", expected, imgURL["url"])
		}

		w.Header().Set("X-Request-Id", "req-img-raw")
		resp := map[string]interface{}{
			"id": "chatcmpl-img3",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "raw"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	msgs := []EyrieMessage{
		{Role: "user", Images: []string{"AAAA"}},
	}
	resp, err := c.Chat(context.Background(), msgs, defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "raw" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

// --- TestOpenAI_ToolResultMessages ---

func TestOpenAIChat_ToolResultMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		msgs := reqBody["messages"].([]interface{})
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}

		// First: user message
		first := msgs[0].(map[string]interface{})
		if first["role"] != "user" {
			t.Errorf("expected user role, got %v", first["role"])
		}

		// Second: assistant with tool_calls
		second := msgs[1].(map[string]interface{})
		if second["role"] != "assistant" {
			t.Errorf("expected assistant role, got %v", second["role"])
		}
		tcs := second["tool_calls"].([]interface{})
		if len(tcs) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(tcs))
		}

		// Third: tool result
		third := msgs[2].(map[string]interface{})
		if third["role"] != "tool" {
			t.Errorf("expected tool role, got %v", third["role"])
		}
		if third["tool_call_id"] != "call_xyz" {
			t.Errorf("expected tool_call_id=call_xyz, got %v", third["tool_call_id"])
		}
		if third["content"] != "file contents here" {
			t.Errorf("unexpected tool result content: %v", third["content"])
		}

		w.Header().Set("X-Request-Id", "req-tr")
		resp := map[string]interface{}{
			"id": "chatcmpl-tr",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "Got it"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	msgs := []EyrieMessage{
		{Role: "user", Content: "Read main.go"},
		{Role: "assistant", ToolUse: []ToolCall{{ID: "call_xyz", Name: "read_file", Arguments: map[string]interface{}{"path": "main.go"}}}},
		{Role: "user", ToolResult: &ToolResult{ToolUseID: "call_xyz", Content: "file contents here"}},
	}
	resp, err := c.Chat(context.Background(), msgs, defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Got it" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

// --- TestOpenAI_Name ---

func TestOpenAIClient_Name(t *testing.T) {
	c := NewOpenAIClient("key", "http://example.com", nil)
	if c.Name() != "openai" {
		t.Errorf("expected name=openai, got %s", c.Name())
	}
}

// --- TestOpenAI_DefaultBaseURL ---

func TestOpenAIClient_DefaultBaseURL(t *testing.T) {
	c := NewOpenAIClient("key", "", nil)
	if c.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default baseURL, got %s", c.baseURL)
	}
}

// --- TestOpenAI_MaxTokensDefault ---

func TestOpenAIChat_MaxTokensDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		// Default compat is OpenAICompat which uses max_completion_tokens
		mct, ok := reqBody["max_completion_tokens"]
		if !ok {
			t.Fatal("expected max_completion_tokens in request")
		}
		if int(mct.(float64)) != 4096 {
			t.Errorf("expected default max_completion_tokens=4096, got %v", mct)
		}

		w.Header().Set("X-Request-Id", "req-mt")
		resp := map[string]interface{}{
			"id": "chatcmpl-mt",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "ok"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, &OpenAICompat)
	_, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIChat_MaxTokensCustom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		mct, ok := reqBody["max_completion_tokens"]
		if !ok {
			t.Fatal("expected max_completion_tokens")
		}
		if int(mct.(float64)) != 8192 {
			t.Errorf("expected max_completion_tokens=8192, got %v", mct)
		}

		w.Header().Set("X-Request-Id", "req-mt2")
		resp := map[string]interface{}{
			"id": "chatcmpl-mt2",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "ok"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, &OpenAICompat)
	opts := ChatOptions{Model: "gpt-4o", MaxTokens: 8192}
	_, err := c.Chat(context.Background(), msgs(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func msgs() []EyrieMessage {
	return basicMessages()
}

// --- TestOpenAI_EmptyChoices ---

func TestOpenAIChat_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-empty")
		resp := map[string]interface{}{
			"id":      "chatcmpl-empty",
			"choices": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.Chat(context.Background(), basicMessages(), defaultChatOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content, got %q", resp.Content)
	}
	if resp.FinishReason != "unknown" {
		t.Errorf("expected finish_reason=unknown for empty choices, got %s", resp.FinishReason)
	}
}

// --- TestOpenAI_Temperature ---

func TestOpenAIChat_Temperature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		temp, ok := reqBody["temperature"]
		if !ok {
			t.Fatal("expected temperature in request")
		}
		if temp.(float64) != 0.7 {
			t.Errorf("expected temperature=0.7, got %v", temp)
		}

		w.Header().Set("X-Request-Id", "req-temp")
		resp := map[string]interface{}{
			"id": "chatcmpl-temp",
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "ok"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	temp := 0.7
	opts := ChatOptions{Model: "gpt-4o", Temperature: &temp}
	_, err := c.Chat(context.Background(), basicMessages(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
