package eyrie

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient_DefaultConfig(t *testing.T) {
	c := NewClient("http://localhost:8080", "test-api-key")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL http://localhost:8080, got %q", c.baseURL)
	}
	if c.apiKey != "test-api-key" {
		t.Errorf("expected apiKey test-api-key, got %q", c.apiKey)
	}
	if c.httpClient.Timeout != 120*time.Second {
		t.Errorf("expected timeout 120s, got %v", c.httpClient.Timeout)
	}
}

func TestClient_APIKey(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key")
	_ = c.Health(context.Background())
	if authHeader != "Bearer secret-key" {
		t.Errorf("expected 'Bearer secret-key', got %q", authHeader)
	}
}

func TestClient_EmptyAPIKey_NoAuthHeader(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_ = c.Health(context.Background())
	if authHeader != "" {
		t.Errorf("expected no Authorization header, got %q", authHeader)
	}
}

func TestClient_Prompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/prompt" {
			t.Errorf("expected /prompt, got %s", r.URL.Path)
		}
		var req PromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Message != "test message" {
			t.Errorf("expected message 'test message', got %q", req.Message)
		}
		resp, _ := json.Marshal(PromptResponse{Content: "response content", NodeID: "node-123"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	result, err := c.Prompt(context.Background(), PromptRequest{Message: "test message"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "response content" {
		t.Errorf("expected 'response content', got %q", result.Content)
	}
	if result.NodeID != "node-123" {
		t.Errorf("expected 'node-123', got %q", result.NodeID)
	}
}

func TestClient_PromptWithTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req PromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if len(req.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Name != "test_tool" {
			t.Errorf("tool name = %q", req.Tools[0].Name)
		}
		resp, _ := json.Marshal(PromptResponse{Content: "tool response"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	_, err := c.Prompt(context.Background(), PromptRequest{
		Message: "use a tool",
		Tools:   []ToolDef{{Name: "test_tool", Description: "A test tool"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_PromptFrom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/nodes/parent-456/prompt" {
			t.Errorf("expected /nodes/parent-456/prompt, got %s", r.URL.Path)
		}
		resp, _ := json.Marshal(PromptResponse{Content: "child response", NodeID: "child-789"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	result, err := c.PromptFrom(context.Background(), "parent-456", PromptRequest{Message: "continue"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "child response" {
		t.Errorf("expected 'child response', got %q", result.Content)
	}
}

func TestClient_StreamPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for i, content := range []string{"Hello", " World", "!"} {
			evt, _ := json.Marshal(map[string]any{
				"type": "delta",
				"data": map[string]string{"content": content},
			})
			fmt.Fprintf(w, "data: %s\n\n", evt)
			flusher.Flush()
			if i == 2 {
				done, _ := json.Marshal(map[string]string{"type": "done"})
				fmt.Fprintf(w, "data: %s\n\n", done)
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	events, err := c.StreamPrompt(context.Background(), PromptRequest{Message: "stream test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var contents []string
	for evt := range events {
		if evt.Type == "delta" {
			var data map[string]string
			json.Unmarshal(evt.Data, &data)
			contents = append(contents, data["content"])
		}
	}
	result := strings.Join(contents, "")
	if result != "Hello World!" {
		t.Errorf("expected 'Hello World!', got %q", result)
	}
}

func TestClient_ListConversations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/nodes" {
			t.Errorf("expected /nodes, got %s", r.URL.Path)
		}
		nodes, _ := json.Marshal([]Node{
			{ID: "1", Content: "hello", Model: "gpt-4", Sequence: 1, NodeType: "conversation"},
			{ID: "2", Content: "world", Model: "gpt-4", Sequence: 2, NodeType: "conversation"},
		})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(nodes)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	results, err := c.ListConversations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(results))
	}
	if results[0].ID != "1" {
		t.Errorf("expected ID '1', got %q", results[0].ID)
	}
}

func TestClient_GetNode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/nodes/node-42" {
			t.Errorf("expected /nodes/node-42, got %s", r.URL.Path)
		}
		node, _ := json.Marshal(Node{ID: "node-42", Content: "node content", Model: "claude-3"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(node)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	node, err := c.GetNode(context.Background(), "node-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.ID != "node-42" {
		t.Errorf("expected 'node-42', got %q", node.ID)
	}
}

func TestClient_GetTree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nodes/root-1/tree" {
			t.Errorf("expected /nodes/root-1/tree, got %s", r.URL.Path)
		}
		nodes, _ := json.Marshal([]Node{
			{ID: "root-1", Sequence: 1},
			{ID: "child-1", Sequence: 2, ParentID: "root-1"},
		})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(nodes)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	nodes, err := c.GetTree(context.Background(), "root-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestClient_DeleteNode(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	err := c.DeleteNode(context.Background(), "node-to-delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != "DELETE" {
		t.Errorf("expected DELETE, got %s", method)
	}
	if path != "/nodes/node-to-delete" {
		t.Errorf("expected /nodes/node-to-delete, got %s", path)
	}
}

func TestClient_CreateAlias(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		resp, _ := json.Marshal(AliasResult{Alias: "my-alias", NodeID: "node-123"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	result, err := c.CreateAlias(context.Background(), "node-123", "my-alias")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != "PUT" {
		t.Errorf("expected PUT, got %s", method)
	}
	if path != "/nodes/node-123/aliases/my-alias" {
		t.Errorf("expected /nodes/node-123/aliases/my-alias, got %s", path)
	}
	if result.Alias != "my-alias" {
		t.Errorf("alias = %q", result.Alias)
	}
}

func TestClient_DeleteAlias(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	err := c.DeleteAlias(context.Background(), "my-alias")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != "DELETE" {
		t.Errorf("expected DELETE, got %s", method)
	}
	if path != "/aliases/my-alias" {
		t.Errorf("expected /aliases/my-alias, got %s", path)
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("status = %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "bad request") {
		t.Errorf("error body missing: %s", apiErr.Error())
	}
}

func TestClient_StreamPromptReturnsErrorOnNonStream(t *testing.T) {
	c := NewClient("http://localhost:9999", "key")
	_, err := c.Prompt(context.Background(), PromptRequest{Message: "test", Stream: true})
	if err == nil {
		t.Fatal("expected error for stream=true with Prompt")
	}
}
