//nolint:errcheck
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDetectProvider(t *testing.T) {
	// Clear all
	for _, k := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY", "GROK_API_KEY", "XAI_API_KEY", "GEMINI_API_KEY", "CANOPYWAVE_API_KEY", "OPENCODEGO_API_KEY", "OLLAMA_BASE_URL"} {
		_ = os.Unsetenv(k)
	}
	// Default should be anthropic
	if p := DetectProvider(); p != "anthropic" {
		t.Errorf("expected anthropic default, got %s", p)
	}
	_ = os.Setenv("OPENAI_API_KEY", "test")
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }()
	if p := DetectProvider(); p != "openai" {
		t.Errorf("expected openai, got %s", p)
	}
}

func TestParseCustomHeaders(t *testing.T) {
	_ = os.Setenv("GRAYCODE_CUSTOM_HEADERS", "X-Custom: value1\nX-Other: value2")
	defer func() { _ = os.Unsetenv("GRAYCODE_CUSTOM_HEADERS") }()
	h := ParseCustomHeaders()
	if h["X-Custom"] != "value1" {
		t.Errorf("expected value1, got %s", h["X-Custom"])
	}
	if h["X-Other"] != "value2" {
		t.Errorf("expected value2, got %s", h["X-Other"])
	}
}

func TestClient(t *testing.T) {
	c := Client(&EyrieConfig{Provider: "openai", APIKey: "test-key"})
	if c.defaultProvider != "openai" {
		t.Errorf("expected openai, got %s", c.defaultProvider)
	}
	providers := c.GetProviders()
	if len(providers) == 0 {
		t.Error("expected providers list")
	}
}

func TestOpenAICompatConfig(t *testing.T) {
	if !OpenAICompat.SupportsStore {
		t.Error("expected OpenAI to support store")
	}
	if GrokCompat.SupportsStore {
		t.Error("expected Grok to not support store")
	}
	if OpenRouterCompat.ThinkingFormat != "openrouter" {
		t.Error("expected openrouter thinking format")
	}
}

func TestAnthropicClientChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Error("missing API key header")
		}
		if r.Header.Get("Anthropic-Version") == "" {
			t.Error("missing Anthropic-Version header")
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent header")
		}
		w.Header().Set("Request-Id", "req-123")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "msg_123", "type": "message", "role": "assistant",
			"content":     []map[string]interface{}{{"type": "text", "text": "Hello!"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer server.Close()

	ac := NewAnthropicClient("test-key", server.URL)
	resp, err := ac.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "Hi"},
	}, ChatOptions{Model: "claude-sonnet-4-6", MaxTokens: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("expected Hello!, got %s", resp.Content)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("expected end_turn, got %s", resp.FinishReason)
	}
	if resp.RequestID != "req-123" {
		t.Errorf("expected request ID req-123, got %s", resp.RequestID)
	}
}

func TestOpenAIClientChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing auth header")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{{
				"message":       map[string]string{"content": "Hi there!", "role": "assistant"},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 8, "completion_tokens": 4, "total_tokens": 12},
		})
	}))
	defer server.Close()

	oc := NewOpenAIClient("test-key", server.URL, &OpenAICompat)
	resp, err := oc.Chat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "Hello"},
	}, ChatOptions{Model: "gpt-4o", MaxTokens: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hi there!" {
		t.Errorf("expected Hi there!, got %s", resp.Content)
	}
	if resp.Usage.TotalTokens != 12 {
		t.Errorf("expected 12 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestAnthropicClientPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "msg_123", "type": "message", "role": "assistant",
			"content": []map[string]interface{}{{"type": "text", "text": "hi"}},
			"usage":   map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	ac := NewAnthropicClient("test-key", server.URL)
	if err := ac.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestOpenAIClientPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	oc := NewOpenAIClient("test-key", server.URL, nil)
	if err := oc.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestRetryConfig(t *testing.T) {
	rc := DefaultRetryConfig()
	if rc.MaxRetries != 3 {
		t.Errorf("expected 3 max retries, got %d", rc.MaxRetries)
	}
	if !rc.shouldRetry(429) {
		t.Error("expected 429 to be retryable")
	}
	if rc.shouldRetry(200) {
		t.Error("expected 200 to not be retryable")
	}
	if !rc.shouldRetry(529) {
		t.Error("expected 529 to be retryable")
	}
}

func TestStreamParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	oc := NewOpenAIClient("test-key", server.URL, &OpenAICompat)
	sr, err := oc.StreamChat(context.Background(), []EyrieMessage{
		{Role: "user", Content: "Hi"},
	}, ChatOptions{Model: "gpt-4o"})
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
	if content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", content)
	}
}

func TestEyrieClientEmptyMessages(t *testing.T) {
	c := Client(&EyrieConfig{Provider: "openai", APIKey: "test"})
	_, err := c.Chat(context.Background(), nil, ChatOptions{Model: "gpt-4o"})
	if err == nil {
		t.Error("expected error for empty messages")
	}
}

func TestProviderInterface(t *testing.T) {
	// Verify both clients implement Provider
	var _ Provider = (*AnthropicClient)(nil)
	var _ Provider = (*OpenAIClient)(nil)
}
