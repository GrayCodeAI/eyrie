package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestStreamResultFromChat(t *testing.T) {
	t.Parallel()
	result := streamResultFromChat(&EyrieResponse{
		Content:      "Hi there!",
		FinishReason: "stop",
	})
	var content string
	for ev := range result.Events {
		if ev.Type == "content" {
			content += ev.Content
		}
	}
	if content != "Hi there!" {
		t.Fatalf("content = %q, want Hi there!", content)
	}
}

func TestNewStreamWithReasoningFallback_ChatFirst(t *testing.T) {
	t.Parallel()
	primaryEvents := make(chan EyrieStreamEvent, 4)
	primaryEvents <- EyrieStreamEvent{Type: "thinking", Thinking: "internal reasoning"}
	primaryEvents <- EyrieStreamEvent{Type: "done", StopReason: "end_turn"}
	close(primaryEvents)
	primary := NewStreamResult(primaryEvents, func() {})

	var chatCalled, streamCalled bool
	fallback := protocolStreamFallback{
		chat: func(context.Context, []EyrieMessage, ChatOptions) (*EyrieResponse, error) {
			chatCalled = true
			return &EyrieResponse{Content: "Hello from chat fallback!", FinishReason: "stop"}, nil
		},
		stream: func(context.Context, []EyrieMessage, ChatOptions) (*StreamResult, error) {
			streamCalled = true
			return nil, fmt.Errorf("stream fallback should not run")
		},
	}

	result := newStreamWithReasoningFallback(context.Background(), nil, ChatOptions{}, primary, fallback)
	var content string
	for ev := range result.Events {
		switch ev.Type {
		case "thinking":
			t.Fatal("primary reasoning must not leak before chat fallback succeeds")
		case "content":
			content += ev.Content
		}
	}
	if content != "Hello from chat fallback!" {
		t.Fatalf("content = %q, want Hello from chat fallback!", content)
	}
	if !chatCalled {
		t.Fatal("expected chat fallback to run")
	}
	if streamCalled {
		t.Fatal("stream fallback must not run when chat fallback succeeds")
	}
}

func TestNewStreamWithReasoningFallback_StreamWhenChatEmpty(t *testing.T) {
	t.Parallel()
	primaryEvents := make(chan EyrieStreamEvent, 4)
	primaryEvents <- EyrieStreamEvent{Type: "thinking", Thinking: "internal reasoning"}
	primaryEvents <- EyrieStreamEvent{Type: "done", StopReason: "end_turn"}
	close(primaryEvents)
	primary := NewStreamResult(primaryEvents, func() {})

	fallbackEvents := make(chan EyrieStreamEvent, 4)
	fallbackEvents <- EyrieStreamEvent{Type: "content", Content: "stream answer"}
	fallbackEvents <- EyrieStreamEvent{Type: "done", StopReason: "stop"}
	close(fallbackEvents)

	var chatCalled, streamCalled bool
	fallback := protocolStreamFallback{
		chat: func(context.Context, []EyrieMessage, ChatOptions) (*EyrieResponse, error) {
			chatCalled = true
			return &EyrieResponse{Thinking: "still thinking"}, nil
		},
		stream: func(context.Context, []EyrieMessage, ChatOptions) (*StreamResult, error) {
			streamCalled = true
			return NewStreamResult(fallbackEvents, func() {}), nil
		},
	}

	result := newStreamWithReasoningFallback(context.Background(), nil, ChatOptions{}, primary, fallback)
	var content string
	for ev := range result.Events {
		if ev.Type == "content" {
			content += ev.Content
		}
	}
	if content != "stream answer" {
		t.Fatalf("content = %q, want stream answer", content)
	}
	if !chatCalled || !streamCalled {
		t.Fatalf("chatCalled=%v streamCalled=%v, want both true", chatCalled, streamCalled)
	}
}

func TestAnthropicBaseFromOpenAIV1(t *testing.T) {
	t.Parallel()
	if got := AnthropicBaseFromOpenAIV1("https://example.com/zen/go/v1"); got != "https://example.com/zen/go" {
		t.Fatalf("got %q", got)
	}
	if got := AnthropicBaseFromOpenAIV1("https://example.com/zen/go"); got != "https://example.com/zen/go" {
		t.Fatalf("got %q", got)
	}
}

func TestProtocolRouter_ChatFallbackOnError(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
