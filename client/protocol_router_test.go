package client

import (
	"context"
	"net/http"
	"testing"
)

func TestAnthropicBaseFromOpenAIV1(t *testing.T) {
	if got := AnthropicBaseFromOpenAIV1("https://example.com/zen/go/v1"); got != "https://example.com/zen/go" {
		t.Fatalf("got %q", got)
	}
	if got := AnthropicBaseFromOpenAIV1("https://example.com/zen/go"); got != "https://example.com/zen/go" {
		t.Fatalf("got %q", got)
	}
}

func TestProtocolRouter_ChatFallbackOnError(t *testing.T) {
	openAI := NewOpenAIClient("key", "https://example/openai", nil)
	anthropic := NewAnthropicClient("key", "https://example")
	openAI.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, map[string]string{"error": "down"}), nil
	})}
	anthropic.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "msg_1", "type": "message", "role": "assistant",
			"content":     []map[string]string{{"type": "text", "text": "fallback"}},
			"stop_reason": "end_turn",
		}), nil
	})}

	router := ProtocolRouter{OpenAI: openAI, Anthropic: anthropic}
	resp, err := router.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "hi"}}, ChatOptions{
		Model: "test", MaxTokens: 16,
	}, ChatProtocolCompletions, func(err error, _ *EyrieResponse) bool {
		return err != nil
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "fallback" {
		t.Fatalf("content = %q, want fallback", resp.Content)
	}
}

func TestProtocolRouter_NoFallbackWhenNil(t *testing.T) {
	openAI := NewOpenAIClient("key", "https://example/openai", nil)
	openAI.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, map[string]string{"error": "down"}), nil
	})}
	router := ProtocolRouter{OpenAI: openAI}
	_, err := router.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "hi"}}, ChatOptions{
		Model: "test", MaxTokens: 16,
	}, ChatProtocolCompletions, nil)
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}
