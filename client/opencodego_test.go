package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenCodeGoUsesAnthropicMessages(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"minimax-m2.5", true},
		{"opencodego/minimax-m2.5", true},
		{"minimax-m2.7", true},
		{"minimax-m3", true},
		{"MiniMax-M2.5-highspeed", true},
		{"qwen3.5-plus", true},
		{"qwen3.7-max", true},
		{"kimi-k2.5", false},
		{"glm-5", false},
		{"mimo-v2.5-pro", false},
		{"deepseek-v4-flash", false},
	}
	for _, tc := range tests {
		if got := openCodeGoUsesAnthropicMessages(tc.model); got != tc.want {
			t.Errorf("openCodeGoUsesAnthropicMessages(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestOpenCodeGoAnthropicBase(t *testing.T) {
	if got := openCodeGoAnthropicBase("https://opencode.ai/zen/go/v1"); got != "https://opencode.ai/zen/go" {
		t.Fatalf("base = %q, want https://opencode.ai/zen/go", got)
	}
}

func TestOpenCodeGoClient_RoutesMiniMaxToAnthropic(t *testing.T) {
	var gotPath string
	var gotAuth string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("X-Api-Key")
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id":          "msg_1",
			"type":        "message",
			"role":        "assistant",
			"content":     []map[string]string{{"type": "text", "text": "Hello!"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 1, "output_tokens": 2},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.anthropic.httpClient = &http.Client{Transport: transport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model:     "minimax-m2.5",
		MaxTokens: 256,
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
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"choices": []map[string]interface{}{{"message": map[string]string{"role": "assistant", "content": "Hi there!"}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3},
		}), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.openAI.httpClient = &http.Client{Transport: transport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model:     "kimi-k2.5",
		MaxTokens: 256,
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

func TestOpenCodeGoClient_AnthropicReasoningOnlyFallsBackToOpenAI(t *testing.T) {
	var paths []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/messages") {
			w := httptest.NewRecorder()
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"thinking\"}}\n\n")
			_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"text\":\"hmm\"}}\n\n")
			_, _ = fmt.Fprintf(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
			_, _ = fmt.Fprintf(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
			return w.Result(), nil
		}
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello!\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		return w.Result(), nil
	})

	c := NewOpenCodeGoClient("ocg-test-key", "https://opencode.example/zen/go/v1")
	c.anthropic.httpClient = &http.Client{Transport: transport}
	c.openAI.httpClient = &http.Client{Transport: transport}

	sr, err := c.StreamChat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model:     "minimax-m3",
		MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	defer sr.Close()

	var content string
	for ev := range sr.Events {
		if ev.Type == "content" {
			content += ev.Content
		}
	}
	if content != "Hello!" {
		t.Fatalf("content = %q, want Hello!; paths=%v", content, paths)
	}
	if len(paths) < 2 {
		t.Fatalf("expected anthropic then openai paths, got %v", paths)
	}
}
