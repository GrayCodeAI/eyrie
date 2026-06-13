package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewOpenCodeGoClient_NameAndModelNormalize(t *testing.T) {
	var gotModel string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(body, &payload)
		gotModel = payload.Model
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id": "chatcmpl-1", "object": "chat.completion",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hi"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		}), nil
	})

	c := NewOpenCodeGoClient("test-key", "https://opencode.example/zen/go/v1")
	c.inner.httpClient = &http.Client{Transport: transport}

	if c.Name() != "opencodego" {
		t.Fatalf("name = %q, want opencodego", c.Name())
	}

	_, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "Hi"}}, ChatOptions{
		Model:     "opencode-go/kimi-k2.6",
		MaxTokens: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotModel != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6", gotModel)
	}
}

func TestOpenCodeGoClient_PingUsesModelsEndpoint(t *testing.T) {
	var path string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		path = r.URL.Path
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"object": "list",
			"data":   []map[string]string{{"id": "glm-5"}},
		}), nil
	})
	c := NewOpenCodeGoClient("test-key", "https://opencode.example/zen/go/v1")
	c.inner.httpClient = &http.Client{Transport: transport}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "/models") {
		t.Fatalf("ping path = %q, want suffix /models", path)
	}
}
