package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- buildAnthropicMessages tests ---

func TestAnthropicBuildMessages_TextOnly(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}
	result, system := buildAnthropicMessages(msgs)
	if system != "You are helpful." {
		t.Errorf("expected system to be extracted, got %q", system)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 messages (system excluded), got %d", len(result))
	}
	if result[0]["role"] != "user" {
		t.Errorf("expected first message role=user, got %v", result[0]["role"])
	}
	if result[0]["content"] != "Hello" {
		t.Errorf("expected content=Hello, got %v", result[0]["content"])
	}
	if result[1]["role"] != "assistant" {
		t.Errorf("expected second message role=assistant, got %v", result[1]["role"])
	}
	if result[2]["content"] != "How are you?" {
		t.Errorf("expected last content, got %v", result[2]["content"])
	}
}

func TestAnthropicBuildMessages_ToolUse(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "assistant", Content: "Let me check.", ToolUse: []ToolCall{
			{ID: "call_1", Name: "get_weather", Arguments: map[string]interface{}{"city": "NYC"}},
		}},
	}
	result, _ := buildAnthropicMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	content, ok := result[0]["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected content to be []map, got %T", result[0]["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks (text + tool_use), got %d", len(content))
	}
	if content[0]["type"] != "text" || content[0]["text"] != "Let me check." {
		t.Errorf("expected text block, got %v", content[0])
	}
	if content[1]["type"] != "tool_use" {
		t.Errorf("expected tool_use block, got %v", content[1])
	}
	if content[1]["id"] != "call_1" {
		t.Errorf("expected id=call_1, got %v", content[1]["id"])
	}
	if content[1]["name"] != "get_weather" {
		t.Errorf("expected name=get_weather, got %v", content[1]["name"])
	}
}

func TestAnthropicBuildMessages_ToolUseNoText(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "assistant", ToolUse: []ToolCall{
			{ID: "call_2", Name: "read_file", Arguments: map[string]interface{}{"path": "/tmp/x"}},
		}},
	}
	result, _ := buildAnthropicMessages(msgs)
	content := result[0]["content"].([]map[string]interface{})
	// No text block, only tool_use
	if len(content) != 1 {
		t.Fatalf("expected 1 content block (tool_use only), got %d", len(content))
	}
	if content[0]["type"] != "tool_use" {
		t.Errorf("expected tool_use, got %v", content[0]["type"])
	}
}

func TestAnthropicBuildMessages_ToolResult(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", ToolResults: []ToolResult{{
			ToolUseID: "call_1",
			Content:   "Temperature: 72F",
			IsError:   false,
		}}},
	}
	result, _ := buildAnthropicMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	content, ok := result[0]["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected content to be []map, got %T", result[0]["content"])
	}
	if content[0]["type"] != "tool_result" {
		t.Errorf("expected tool_result type, got %v", content[0]["type"])
	}
	if content[0]["tool_use_id"] != "call_1" {
		t.Errorf("expected tool_use_id=call_1, got %v", content[0]["tool_use_id"])
	}
	if content[0]["content"] != "Temperature: 72F" {
		t.Errorf("expected tool content, got %v", content[0]["content"])
	}
	// is_error should NOT be present for non-error results
	if _, exists := content[0]["is_error"]; exists {
		t.Errorf("is_error should not be set for non-error result")
	}
}

func TestAnthropicBuildMessages_ToolResultError(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", ToolResults: []ToolResult{{
			ToolUseID: "call_err",
			Content:   "connection refused",
			IsError:   true,
		}}},
	}
	result, _ := buildAnthropicMessages(msgs)
	content := result[0]["content"].([]map[string]interface{})
	if content[0]["is_error"] != true {
		t.Errorf("expected is_error=true, got %v", content[0]["is_error"])
	}
}

func TestAnthropicBuildMessages_ImageBase64(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "What is this?", Images: []string{
			"data:image/png;base64,iVBORw0KGgoAAAANS",
		}},
	}
	result, _ := buildAnthropicMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	content, ok := result[0]["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected multi-part content, got %T", result[0]["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 blocks (text + image), got %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Errorf("expected text block first, got %v", content[0]["type"])
	}
	if content[1]["type"] != "image" {
		t.Errorf("expected image block, got %v", content[1]["type"])
	}
	source, ok := content[1]["source"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected source map, got %T", content[1]["source"])
	}
	if source["type"] != "base64" {
		t.Errorf("expected base64 source type, got %v", source["type"])
	}
	if source["media_type"] != "image/png" {
		t.Errorf("expected media_type=image/png, got %v", source["media_type"])
	}
	if source["data"] != "iVBORw0KGgoAAAANS" {
		t.Errorf("expected base64 data, got %v", source["data"])
	}
}

func TestAnthropicBuildMessages_ImageURL(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "Describe this", Images: []string{
			"https://example.com/image.jpg",
		}},
	}
	result, _ := buildAnthropicMessages(msgs)
	content := result[0]["content"].([]map[string]interface{})
	if len(content) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(content))
	}
	source := content[1]["source"].(map[string]interface{})
	if source["type"] != "url" {
		t.Errorf("expected url source type, got %v", source["type"])
	}
	if source["url"] != "https://example.com/image.jpg" {
		t.Errorf("expected URL, got %v", source["url"])
	}
}

func TestAnthropicBuildMessages_ImageNoText(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Images: []string{"https://example.com/pic.png"}},
	}
	result, _ := buildAnthropicMessages(msgs)
	content := result[0]["content"].([]map[string]interface{})
	// Only 1 block (image), no text block since Content is empty
	if len(content) != 1 {
		t.Fatalf("expected 1 block (image only), got %d", len(content))
	}
	if content[0]["type"] != "image" {
		t.Errorf("expected image type, got %v", content[0]["type"])
	}
}

func TestAnthropicBuildMessages_MultipleImages(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "Compare these", Images: []string{
			"data:image/jpeg;base64,/9j/4AAQ",
			"https://example.com/other.png",
		}},
	}
	result, _ := buildAnthropicMessages(msgs)
	content := result[0]["content"].([]map[string]interface{})
	// text + 2 images = 3 blocks
	if len(content) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(content))
	}
	// First image is base64
	src1 := content[1]["source"].(map[string]interface{})
	if src1["type"] != "base64" {
		t.Errorf("first image should be base64, got %v", src1["type"])
	}
	if src1["media_type"] != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %v", src1["media_type"])
	}
	// Second image is URL
	src2 := content[2]["source"].(map[string]interface{})
	if src2["type"] != "url" {
		t.Errorf("second image should be url, got %v", src2["type"])
	}
}

func TestAnthropicBuildMessages_NoSystem(t *testing.T) {
	msgs := []EyrieMessage{
		{Role: "user", Content: "Hello"},
	}
	result, system := buildAnthropicMessages(msgs)
	if system != "" {
		t.Errorf("expected empty system, got %q", system)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
}

func TestAnthropicBuildMessages_EmptyInput(t *testing.T) {
	result, system := buildAnthropicMessages(nil)
	if system != "" {
		t.Errorf("expected empty system, got %q", system)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

// --- convertToAnthropicTools tests ---

func TestAnthropicConvertTools(t *testing.T) {
	tools := []EyrieTool{
		{
			Name:        "get_weather",
			Description: "Get weather for a city",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type":        "string",
						"description": "City name",
					},
				},
				"required": []interface{}{"city"},
			},
		},
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	result := convertToAnthropicTools(tools)
	if len(result) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result))
	}
	if result[0].Name != "get_weather" {
		t.Errorf("expected get_weather, got %s", result[0].Name)
	}
	if result[0].Description != "Get weather for a city" {
		t.Errorf("expected description, got %s", result[0].Description)
	}
	if result[0].InputSchema["type"] != "object" {
		t.Errorf("expected type=object in input_schema, got %v", result[0].InputSchema["type"])
	}
	if result[1].Name != "read_file" {
		t.Errorf("expected read_file, got %s", result[1].Name)
	}
}

func TestAnthropicConvertTools_Empty(t *testing.T) {
	result := convertToAnthropicTools(nil)
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}
	result = convertToAnthropicTools([]EyrieTool{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

// --- AnthropicClient.Chat() tests ---

func TestAnthropicChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "sk-test-123" {
			t.Errorf("expected API key header, got %q", r.Header.Get("X-Api-Key"))
		}
		if r.Header.Get("Anthropic-Version") != "2023-06-01" {
			t.Errorf("expected version header, got %q", r.Header.Get("Anthropic-Version"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected json content-type, got %q", r.Header.Get("Content-Type"))
		}

		// Decode request body
		var req anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Model != "claude-sonnet-4-6" {
			t.Errorf("expected model claude-sonnet-4-6, got %s", req.Model)
		}
		if req.MaxTokens != 1024 {
			t.Errorf("expected max_tokens 1024, got %d", req.MaxTokens)
		}
		if req.System != "Be helpful" {
			t.Errorf("expected system prompt, got %q", req.System)
		}

		w.Header().Set("Request-Id", "req-abc-123")
		_, _ = w.Write([]byte(`{"id":"msg_001","content":[{"type":"text","text":"Hello! How can I help?"}],"stop_reason":"end_turn","usage":{"input_tokens":25,"output_tokens":12}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient(
		"sk-test-123", server.URL,
		WithRetry(NewRetryConfig(0, 0, 0)),
	)

	resp, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "Hi there"},
	}, ChatOptions{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 1024,
		System:    "Be helpful",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello! How can I help?" {
		t.Errorf("expected response text, got %q", resp.Content)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("expected end_turn, got %s", resp.FinishReason)
	}
	if resp.RequestID != "req-abc-123" {
		t.Errorf("expected request ID, got %s", resp.RequestID)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if resp.Usage.PromptTokens != 25 {
		t.Errorf("expected 25 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 12 {
		t.Errorf("expected 12 completion tokens, got %d", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 37 {
		t.Errorf("expected 37 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestAnthropicChat_WithToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-tool-1")
		// Return a response with tool_use blocks
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "msg_002",
			"type": "message",
			"content": []map[string]interface{}{
				{"type": "text", "text": "I'll check the weather."},
				{
					"type":  "tool_use",
					"id":    "toolu_01",
					"name":  "get_weather",
					"input": map[string]interface{}{"city": "San Francisco"},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]int{"input_tokens": 50, "output_tokens": 30},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient(
		"sk-test", server.URL,
		WithRetry(NewRetryConfig(0, 0, 0)),
	)
	resp, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "What is the weather in SF?"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "I'll check the weather." {
		t.Errorf("expected text content, got %q", resp.Content)
	}
	if resp.FinishReason != "tool_use" {
		t.Errorf("expected tool_use finish reason, got %s", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_01" {
		t.Errorf("expected tool call ID toolu_01, got %s", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("expected tool name get_weather, got %s", tc.Name)
	}
	if tc.Arguments["city"] != "San Francisco" {
		t.Errorf("expected city=San Francisco, got %v", tc.Arguments["city"])
	}
}

func TestAnthropicChat_MultipleToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-multi")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "msg_003",
			"type": "message",
			"content": []map[string]interface{}{
				{
					"type":  "tool_use",
					"id":    "toolu_a",
					"name":  "read_file",
					"input": map[string]interface{}{"path": "/etc/hosts"},
				},
				{
					"type":  "tool_use",
					"id":    "toolu_b",
					"name":  "list_dir",
					"input": map[string]interface{}{"path": "/tmp"},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]int{"input_tokens": 20, "output_tokens": 40},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	resp, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "read /etc/hosts and list /tmp"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected read_file, got %s", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[1].Name != "list_dir" {
		t.Errorf("expected list_dir, got %s", resp.ToolCalls[1].Name)
	}
}

func TestAnthropicChat_DefaultMaxTokens(t *testing.T) {
	var capturedBody anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Request-Id", "req-default")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_004",
			"content":     []map[string]interface{}{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 2},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"}) // MaxTokens not set
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens=4096, got %d", capturedBody.MaxTokens)
	}
}

func TestAnthropicChat_ModelRequired(t *testing.T) {
	client := NewAnthropicClient("key", "http://localhost")
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{}) // No model set
	if err == nil {
		t.Fatal("expected error when model is empty")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Errorf("expected model required error, got: %v", err)
	}
}

func TestAnthropicChat_SystemMerge(t *testing.T) {
	var capturedBody anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Request-Id", "req-sys")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_005",
			"content":     []map[string]interface{}{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 2},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "system", Content: "From messages"},
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6", System: "From opts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// System from opts + system from message should be merged
	if !strings.Contains(capturedBody.System, "From opts") {
		t.Errorf("expected opts system in merged system, got %q", capturedBody.System)
	}
	if !strings.Contains(capturedBody.System, "From messages") {
		t.Errorf("expected message system in merged system, got %q", capturedBody.System)
	}
}

func TestAnthropicChat_WithTools(t *testing.T) {
	var capturedBody anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Request-Id", "req-tools")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_006",
			"content":     []map[string]interface{}{{"type": "text", "text": "done"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 30, "output_tokens": 5},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{
		Model: "claude-sonnet-4-6",
		Tools: []EyrieTool{
			{Name: "calculator", Description: "Math", Parameters: map[string]interface{}{"type": "object"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedBody.Tools) != 1 {
		t.Fatalf("expected 1 tool in request, got %d", len(capturedBody.Tools))
	}
	if capturedBody.Tools[0].Name != "calculator" {
		t.Errorf("expected tool name calculator, got %s", capturedBody.Tools[0].Name)
	}
}

func TestAnthropicChat_CacheUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-cache")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_007",
			"content":     []map[string]interface{}{{"type": "text", "text": "cached!"}},
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":                10,
				"output_tokens":               5,
				"cache_creation_input_tokens": 100,
				"cache_read_input_tokens":     50,
			},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	resp, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Usage.CacheCreationTokens != 100 {
		t.Errorf("expected 100 cache creation tokens, got %d", resp.Usage.CacheCreationTokens)
	}
	if resp.Usage.CacheReadTokens != 50 {
		t.Errorf("expected 50 cache read tokens, got %d", resp.Usage.CacheReadTokens)
	}
}

// --- StreamChat tests ---

func TestAnthropicStreamChat_TextContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected Accept: text/event-stream, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Request-Id", "req-stream-1")
		flusher, _ := w.(http.Flusher)

		// message_start with usage
		_, _ = fmt.Fprintf(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":15,\"output_tokens\":0}}}\n\n")
		flusher.Flush()

		// content_block_start
		_, _ = fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		flusher.Flush()

		// text deltas
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n")
		flusher.Flush()

		// content_block_stop
		_, _ = fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
		flusher.Flush()

		// message_delta with stop_reason
		_, _ = fmt.Fprintf(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":8}}\n\n")
		flusher.Flush()

		// message_stop
		_, _ = fmt.Fprintf(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient("sk-test", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	sr, err := client.StreamChat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "Hello"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	if sr.RequestID != "req-stream-1" {
		t.Errorf("expected request ID req-stream-1, got %s", sr.RequestID)
	}

	var content string
	var gotDone bool
	var gotUsage bool
	var stopReason string
	for evt := range sr.Events {
		switch evt.Type {
		case "content":
			content += evt.Content
		case "done":
			gotDone = true
			stopReason = evt.StopReason
		case "usage":
			gotUsage = true
		}
	}
	if content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", content)
	}
	if !gotDone {
		t.Error("expected done event")
	}
	if stopReason != "end_turn" {
		t.Errorf("expected end_turn stop reason, got %q", stopReason)
	}
	if !gotUsage {
		t.Error("expected usage event")
	}
}

func TestAnthropicStreamChat_ToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Request-Id", "req-stream-tool")
		flusher, _ := w.(http.Flusher)

		// message_start
		_, _ = fmt.Fprintf(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":20,\"output_tokens\":0}}}\n\n")
		flusher.Flush()

		// Text block
		_, _ = fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Let me check.\"}}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
		flusher.Flush()

		// Tool use block
		_, _ = fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_stream1\",\"name\":\"get_weather\"}}\n\n")
		flusher.Flush()

		// Tool input deltas (streamed JSON)
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"city\\\"\"}}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\": \\\"NYC\\\"}\"}}\n\n")
		flusher.Flush()

		// Tool block stop
		_, _ = fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":1}\n\n")
		flusher.Flush()

		// message_delta
		_, _ = fmt.Fprintf(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":25}}\n\n")
		flusher.Flush()

		// message_stop
		_, _ = fmt.Fprintf(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient("sk-test", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	sr, err := client.StreamChat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "Weather in NYC?"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	var content string
	var toolCalls []ToolCall
	var stopReason string
	for evt := range sr.Events {
		switch evt.Type {
		case "content":
			content += evt.Content
		case "tool_call":
			if evt.ToolCall != nil {
				toolCalls = append(toolCalls, *evt.ToolCall)
			}
		case "done":
			stopReason = evt.StopReason
		}
	}
	if content != "Let me check." {
		t.Errorf("expected 'Let me check.', got %q", content)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0].ID != "toolu_stream1" {
		t.Errorf("expected tool ID toolu_stream1, got %s", toolCalls[0].ID)
	}
	if toolCalls[0].Name != "get_weather" {
		t.Errorf("expected tool name get_weather, got %s", toolCalls[0].Name)
	}
	if toolCalls[0].Arguments["city"] != "NYC" {
		t.Errorf("expected city=NYC, got %v", toolCalls[0].Arguments["city"])
	}
	if stopReason != "tool_use" {
		t.Errorf("expected tool_use stop reason, got %q", stopReason)
	}
}

func TestAnthropicStreamChat_ModelRequired(t *testing.T) {
	client := NewAnthropicClient("key", "http://localhost")
	_, err := client.StreamChat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{})
	if err == nil {
		t.Fatal("expected error when model is empty")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Errorf("expected model required error, got: %v", err)
	}
}

func TestAnthropicStreamChat_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Request-Id", "req-cancel")
		flusher, _ := w.(http.Flusher)

		// Send a few events then hang
		_, _ = fmt.Fprintf(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":0}}}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"partial\"}}\n\n")
		flusher.Flush()

		// Hang to simulate slow response
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	sr, err := client.StreamChat(ctx, []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer sr.Close()

	var content string
	for evt := range sr.Events {
		if evt.Type == "content" {
			content += evt.Content
		}
	}
	// Should have received partial content before cancellation
	if content != "partial" {
		t.Errorf("expected 'partial', got %q", content)
	}
}

// --- Ping tests ---

func TestAnthropicPing_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected /v1/models for ping, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "valid-key" {
			t.Errorf("expected valid-key, got %q", r.Header.Get("X-Api-Key"))
		}
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "msg_ping", "content": []map[string]interface{}{{"type": "text", "text": "hi"}},
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("valid-key", server.URL)
	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestAnthropicPing_InvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"type": "authentication_error", "message": "invalid x-api-key"},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("bad-key", server.URL)
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid API key") {
		t.Errorf("expected 'invalid API key' error, got: %v", err)
	}
}

func TestAnthropicPing_NonAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 500 is not treated as auth error by Ping
		w.WriteHeader(500)
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL)
	err := client.Ping(context.Background())
	// Non-401 errors should pass without error in current implementation
	if err != nil {
		t.Fatalf("expected no error for 500 (non-auth), got: %v", err)
	}
}

// --- Error handling tests ---

func TestAnthropicChat_Error401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-401")
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "invalid x-api-key",
			},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("bad-key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "authentication_error") {
		t.Errorf("expected authentication_error in message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "req-401") {
		t.Errorf("expected request ID in error, got: %v", err)
	}
}

func TestAnthropicChat_Error429_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(429)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"type": "rate_limit_error", "message": "Too many requests"},
			})
			return
		}
		w.Header().Set("Request-Id", "req-retry-ok")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_retry",
			"content":     []map[string]interface{}{{"type": "text", "text": "finally!"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 3},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient(
		"key", server.URL,
		WithRetry(NewRetryConfig(3, 1*time.Millisecond, 10*time.Millisecond, 429, 500, 502, 503)),
	)
	resp, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if resp.Content != "finally!" {
		t.Errorf("expected 'finally!', got %q", resp.Content)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestAnthropicChat_Error500_ExhaustedRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"type": "server_error", "message": "Internal error"},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient(
		"key", server.URL,
		WithRetry(NewRetryConfig(2, 1*time.Millisecond, 5*time.Millisecond, 500)),
	)
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if !strings.Contains(err.Error(), "max retries") {
		t.Errorf("expected 'max retries' in error, got: %v", err)
	}
	// 1 initial + 2 retries = 3 attempts
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestAnthropicChat_ErrorInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-bad-json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "failed to decode") {
		t.Errorf("expected decode error, got: %v", err)
	}
}

func TestAnthropicStreamChat_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-stream-err")
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "messages: roles must alternate",
			},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.StreamChat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "roles must alternate") {
		t.Errorf("expected roles must alternate error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "req-stream-err") {
		t.Errorf("expected request ID in error, got: %v", err)
	}
}

// --- Client configuration tests ---

func TestAnthropicClient_Name(t *testing.T) {
	client := NewAnthropicClient("key", "")
	if client.Name() != "anthropic" {
		t.Errorf("expected 'anthropic', got %q", client.Name())
	}
}

func TestAnthropicClient_DefaultBaseURL(t *testing.T) {
	client := NewAnthropicClient("key", "")
	if client.baseURL != "https://api.anthropic.com" {
		t.Errorf("expected default base URL, got %q", client.baseURL)
	}
}

func TestAnthropicClient_CustomBaseURL(t *testing.T) {
	client := NewAnthropicClient("key", "https://custom.proxy.com")
	if client.baseURL != "https://custom.proxy.com" {
		t.Errorf("expected custom base URL, got %q", client.baseURL)
	}
}

func TestAnthropicClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{Timeout: 30 * time.Second}
	retryConfig := NewRetryConfig(5, 2*time.Second, 60*time.Second, 429)

	client := NewAnthropicClient(
		"key", "",
		WithHTTPClient(customHTTP),
		WithRetry(retryConfig),
	)
	if client.httpClient != customHTTP {
		t.Error("expected custom HTTP client to be set")
	}
	if client.retry.MaxRetries != 5 {
		t.Errorf("expected 5 max retries, got %d", client.retry.MaxRetries)
	}
}

// --- parseImageString tests ---

func TestAnthropicParseImageString_Base64(t *testing.T) {
	tests := []struct {
		input     string
		mediaType string
		data      string
		isBase64  bool
	}{
		{
			input:     "data:image/png;base64,iVBORw0KGgo=",
			mediaType: "image/png",
			data:      "iVBORw0KGgo=",
			isBase64:  true,
		},
		{
			input:     "data:image/jpeg;base64,/9j/4AAQSkZJRg==",
			mediaType: "image/jpeg",
			data:      "/9j/4AAQSkZJRg==",
			isBase64:  true,
		},
		{
			input:     "data:image/gif;base64,R0lGODlh",
			mediaType: "image/gif",
			data:      "R0lGODlh",
			isBase64:  true,
		},
		{
			input:     "data:image/webp;base64,UklGRl4=",
			mediaType: "image/webp",
			data:      "UklGRl4=",
			isBase64:  true,
		},
	}
	for _, tt := range tests {
		mediaType, data, isBase64 := parseImageString(tt.input)
		if mediaType != tt.mediaType {
			t.Errorf("parseImageString(%q): mediaType=%q, want %q", tt.input, mediaType, tt.mediaType)
		}
		if data != tt.data {
			t.Errorf("parseImageString(%q): data=%q, want %q", tt.input, data, tt.data)
		}
		if isBase64 != tt.isBase64 {
			t.Errorf("parseImageString(%q): isBase64=%v, want %v", tt.input, isBase64, tt.isBase64)
		}
	}
}

func TestAnthropicParseImageString_URL(t *testing.T) {
	tests := []string{
		"https://example.com/image.png",
		"http://localhost:8080/pic.jpg",
		"https://cdn.example.com/path/to/image.webp?w=800",
	}
	for _, url := range tests {
		mediaType, data, isBase64 := parseImageString(url)
		if mediaType != "" {
			t.Errorf("parseImageString(%q): expected empty mediaType, got %q", url, mediaType)
		}
		if data != url {
			t.Errorf("parseImageString(%q): expected data=url, got %q", url, data)
		}
		if isBase64 {
			t.Errorf("parseImageString(%q): expected isBase64=false", url)
		}
	}
}

func TestAnthropicParseImageString_DataURIWithoutBase64(t *testing.T) {
	// data: URI without ;base64, marker should be treated as URL
	input := "data:text/plain,Hello"
	_, data, isBase64 := parseImageString(input)
	if isBase64 {
		t.Error("expected isBase64=false for non-base64 data URI")
	}
	if data != input {
		t.Errorf("expected data to equal input, got %q", data)
	}
}

// --- Temperature tests ---

func TestAnthropicChat_WithTemperature(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Request-Id", "req-temp")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_temp",
			"content":     []map[string]interface{}{{"type": "text", "text": "warm"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 2},
		})
	}))
	defer server.Close()

	temp := 0.7
	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6", Temperature: &temp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody["temperature"] != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", capturedBody["temperature"])
	}
}

// --- Request body verification tests ---

func TestAnthropicChat_RequestBodyStructure(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Request-Id", "req-body")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "msg_body",
			"content":     []map[string]interface{}{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 2},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "test message"},
	}, ChatOptions{Model: "claude-sonnet-4-6", MaxTokens: 2048})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody["model"] != "claude-sonnet-4-6" {
		t.Errorf("expected model in body, got %v", capturedBody["model"])
	}
	if int(capturedBody["max_tokens"].(float64)) != 2048 {
		t.Errorf("expected max_tokens=2048, got %v", capturedBody["max_tokens"])
	}
	msgs, ok := capturedBody["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected 1 message in body, got %v", capturedBody["messages"])
	}
}

// --- Conversation with tool round-trip ---

func TestAnthropicChat_FullToolRoundTrip(t *testing.T) {
	callNum := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNum++
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.Header().Set("Request-Id", fmt.Sprintf("req-rt-%d", callNum))

		if callNum == 1 {
			// First call: model wants to use a tool
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "msg_rt1",
				"content": []map[string]interface{}{
					{"type": "tool_use", "id": "toolu_rt", "name": "get_time", "input": map[string]interface{}{}},
				},
				"stop_reason": "tool_use",
				"usage":       map[string]int{"input_tokens": 20, "output_tokens": 15},
			})
		} else {
			// Second call: with tool result, model provides final answer
			// Verify the messages include tool result
			msgs := reqBody["messages"].([]interface{})
			if len(msgs) < 3 {
				t.Errorf("expected at least 3 messages in second call, got %d", len(msgs))
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "msg_rt2",
				"content":     []map[string]interface{}{{"type": "text", "text": "It is 3pm."}},
				"stop_reason": "end_turn",
				"usage":       map[string]int{"input_tokens": 40, "output_tokens": 8},
			})
		}
	}))
	defer server.Close()

	client := NewAnthropicClient("key", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))

	// First call
	resp1, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "What time is it?"},
	}, ChatOptions{
		Model: "claude-sonnet-4-6",
		Tools: []EyrieTool{{Name: "get_time", Description: "Get current time", Parameters: map[string]interface{}{"type": "object"}}},
	})
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if len(resp1.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp1.ToolCalls))
	}

	// Second call with tool result
	resp2, err := client.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "What time is it?"},
		{Role: "assistant", ToolUse: resp1.ToolCalls},
		{Role: "user", ToolResults: []ToolResult{{ToolUseID: "toolu_rt", Content: "15:00 UTC"}}},
	}, ChatOptions{
		Model: "claude-sonnet-4-6",
		Tools: []EyrieTool{{Name: "get_time", Description: "Get current time", Parameters: map[string]interface{}{"type": "object"}}},
	})
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if resp2.Content != "It is 3pm." {
		t.Errorf("expected final answer, got %q", resp2.Content)
	}
	if resp2.FinishReason != "end_turn" {
		t.Errorf("expected end_turn, got %s", resp2.FinishReason)
	}
}

// =============================================================================
// New feature tests
// =============================================================================

func TestResolveThinking_Modes(t *testing.T) {
	tests := []struct {
		name     string
		opts     ChatOptions
		wantType string
		wantNil  bool
	}{
		{"adaptive", ChatOptions{ThinkingMode: "adaptive"}, "adaptive", false},
		{"disabled", ChatOptions{ThinkingMode: "disabled"}, "disabled", false},
		{"enabled with budget", ChatOptions{ThinkingMode: "enabled", ThinkingBudgetTokens: 10000}, "enabled", false},
		{"enabled zero budget", ChatOptions{ThinkingMode: "enabled"}, "", true},
		{"legacy budget", ChatOptions{ThinkingBudgetTokens: 5000}, "enabled", false},
		{"legacy zero", ChatOptions{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveThinking(tt.opts)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil")
			}
			if got.Type != tt.wantType {
				t.Errorf("type = %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

func TestResolveThinking_Display(t *testing.T) {
	got := resolveThinking(ChatOptions{ThinkingMode: "enabled", ThinkingBudgetTokens: 5000, ThinkingDisplay: "omitted"})
	if got == nil || got.Display != "omitted" {
		t.Fatalf("expected display=omitted, got %+v", got)
	}
}

func TestResolveToolChoice(t *testing.T) {
	if resolveToolChoice(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
	tc := resolveToolChoice(&ToolChoiceOption{Type: "tool", Name: "search", DisableParallelToolUse: true})
	if tc.Type != "tool" || tc.Name != "search" || !tc.DisableParallelToolUse {
		t.Fatalf("unexpected: %+v", tc)
	}
}

func TestResolveOutputConfig(t *testing.T) {
	if resolveOutputConfig(ChatOptions{}) != nil {
		t.Fatal("expected nil for empty opts")
	}
	cfg := resolveOutputConfig(ChatOptions{OutputEffort: "high"})
	if cfg.Effort != "high" || cfg.Format != nil {
		t.Fatalf("unexpected: %+v", cfg)
	}
	cfg2 := resolveOutputConfig(ChatOptions{OutputSchema: `{"type":"object","properties":{"x":{"type":"string"}}}`})
	if cfg2.Format == nil || cfg2.Format.Type != "json_schema" {
		t.Fatalf("unexpected: %+v", cfg2)
	}
}

func TestAnthropicRequest_NewFields(t *testing.T) {
	req := anthropicRequest{
		Model:         "claude-sonnet-4-6",
		MaxTokens:     4096,
		TopP:          float64Ptr(0.9),
		TopK:          intPtr(50),
		StopSequences: []string{"STOP"},
		ToolChoice:    &anthropicToolChoice{Type: "any"},
		Thinking:      &anthropicThinking{Type: "adaptive"},
		Metadata:      &anthropicMetadata{UserID: "user-123"},
		ServiceTier:   "standard_only",
		OutputConfig:  &anthropicOutputConfig{Effort: "high"},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{`"top_p":0.9`, `"top_k":50`, `"stop_sequences":["STOP"]`, `"tool_choice":{"type":"any"}`, `"thinking":{"type":"adaptive"}`, `"metadata":{"user_id":"user-123"}`, `"service_tier":"standard_only"`, `"output_config":{"effort":"high"}`} {
		if !contains(s, want) {
			t.Errorf("missing %q in JSON: %s", want, s)
		}
	}
}

func TestAnthropicChat_ThinkingBlocksInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req-think-1")
		_, _ = w.Write([]byte(`{"id":"msg_think","content":[{"type":"thinking","thinking":"Let me reason..."},{"type":"text","text":"The answer is 42."}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":20,"output_tokens_details":{"thinking_tokens":10}}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient("sk-test", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	resp, err := client.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "What is the answer?"}}, ChatOptions{
		Model:              "claude-sonnet-4-6",
		ThinkingMode:       "enabled",
		ThinkingBudgetTokens: 5000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "The answer is 42." {
		t.Errorf("content = %q", resp.Content)
	}
	if resp.Thinking != "Let me reason..." {
		t.Errorf("thinking = %q", resp.Thinking)
	}
	if resp.Usage.ThinkingTokens != 10 {
		t.Errorf("thinking_tokens = %d", resp.Usage.ThinkingTokens)
	}
}

func TestAnthropicChat_RedactedThinkingSkipped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"msg_rt","content":[{"type":"redacted_thinking","data":"encrypted"},{"type":"text","text":"Done."}],"stop_reason":"end_turn","usage":{"input_tokens":5,"output_tokens":3}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient("sk-test", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	resp, err := client.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "Done." {
		t.Errorf("content = %q", resp.Content)
	}
	if resp.Thinking != "" {
		t.Errorf("thinking should be empty for redacted, got %q", resp.Thinking)
	}
}

func TestAnthropicRequest_WithToolChoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		tc, ok := body["tool_choice"].(map[string]interface{})
		if !ok {
			t.Errorf("expected tool_choice in request, got %v", body["tool_choice"])
			w.WriteHeader(400)
			return
		}
		if tc["type"] != "tool" || tc["name"] != "search" {
			t.Errorf("unexpected tool_choice: %v", tc)
		}
		_, _ = w.Write([]byte(`{"id":"msg","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient("sk-test", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "search"}}, ChatOptions{
		Model:      "claude-sonnet-4-6",
		ToolChoice: &ToolChoiceOption{Type: "tool", Name: "search"},
		Tools:      []EyrieTool{{Name: "search", Description: "Search", Parameters: map[string]interface{}{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAnthropicRequest_WithTopPAndStopSequences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["top_p"] != 0.8 {
			t.Errorf("top_p = %v", body["top_p"])
		}
		stops, ok := body["stop_sequences"].([]interface{})
		if !ok || len(stops) != 1 || stops[0] != "END" {
			t.Errorf("stop_sequences = %v", body["stop_sequences"])
		}
		_, _ = w.Write([]byte(`{"id":"msg","content":[{"type":"text","text":"ok"}],"stop_reason":"stop_sequence","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client := NewAnthropicClient("sk-test", server.URL, WithRetry(NewRetryConfig(0, 0, 0)))
	_, err := client.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Go"}}, ChatOptions{
		Model:         "claude-sonnet-4-6",
		TopP:          float64Ptr(0.8),
		StopSequences: []string{"END"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Helpers
func float64Ptr(f float64) *float64 { return &f }
func intPtr(i int) *int             { return &i }
func contains(s, sub string) bool   { return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub)) }
func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
