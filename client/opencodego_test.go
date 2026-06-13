package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog/opencodego"
)

func TestOpenCodeGoUsesMessagesAPI(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"minimax-m2.5", true},
		{"opencodego/minimax-m2.5", true},
		{"qwen3.7-max", true},
		{"kimi-k2.5", false},
		{"glm-5", false},
		{"mimo-v2.5-pro", false},
	}
	for _, tc := range tests {
		if got := opencodego.UsesMessagesAPI(tc.model); got != tc.want {
			t.Errorf("UsesMessagesAPI(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestOpenCodeGoAnthropicBase(t *testing.T) {
	if got := AnthropicBaseFromOpenAIV1("https://opencode.ai/zen/go/v1"); got != "https://opencode.ai/zen/go" {
		t.Fatalf("base = %q, want https://opencode.ai/zen/go", got)
	}
}

func TestOpenCodeGoOACompatUnsupportedError(t *testing.T) {
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
		if got := oaCompatUnsupportedError(tc.err); got != tc.want {
			t.Errorf("oaCompatUnsupportedError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestOpenCodeGoClient_RoutesMiniMaxToAnthropic(t *testing.T) {
	var gotPath, gotAuth string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("X-Api-Key")
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "msg_1", "type": "message", "role": "assistant",
			"content":     []map[string]string{{"type": "text", "text": "Hello!"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 1, "output_tokens": 2},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.router.Anthropic.httpClient = &http.Client{Transport: transport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model: "minimax-m2.5", MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Fatalf("content = %q, want Hello!", resp.Content)
	}
	if !strings.HasSuffix(gotPath, "/v1/messages") {
		t.Fatalf("path = %q, want suffix /v1/messages", gotPath)
	}
	if gotAuth != "ocg-test-key" {
		t.Fatalf("X-Api-Key = %q, want ocg-test-key", gotAuth)
	}
}

func TestOpenCodeGoClient_RoutesKimiToOpenAI(t *testing.T) {
	var gotPath string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hi there!"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.router.OpenAI.httpClient = &http.Client{Transport: transport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model: "kimi-k2.5", MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hi there!" {
		t.Fatalf("content = %q, want Hi there!", resp.Content)
	}
	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Fatalf("path = %q, want suffix /chat/completions", gotPath)
	}
}

func TestOpenCodeGoClient_Qwen401FallsBackToOpenAI(t *testing.T) {
	var paths []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/messages") {
			return jsonResponse(http.StatusUnauthorized, map[string]interface{}{
				"error": map[string]string{"message": "Invalid API key"},
			}), nil
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "OK"}, "finish_reason": "stop"},
			},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.router.Anthropic.httpClient = &http.Client{Transport: transport}
	c.router.OpenAI.httpClient = &http.Client{Transport: transport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model: "qwen3.7-max", MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "OK" {
		t.Fatalf("content = %q, want OK; paths=%v", resp.Content, paths)
	}
}

func TestOpenCodeGoClient_MessagesEmptyFallsBackToOpenAI(t *testing.T) {
	var paths []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/messages") {
			return jsonResponse(http.StatusOK, map[string]interface{}{
				"id": "msg_1", "type": "message", "role": "assistant",
				"content":     []map[string]string{{"type": "thinking", "thinking": "hmm"}},
				"stop_reason": "end_turn",
			}), nil
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"},
			},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.router.Anthropic.httpClient = &http.Client{Transport: transport}
	c.router.OpenAI.httpClient = &http.Client{Transport: transport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model: "minimax-m3", MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Fatalf("content = %q, want Hello!; paths=%v", resp.Content, paths)
	}
	if len(paths) < 2 {
		t.Fatalf("expected anthropic then openai, got %v", paths)
	}
}

func TestOpenCodeGoClient_NormalizesModelID(t *testing.T) {
	var gotModel string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/chat/completions") {
			var body struct {
				Model string `json:"model"`
			}
			_ = jsonDecodeRequest(r, &body)
			gotModel = body.Model
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hi"}, "finish_reason": "stop"},
			},
		}), nil
	})
	c := NewOpenCodeGoClient("key", "https://opencode.example/zen/go/v1")
	c.router.OpenAI.httpClient = &http.Client{Transport: transport}
	_, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model: "opencode-go/kimi-k2.6", MaxTokens: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotModel != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6", gotModel)
	}
}

func TestOpenCodeGoClient_StreamMiniMaxReasoningOnlyFallsBackToChat(t *testing.T) {
	var paths []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/messages") {
			body := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\n" +
				"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"thinking\"}}\n\n" +
				"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"text\":\"hmm\"}}\n\n" +
				"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
				"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
				"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}
		if strings.HasSuffix(r.URL.Path, "/chat/completions") && r.Header.Get("Accept") == "text/event-stream" {
			t.Fatal("stream fallback should not run before non-streaming chat fallback")
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"},
			},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.router.Anthropic.httpClient = &http.Client{Transport: transport}
	c.router.OpenAI.httpClient = &http.Client{Transport: transport}
	result, err := c.StreamChat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hello how are you?"}}, ChatOptions{
		Model: "minimax-m3", MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	defer result.Close()

	var content string
	for ev := range result.Events {
		switch ev.Type {
		case "thinking":
			t.Fatal("reasoning-only primary stream must not leak thinking before chat fallback")
		case "content":
			content += ev.Content
		case "error":
			t.Fatalf("unexpected stream error: %s", ev.Error)
		}
	}
	if content != "Hello!" {
		t.Fatalf("content = %q, want Hello!; paths=%v", content, paths)
	}
	if len(paths) < 2 {
		t.Fatalf("expected /messages then /chat/completions, got %v", paths)
	}
}

func jsonDecodeRequest(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
