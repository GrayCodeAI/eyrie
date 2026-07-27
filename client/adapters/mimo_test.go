package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/client/core"
	"github.com/GrayCodeAI/eyrie/types"
)

func TestMiMoClientChatFallsBackToAnthropicOnParamIncorrect(t *testing.T) {
	t.Parallel()
	anthropicCalls := 0
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/chat/completions" {
			t.Fatalf("openai path = %q", req.URL.Path)
		}
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "Param Incorrect"}}), nil
	})
	anthropicTransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		anthropicCalls++
		if req.URL.Path != "/anthropic/v1/messages" {
			t.Fatalf("anthropic path = %q", req.URL.Path)
		}
		if req.Header.Get("api-key") != "tp-test-key" {
			t.Fatalf("missing MiMo api-key auth header")
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"id":          "msg_1",
			"type":        "message",
			"role":        "assistant",
			"content":     []map[string]string{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 1, "output_tokens": 1},
		}), nil
	})

	client := NewMiMoClient("tp-test-key", "https://openai.example/v1", "https://anthropic.example/anthropic", &XiaomiCompat, "xiaomi_mimo_token_plan")
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	client.router.Anthropic.httpClient = &http.Client{Transport: anthropicTransport}
	response, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "hi"}}, core.ChatOptions{
		Model:     "mimo-v2.5-pro",
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if response.Content != "ok" {
		t.Fatalf("content = %q, want ok", response.Content)
	}
	if anthropicCalls != 1 {
		t.Fatalf("anthropic calls = %d, want 1", anthropicCalls)
	}
}

func TestMiMoClientPreservesProtocolBaseURLs(t *testing.T) {
	t.Parallel()
	client := NewMiMoClient(
		"key",
		"https://openai.example/v1/",
		"https://anthropic.example/anthropic/",
		&XiaomiCompat,
		"xiaomi_mimo_token_plan",
	)
	if got := client.router.OpenAI.baseURL; got != "https://openai.example/v1" {
		t.Fatalf("OpenAI base URL = %q", got)
	}
	if client.router.Anthropic == nil {
		t.Fatal("Anthropic fallback client is nil")
	}
	if got := client.router.Anthropic.baseURL; got != "https://anthropic.example/anthropic" {
		t.Fatalf("Anthropic base URL = %q", got)
	}
}

func TestMiMoClient_Name(t *testing.T) {
	t.Parallel()
	client := NewMiMoClient("key", "https://oai.example/v1", "https://ant.example", &XiaomiCompat, "providerA")
	if client.Name() != "providerA" {
		t.Errorf("Name() = %q, want providerA", client.Name())
	}
}

func TestMiMoClient_Name_NoOpenAI(t *testing.T) {
	t.Parallel()
	c := &MiMoClient{providerID: "bare-id"}
	if c.Name() != "bare-id" {
		t.Errorf("Name() = %q, want bare-id", c.Name())
	}
}

func TestMiMoClient_ProviderID(t *testing.T) {
	t.Parallel()
	client := NewMiMoClient("key", "https://oai.example/v1", "", &XiaomiCompat, "my_provider")
	if client.ProviderID() != "my_provider" {
		t.Errorf("ProviderID = %q", client.ProviderID())
	}
}

func TestMiMoClient_Ping_SuccessViaOpenAI(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{"data": []map[string]any{}}), nil
	})
	client := NewMiMoClient("key", "https://oai.example/v1", "https://ant.example", &XiaomiCompat, "p")
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestMiMoClient_Ping_FallbackToAnthropic(t *testing.T) {
	t.Parallel()
	anthropicPinged := false
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("connection refused")
	})
	anthropicTransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		anthropicPinged = true
		return jsonResponse(http.StatusOK, nil), nil
	})
	client := NewMiMoClient("key", "https://oai.example/v1", "https://ant.example", &XiaomiCompat, "p")
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	client.router.Anthropic.httpClient = &http.Client{Transport: anthropicTransport}
	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !anthropicPinged {
		t.Fatal("expected Anthropic ping fallback")
	}
}

func TestMiMoClient_Ping_NoFallbackOnNonRetryable(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("some unknown error")
	})
	client := NewMiMoClient("key", "https://oai.example/v1", "https://ant.example", &XiaomiCompat, "p")
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected ping error when no fallback")
	}
}

func TestMiMoClient_Ping_NoAnthropicNoFallback(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("some unknown error")
	})
	client := NewMiMoClient("key", "https://oai.example/v1", "", &XiaomiCompat, "p")
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error without anthropic fallback client")
	}
}

func TestMiMoClient_Chat_NoFallbackOnNonRetriable(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "some other error"}}), nil
	})
	client := NewMiMoClient("key", "https://oai.example/v1", "", &XiaomiCompat, "p")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}

	_, err := client.Chat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "mimo"})
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}

func TestMiMoClient_StreamChat_NoFallbackOnNonRetriable(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "some other error"}}), nil
	})
	client := NewMiMoClient("key", "https://oai.example/v1", "", &XiaomiCompat, "p")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}

	_, err := client.StreamChat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "Hi"}}, core.ChatOptions{Model: "mimo"})
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}

func TestMiMoClient_StreamChat_FallbackToAnthropic(t *testing.T) {
	t.Parallel()
	openAITransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]any{"error": map[string]string{"message": "param incorrect"}}), nil
	})
	anthropicTransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"stream ok\"}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	client := NewMiMoClient("key", "https://oai.example/v1", "https://ant.example", &XiaomiCompat, "p")
	client.router.OpenAI.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.Anthropic.SetRetry(core.RetryConfig{RetryConfig: types.RetryConfig{MaxRetries: 0}})
	client.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	client.router.Anthropic.httpClient = &http.Client{Transport: anthropicTransport}

	result, err := client.StreamChat(context.Background(), []core.EyrieMessage{{Role: "user", Content: "hi"}}, core.ChatOptions{Model: "mimo-pro", MaxTokens: 64})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	defer result.Close()
	var content string
	for evt := range result.Events {
		if evt.Type == "content" {
			content += evt.Content
		}
	}
	if content != "stream ok" {
		t.Errorf("content = %q, want stream ok", content)
	}
}

func TestMimoFallbackChatError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{fmt.Errorf("param incorrect"), true},
		{fmt.Errorf("invalid format"), true},
		{fmt.Errorf("reasoning_content"), true},
		{fmt.Errorf("HTTP 400 xiaomi"), true},
		{fmt.Errorf("HTTP 400"), false},
		{fmt.Errorf("something else"), false},
		{fmt.Errorf(""), false},
	}
	for _, tt := range tests {
		got := MimoFallbackChatError(tt.err)
		if got != tt.want {
			t.Errorf("MimoFallbackChatError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestMimoRetryableChatError_HTTPStatus(t *testing.T) {
	err := fmt.Errorf("HTTP 401")
	if !MimoRetryableChatError(err) {
		t.Error("expected 401 to be retryable")
	}
}

func TestMimoRetryableChatError_TransientMessage(t *testing.T) {
	err := fmt.Errorf("connection refused")
	if !MimoRetryableChatError(err) {
		t.Error("expected connection refused to be retryable")
	}
}

func TestMimoRetryableChatError_NonTransient(t *testing.T) {
	err := fmt.Errorf("some unknown error")
	if MimoRetryableChatError(err) {
		t.Error("expected unknown error to NOT be retryable")
	}
}

func TestParseHTTPStatusFromError(t *testing.T) {
	tests := []struct {
		msg  string
		want int
	}{
		{"HTTP 404 Not Found", 404},
		{"status 500 internal", 500},
		{"error (403) forbidden", 403},
		{"no status here", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := parseHTTPStatusFromError(tt.msg)
		if got != tt.want {
			t.Errorf("parseHTTPStatusFromError(%q) = %d, want %d", tt.msg, got, tt.want)
		}
	}
}
