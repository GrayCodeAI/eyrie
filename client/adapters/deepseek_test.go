package adapters

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/client/core"
	"github.com/GrayCodeAI/eyrie/types"
)

func TestNewDeepSeekClient_WithAnthropicFallback(t *testing.T) {
	t.Parallel()
	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "https://api.deepseek.com/anthropic", nil)
	if client == nil {
		t.Fatal("NewDeepSeekClient returned nil")
	}
	if client.Name() != "deepseek" {
		t.Errorf("expected name 'deepseek', got %q", client.Name())
	}
	if client.router.OpenAI == nil {
		t.Fatal("expected OpenAI client")
	}
	if client.router.Anthropic == nil {
		t.Fatal("expected Anthropic client for fallback")
	}
}

func TestNewDeepSeekClient_WithoutAnthropicFallback(t *testing.T) {
	t.Parallel()
	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "", nil)
	if client.router.Anthropic != nil {
		t.Fatal("expected no Anthropic client when anthropicBase is empty")
	}
}

func TestDeepSeekClient_Name(t *testing.T) {
	t.Parallel()
	client := NewDeepSeekClient("key", "https://api.deepseek.com/v1", "", nil)
	if client.Name() != "deepseek" {
		t.Errorf("expected 'deepseek', got %q", client.Name())
	}
}

func TestDeepSeekClient_ChatOpenAISuccess(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3},
		}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "", nil)
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}

	resp, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "deepseek-chat", MaxTokens: 256})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("content = %q, want Hello!", resp.Content)
	}
}

func TestDeepSeekClient_ChatFallbackToAnthropic(t *testing.T) {
	t.Parallel()
	var paths []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		if strings.HasSuffix(req.URL.Path, "/chat/completions") {
			return jsonResponse(http.StatusServiceUnavailable, map[string]any{
				"error": map[string]string{"message": "service unavailable"},
			}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant",
			"content":     []map[string]string{{"type": "text", "text": "Hello from Anthropic!"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 1, "output_tokens": 2},
		}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "https://api.deepseek.com/anthropic", nil)
	// Disable retries to speed up fallback test
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	resp, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "deepseek-chat", MaxTokens: 256})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from Anthropic!" {
		t.Errorf("content = %q, want Hello from Anthropic!", resp.Content)
	}
	if len(paths) < 2 {
		t.Fatalf("expected fallback to anthropic, got %v paths", len(paths))
	}
}

func TestDeepSeekClient_StreamChatFallbackToAnthropic(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/chat/completions") {
			return jsonResponse(http.StatusServiceUnavailable, map[string]any{
				"error": map[string]string{"message": "service unavailable"},
			}), nil
		}
		if strings.HasSuffix(req.URL.Path, "/messages") {
			body := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello stream!\"}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "OK"}, "finish_reason": "stop"}},
		}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "https://api.deepseek.com/anthropic", nil)
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	result, err := client.StreamChat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "deepseek-chat", MaxTokens: 256})
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
	if content != "Hello stream!" {
		t.Errorf("content = %q, want Hello stream!", content)
	}
}

func TestDeepSeekClient_PingOpenAISuccess(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "", nil)
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestDeepSeekClient_PingFallbackToAnthropic(t *testing.T) {
	t.Parallel()
	var paths []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		// Only fail the OpenAI Ping, not the Anthropic Ping
		if strings.HasSuffix(req.URL.Path, "/models") && strings.Contains(req.URL.String(), "api.deepseek.com/v1") {
			return nil, &transportError{msg: "connection refused"}
		}
		return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "https://api.deepseek.com/anthropic", nil)
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping fallback: %v", err)
	}
	if len(paths) < 2 {
		t.Fatalf("expected anthropic fallback, got %v paths", len(paths))
	}
}

func TestDeepSeekClient_StreamChatNoFallbackWhenAnthropicNil(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{"message": "service unavailable"},
		}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "", nil)
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}

	_, err := client.StreamChat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "deepseek-chat", MaxTokens: 256})
	if err == nil {
		t.Fatal("expected error when no fallback available")
	}
}

func TestDeepSeekClient_PingNoFallbackWhenAnthropicNil(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "unauthorized"},
		}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "", nil)
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}

	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error when no fallback available")
	}
}

func TestDeepSeekClient_PingNonRetriableError(t *testing.T) {
	t.Parallel()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "unauthorized"},
		}), nil
	})

	client := NewDeepSeekClient("ds-key", "https://api.deepseek.com/v1", "https://api.deepseek.com/anthropic", nil)
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: transport}
	client.router.Anthropic.httpClient = &http.Client{Transport: transport}

	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error for non-retriable error")
	}
}
