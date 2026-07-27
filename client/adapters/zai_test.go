package adapters

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/client/core"
	"github.com/GrayCodeAI/eyrie/types"
)

func TestNewZAIClient_WithAnthropicFallback(t *testing.T) {
	t.Parallel()
	client := NewZAIClient("zai-key", "https://zai.example/paas/v4", "https://zai.example/api/anthropic", nil, "zai_payg")
	if client == nil {
		t.Fatal("NewZAIClient returned nil")
	}
	if client.router.OpenAI == nil {
		t.Fatal("expected OpenAI client")
	}
	if client.router.Anthropic == nil {
		t.Fatal("expected Anthropic client for fallback")
	}
}

func TestNewZAIClient_WithoutAnthropicFallback(t *testing.T) {
	t.Parallel()
	client := NewZAIClient("zai-key", "https://zai.example/paas/v4", "", nil, "zai_coding")
	if client.router.Anthropic != nil {
		t.Fatal("expected no Anthropic client when anthropicBase is empty")
	}
}

func TestZAIClient_Name(t *testing.T) {
	t.Parallel()
	client := NewZAIClient("key", "https://zai.example/paas/v4", "", nil, "zai_payg")
	if client.Name() == "" {
		t.Fatal("expected non-empty Name")
	}
}

func TestZAIClient_ChatOpenAISuccess(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "Z.AI Hello!"}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3},
		}), nil
	})

	client := NewZAIClient("key", "https://zai.example/paas/v4", "", nil, "zai_payg")
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}

	resp, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "glm-4", MaxTokens: 256})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Z.AI Hello!" {
		t.Errorf("content = %q, want Z.AI Hello!", resp.Content)
	}
}

func TestZAIClient_ChatFallbackToAnthropic(t *testing.T) {
	t.Parallel()
	var paths []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		if strings.HasSuffix(req.URL.Path, "/chat/completions") {
			return jsonResponse(http.StatusServiceUnavailable, map[string]any{
				"error": map[string]string{"message": "overloaded"},
			}), nil
		}
		if strings.HasSuffix(req.URL.Path, "/messages") {
			return jsonResponse(http.StatusOK, map[string]any{
				"id": "msg_1", "type": "message", "role": "assistant",
				"content":     []map[string]string{{"type": "text", "text": "Hello from Z.AI Anthropic!"}},
				"stop_reason": "end_turn",
				"usage":       map[string]int{"input_tokens": 1, "output_tokens": 2},
			}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "OK"}, "finish_reason": "stop"}},
		}), nil
	})

	client := NewZAIClient("key", "https://zai.example/paas/v4", "https://zai.example/api/anthropic", nil, "zai_payg")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	resp, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "glm-4", MaxTokens: 256})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from Z.AI Anthropic!" {
		t.Errorf("content = %q, want Hello from Z.AI Anthropic!", resp.Content)
	}
	if len(paths) < 2 {
		t.Fatalf("expected anthropic fallback, got %v paths", len(paths))
	}
}

func TestZAIClient_StreamChatFallbackToAnthropic(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/chat/completions") {
			return jsonResponse(http.StatusServiceUnavailable, map[string]any{
				"error": map[string]string{"message": "overloaded"},
			}), nil
		}
		if strings.HasSuffix(req.URL.Path, "/messages") {
			body := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello Z.AI stream!\"}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "OK"}, "finish_reason": "stop"}},
		}), nil
	})

	client := NewZAIClient("key", "https://zai.example/paas/v4", "https://zai.example/api/anthropic", nil, "zai_payg")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	result, err := client.StreamChat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "glm-4", MaxTokens: 256})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	defer result.Close()

	var content string
	for event := range result.Events {
		if event.Type == "error" {
			t.Fatalf("unexpected stream error: %s", event.Error)
		}
		if event.Type == "content" {
			content += event.Content
		}
	}
	if content != "Hello Z.AI stream!" {
		t.Errorf("content = %q, want Hello Z.AI stream!", content)
	}
}

func TestZAIClient_PingOpenAISuccess(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
	})

	client := NewZAIClient("key", "https://zai.example/paas/v4", "", nil, "zai_payg")
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestZAIClient_PingFallbackToAnthropic(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/models") && strings.Contains(req.URL.String(), "paas") {
			return nil, &transportError{msg: "connection refused"}
		}
		return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
	})

	client := NewZAIClient("key", "https://zai.example/paas/v4", "https://zai.example/api/anthropic", nil, "zai_payg")
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping fallback: %v", err)
	}
}

func TestZaiFallbackChatError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"retriable eyrie error", &core.EyrieError{StatusCode: 503}, true},
		{"non-retriable eyrie error", &core.EyrieError{StatusCode: 400}, false},
		{"param incorrect", errors.New("param incorrect"), true},
		{"invalid format", errors.New("invalid format"), true},
		{"reasoning_content", errors.New("reasoning_content"), true},
		{"http 400 with zai", errors.New("http 400 zai error"), true},
		{"http 400 without zai", errors.New("http 400 generic"), false},
		{"generic error", errors.New("something else"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := zaiFallbackChatError(tt.err)
			if got != tt.want {
				t.Errorf("zaiFallbackChatError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestZaiRetryableChatError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"http 500 in message", errors.New("HTTP 500"), true},
		{"http 401 in message", errors.New("HTTP 401"), true},
		{"http 403 in message", errors.New("HTTP 403"), true},
		{"http 400 in message", errors.New("HTTP 400"), false},
		{"http 200 in message", errors.New("HTTP 200"), false},
		{"retriable eyrie error", &core.EyrieError{StatusCode: 503}, true},
		{"non-retriable eyrie error", &core.EyrieError{StatusCode: 400}, false},
		{"transient error", &transportError{msg: "timeout"}, true},
		{"generic error", errors.New("something"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := zaiRetryableChatError(tt.err)
			if got != tt.want {
				t.Errorf("zaiRetryableChatError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestZAIClient_Name_NilOpenAI(t *testing.T) {
	t.Parallel()
	client := &ZAIClient{providerID: "zai_custom"}
	if client.Name() != "zai_custom" {
		t.Errorf("Name() = %q, want zai_custom", client.Name())
	}
}

func TestZAIClient_Chat_NoFallbackOnNonRetriable(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "some other error"}}), nil
	})
	client := NewZAIClient("key", "https://zai.example/paas/v4", "", nil, "zai_payg")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}

	_, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "glm-4"})
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}

func TestZAIClient_StreamChat_NoFallbackOnNonRetriable(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "some other error"}}), nil
	})
	client := NewZAIClient("key", "https://zai.example/paas/v4", "", nil, "zai_payg")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}

	_, err := client.StreamChat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "glm-4"})
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}

func TestZAIClient_Ping_NoAnthropicNoFallback(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, map[string]any{"error": map[string]string{"message": "unauthorized"}}), nil
	})
	client := NewZAIClient("key", "https://zai.example/paas/v4", "", nil, "zai_payg")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}

	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error without anthropic fallback client")
	}
}
