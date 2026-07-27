package adapters

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/GrayCodeAI/eyrie/client/core"
	"github.com/GrayCodeAI/hawk-core-contracts/llm"
)

func TestStreamResultFromChat(t *testing.T) {
	t.Parallel()
	result := streamResultFromChat(&core.EyrieResponse{
		Content:      "Hi there!",
		FinishReason: "stop",
	})
	var content string
	for event := range result.Events {
		if event.Type == "content" {
			content += event.Content
		}
	}
	if content != "Hi there!" {
		t.Fatalf("content = %q, want Hi there!", content)
	}
}

func TestNewStreamWithReasoningFallbackChatFirst(t *testing.T) {
	t.Parallel()
	primaryEvents := make(chan core.EyrieStreamEvent, 4)
	primaryEvents <- core.EyrieStreamEvent{Type: "thinking", Thinking: "internal reasoning"}
	primaryEvents <- core.EyrieStreamEvent{Type: "done", StopReason: "end_turn"}
	close(primaryEvents)
	primary := llm.NewStreamResult(primaryEvents, "", func() {})

	var chatCalled, streamCalled bool
	fallback := protocolStreamFallback{
		chat: func(context.Context, []core.EyrieMessage, core.ChatOptions) (*core.EyrieResponse, error) {
			chatCalled = true
			return &core.EyrieResponse{Content: "Hello from chat fallback!", FinishReason: "stop"}, nil
		},
		stream: func(context.Context, []core.EyrieMessage, core.ChatOptions) (*core.StreamResult, error) {
			streamCalled = true
			return nil, fmt.Errorf("stream fallback should not run")
		},
	}

	result := newStreamWithReasoningFallback(context.Background(), nil, core.ChatOptions{}, primary, fallback)
	var content string
	for event := range result.Events {
		switch event.Type {
		case "thinking":
			t.Fatal("primary reasoning must not leak before chat fallback succeeds")
		case "content":
			content += event.Content
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

func TestNewStreamWithReasoningFallbackStreamWhenChatEmpty(t *testing.T) {
	t.Parallel()
	primaryEvents := make(chan core.EyrieStreamEvent, 4)
	primaryEvents <- core.EyrieStreamEvent{Type: "thinking", Thinking: "internal reasoning"}
	primaryEvents <- core.EyrieStreamEvent{Type: "done", StopReason: "end_turn"}
	close(primaryEvents)
	primary := llm.NewStreamResult(primaryEvents, "", func() {})

	fallbackEvents := make(chan core.EyrieStreamEvent, 4)
	fallbackEvents <- core.EyrieStreamEvent{Type: "content", Content: "stream answer"}
	fallbackEvents <- core.EyrieStreamEvent{Type: "done", StopReason: "stop"}
	close(fallbackEvents)

	var chatCalled, streamCalled bool
	fallback := protocolStreamFallback{
		chat: func(context.Context, []core.EyrieMessage, core.ChatOptions) (*core.EyrieResponse, error) {
			chatCalled = true
			return &core.EyrieResponse{Thinking: "still thinking"}, nil
		},
		stream: func(context.Context, []core.EyrieMessage, core.ChatOptions) (*core.StreamResult, error) {
			streamCalled = true
			return llm.NewStreamResult(fallbackEvents, "", func() {}), nil
		},
	}

	result := newStreamWithReasoningFallback(context.Background(), nil, core.ChatOptions{}, primary, fallback)
	var content string
	for event := range result.Events {
		if event.Type == "content" {
			content += event.Content
		}
	}
	if content != "stream answer" {
		t.Fatalf("content = %q, want stream answer", content)
	}
	if !chatCalled || !streamCalled {
		t.Fatalf("chatCalled=%v streamCalled=%v, want both true", chatCalled, streamCalled)
	}
}

func TestProtocolRouterChatFallbackOnError(t *testing.T) {
	t.Parallel()
	openAI := NewOpenAIClient("key", "https://example/openai", nil)
	anthropic := NewAnthropicClient("key", "https://example")
	openAI.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, map[string]string{"error": "down"}), nil
	})}
	anthropic.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant",
			"content":     []map[string]string{{"type": "text", "text": "fallback"}},
			"stop_reason": "end_turn",
		}), nil
	})}

	router := ProtocolRouter{OpenAI: openAI, Anthropic: anthropic}
	response, err := router.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "hi"}}, core.ChatOptions{
		Model: "test", MaxTokens: 16,
	}, ChatProtocolCompletions, func(err error, _ *core.EyrieResponse) bool {
		return err != nil
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if response.Content != "fallback" {
		t.Fatalf("content = %q, want fallback", response.Content)
	}
}

func TestProtocolRouterNoFallbackWhenNil(t *testing.T) {
	t.Parallel()
	openAI := NewOpenAIClient("key", "https://example/openai", nil)
	openAI.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, map[string]string{"error": "down"}), nil
	})}
	router := ProtocolRouter{OpenAI: openAI}
	_, err := router.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "hi"}}, core.ChatOptions{
		Model: "test", MaxTokens: 16,
	}, ChatProtocolCompletions, nil)
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}

func TestStreamResultFromChat_FullResponse(t *testing.T) {
	t.Parallel()
	resp := &core.EyrieResponse{
		Thinking:     "Let me think...",
		Content:      "Hello there!",
		ToolCalls:    []core.ToolCall{{Name: "get_weather", Arguments: map[string]interface{}{"city": "NYC"}}},
		Usage:        &core.EyrieUsage{TotalTokens: 42},
		FinishReason: "",
	}
	result := streamResultFromChat(resp)
	defer result.Close()

	var thinking, content string
	var toolCalls int
	var gotUsage, gotDone bool
	var stopReason string
	for evt := range result.Events {
		switch evt.Type {
		case "thinking":
			thinking = evt.Thinking
		case "content":
			content = evt.Content
		case "tool_call":
			toolCalls++
		case "usage":
			gotUsage = evt.Usage != nil
		case "done":
			gotDone = true
			stopReason = evt.StopReason
		}
	}
	if thinking != "Let me think..." {
		t.Errorf("thinking = %q", thinking)
	}
	if content != "Hello there!" {
		t.Errorf("content = %q", content)
	}
	if toolCalls != 1 {
		t.Errorf("tool calls = %d", toolCalls)
	}
	if !gotUsage {
		t.Error("expected usage event")
	}
	if !gotDone {
		t.Error("expected done event")
	}
	if stopReason != "stop" {
		t.Errorf("stop_reason = %q, expected 'stop'", stopReason)
	}
}

func TestStreamResultFromChat_NilResponse(t *testing.T) {
	t.Parallel()
	result := streamResultFromChat(nil)
	defer result.Close()

	var eventCount int
	for range result.Events {
		eventCount++
	}
	if eventCount != 0 {
		t.Errorf("expected 0 events for nil response, got %d", eventCount)
	}
}

func TestIsReasoningOnlyStreamDiagnostic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{"error_only_reasoning", "error_only_reasoning", true},
		{"reasoning tokens but no answer", "reasoning tokens but no answer", true},
		{"case insensitive", "ERROR_ONLY_REASONING", true},
		{"with whitespace", "  error_only_reasoning  ", true},
		{"normal message", "Hello world", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReasoningOnlyStreamDiagnostic(tt.message)
			if got != tt.want {
				t.Errorf("isReasoningOnlyStreamDiagnostic(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}
