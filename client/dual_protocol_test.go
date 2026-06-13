package client

import (
	"context"
	"fmt"
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

func TestOACompatUnsupportedError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{fmt.Errorf("status=401 unauthorized"), true},
		{fmt.Errorf("oa-compat not supported"), true},
		{fmt.Errorf("HTTP 400 bad request"), false},
	}
	for _, tc := range tests {
		if got := OACompatUnsupportedError(tc.err); got != tc.want {
			t.Errorf("OACompatUnsupportedError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestDualProtocolPair_ChatFallbackOnError(t *testing.T) {
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

	pair := DualProtocolPair{OpenAI: openAI, Anthropic: anthropic}
	resp, err := pair.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "hi"}}, ChatOptions{
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

func TestDualProtocolPair_NoFallbackWhenNil(t *testing.T) {
	openAI := NewOpenAIClient("key", "https://example/openai", nil)
	openAI.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, map[string]string{"error": "down"}), nil
	})}
	pair := DualProtocolPair{OpenAI: openAI}
	_, err := pair.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "hi"}}, ChatOptions{
		Model: "test", MaxTokens: 16,
	}, ChatProtocolCompletions, nil)
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}
