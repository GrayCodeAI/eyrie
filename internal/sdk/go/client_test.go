package eyrie

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestClient_BaseURLConfig(t *testing.T) {
	c := NewClient("http://example.com:9999/api", "key")
	if c.baseURL != "http://example.com:9999/api" {
		t.Errorf("unexpected baseURL: %q", c.baseURL)
	}
}

func TestClient_RequestHeaders_ContentType(t *testing.T) {
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	_, _ = c.Prompt(context.Background(), PromptRequest{Message: "hello"})
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %q", contentType)
	}
}

func TestClient_Health(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method != "GET" {
		t.Errorf("expected GET, got %s", method)
	}
	if path != "/health" {
		t.Errorf("expected /health, got %s", path)
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
		w.Write(resp)
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
		w.Write(resp)
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
		w.Write(nodes)
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
		w.Write(node)
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
		w.Write(nodes)
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

func TestClient_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := `eyrie: GET /health: 400 {"error": "bad request"}`
	if err.Error() != expected {
		t.Errorf("unexpected error:\ngot:  %q\nwant: %q", err.Error(), expected)
	}
}
