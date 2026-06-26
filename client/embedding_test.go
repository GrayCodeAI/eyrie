package client

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultEmbeddingParamsCohere(t *testing.T) {
	t.Parallel()
	tests := []string{"cohere-embed-v3", "embed-v2", "embed-english-v3.0"}
	for _, model := range tests {
		p := DefaultEmbeddingParams(model)
		if p.Indexing["input_type"] != "search_document" {
			t.Errorf("model %q: Indexing[input_type] = %q, want %q", model, p.Indexing["input_type"], "search_document")
		}
		if p.Query["input_type"] != "search_query" {
			t.Errorf("model %q: Query[input_type] = %q, want %q", model, p.Query["input_type"], "search_query")
		}
	}
}

func TestDefaultEmbeddingParamsVoyage(t *testing.T) {
	t.Parallel()
	tests := []string{"voyage-2", "voyage-large-2", "voyage-code-2"}
	for _, model := range tests {
		p := DefaultEmbeddingParams(model)
		if p.Indexing["input_type"] != "document" {
			t.Errorf("model %q: Indexing[input_type] = %q, want %q", model, p.Indexing["input_type"], "document")
		}
		if p.Query["input_type"] != "query" {
			t.Errorf("model %q: Query[input_type] = %q, want %q", model, p.Query["input_type"], "query")
		}
	}
}

func TestDefaultEmbeddingParamsNomic(t *testing.T) {
	t.Parallel()
	p := DefaultEmbeddingParams("nomic-embed-text-v1")
	if len(p.Indexing) != 0 {
		t.Errorf("Nomic Indexing should be empty, got %v", p.Indexing)
	}
	if p.Query["prompt_name"] != "query" {
		t.Errorf("Nomic Query[prompt_name] = %q, want %q", p.Query["prompt_name"], "query")
	}
}

func TestDefaultEmbeddingParamsGemini(t *testing.T) {
	t.Parallel()
	tests := []string{"gemini-embedding-001", "text-embedding-004"}
	for _, model := range tests {
		p := DefaultEmbeddingParams(model)
		if p.Indexing["task_type"] != "RETRIEVAL_DOCUMENT" {
			t.Errorf("model %q: Indexing[task_type] = %q, want %q", model, p.Indexing["task_type"], "RETRIEVAL_DOCUMENT")
		}
		if p.Query["task_type"] != "RETRIEVAL_QUERY" {
			t.Errorf("model %q: Query[task_type] = %q, want %q", model, p.Query["task_type"], "RETRIEVAL_QUERY")
		}
	}
}

func TestDefaultEmbeddingParamsUnknown(t *testing.T) {
	t.Parallel()
	p := DefaultEmbeddingParams("some-unknown-model")
	if len(p.Indexing) != 0 || len(p.Query) != 0 {
		t.Errorf("unknown model should return empty params, got Indexing=%v Query=%v", p.Indexing, p.Query)
	}
}

func TestDefaultEmbeddingParamsCaseInsensitive(t *testing.T) {
	t.Parallel()
	p := DefaultEmbeddingParams("COHERE-EMBED-V3")
	if p.Indexing["input_type"] != "search_document" {
		t.Errorf("case-insensitive lookup failed: Indexing[input_type] = %q", p.Indexing["input_type"])
	}
}

func TestCreateEmbeddingSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "text-embedding-ada-002" {
			t.Errorf("unexpected model: %v", body["model"])
		}

		json.NewEncoder(w).Encode(openaiEmbeddingResponse{
			Object: "list",
			Model:  "text-embedding-ada-002",
			Data: []openaiEmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
				{Object: "embedding", Index: 1, Embedding: []float64{0.4, 0.5, 0.6}},
			},
			Usage: openaiEmbeddingUsage{PromptTokens: 8, TotalTokens: 8},
		})
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.CreateEmbedding(context.Background(), EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: []string{"hello", "world"},
	})
	if err != nil {
		t.Fatalf("CreateEmbedding: %v", err)
	}
	if resp.Model != "text-embedding-ada-002" {
		t.Errorf("Model = %q, want %q", resp.Model, "text-embedding-ada-002")
	}
	if len(resp.Embeddings) != 2 {
		t.Fatalf("Embeddings length = %d, want 2", len(resp.Embeddings))
	}
	if len(resp.Embeddings[0]) != 3 {
		t.Errorf("Embeddings[0] length = %d, want 3", len(resp.Embeddings[0]))
	}
	if resp.Embeddings[0][0] != 0.1 {
		t.Errorf("Embeddings[0][0] = %f, want 0.1", resp.Embeddings[0][0])
	}
	if resp.Embeddings[1][2] != 0.6 {
		t.Errorf("Embeddings[1][2] = %f, want 0.6", resp.Embeddings[1][2])
	}
	if resp.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if resp.Usage.PromptTokens != 8 {
		t.Errorf("Usage.PromptTokens = %d, want 8", resp.Usage.PromptTokens)
	}
}

func TestCreateEmbeddingMissingModel(t *testing.T) {
	t.Parallel()
	c := newTestOpenAIClient("http://unused", nil)
	_, err := c.CreateEmbedding(context.Background(), EmbeddingRequest{
		Input: []string{"hello"},
	})
	if err == nil {
		t.Fatal("CreateEmbedding should fail without model")
	}
}

func TestCreateEmbeddingServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	_, err := c.CreateEmbedding(context.Background(), EmbeddingRequest{
		Model: "test-model",
		Input: []string{"hello"},
	})
	if err == nil {
		t.Fatal("CreateEmbedding should fail on server error")
	}
}

func TestCreateEmbeddingWithParams(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["input_type"] != "search_document" {
			t.Errorf("expected extra param input_type=search_document, got %v", body["input_type"])
		}

		json.NewEncoder(w).Encode(openaiEmbeddingResponse{
			Object: "list",
			Model:  "test-model",
			Data: []openaiEmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1}},
			},
		})
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.CreateEmbedding(context.Background(), EmbeddingRequest{
		Model:  "test-model",
		Input:  []string{"hello"},
		Params: map[string]string{"input_type": "search_document"},
	})
	if err != nil {
		t.Fatalf("CreateEmbedding: %v", err)
	}
	if len(resp.Embeddings) != 1 {
		t.Fatalf("Embeddings length = %d, want 1", len(resp.Embeddings))
	}
}

func TestCreateEmbeddingNoUsage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openaiEmbeddingResponse{
			Object: "list",
			Model:  "test-model",
			Data: []openaiEmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.5}},
			},
			Usage: openaiEmbeddingUsage{}, // zero usage
		})
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.CreateEmbedding(context.Background(), EmbeddingRequest{
		Model: "test-model",
		Input: []string{"hello"},
	})
	if err != nil {
		t.Fatalf("CreateEmbedding: %v", err)
	}
	if resp.Usage != nil {
		t.Error("Usage should be nil when both prompt and total tokens are zero")
	}
}

func TestCreateEmbeddingFloat32Precision(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openaiEmbeddingResponse{
			Object: "list",
			Model:  "test-model",
			Data: []openaiEmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.123456789012345}},
			},
		})
	}))
	defer srv.Close()

	c := newTestOpenAIClient(srv.URL, nil)
	resp, err := c.CreateEmbedding(context.Background(), EmbeddingRequest{
		Model: "test-model",
		Input: []string{"test"},
	})
	if err != nil {
		t.Fatalf("CreateEmbedding: %v", err)
	}
	// float64 -> float32 conversion
	expected := float32(0.123456789012345)
	if math.Abs(float64(resp.Embeddings[0][0]-expected)) > 1e-6 {
		t.Errorf("Embeddings[0][0] = %f, want %f (within float32 precision)", resp.Embeddings[0][0], expected)
	}
}

func TestEmbeddingRequestStruct(t *testing.T) {
	t.Parallel()
	req := EmbeddingRequest{
		Model:  "test",
		Input:  []string{"a", "b"},
		Params: map[string]string{"key": "value"},
	}
	if req.Model != "test" {
		t.Errorf("Model = %q, want %q", req.Model, "test")
	}
	if len(req.Input) != 2 {
		t.Errorf("Input length = %d, want 2", len(req.Input))
	}
	if req.Params["key"] != "value" {
		t.Errorf("Params[key] = %q, want %q", req.Params["key"], "value")
	}
}

func TestEmbeddingResponseStruct(t *testing.T) {
	t.Parallel()
	resp := &EmbeddingResponse{
		Embeddings: [][]float32{{0.1, 0.2}},
		Model:      "test",
		Usage:      &EyrieUsage{PromptTokens: 5, TotalTokens: 5},
	}
	if resp.Model != "test" {
		t.Errorf("Model = %q, want %q", resp.Model, "test")
	}
	if len(resp.Embeddings) != 1 {
		t.Errorf("Embeddings length = %d, want 1", len(resp.Embeddings))
	}
	if resp.Usage.TotalTokens != 5 {
		t.Errorf("Usage.TotalTokens = %d, want 5", resp.Usage.TotalTokens)
	}
}

func TestEmbeddingParamsStruct(t *testing.T) {
	t.Parallel()
	p := EmbeddingParams{
		Indexing: map[string]string{"task": "document"},
		Query:    map[string]string{"task": "query"},
	}
	if p.Indexing["task"] != "document" {
		t.Errorf("Indexing[task] = %q, want %q", p.Indexing["task"], "document")
	}
	if p.Query["task"] != "query" {
		t.Errorf("Query[task] = %q, want %q", p.Query["task"], "query")
	}
}
