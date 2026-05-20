package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchOpenRouter_Mock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		ctx := 200000
		maxComp := 32000
		resp := struct {
			Data []openRouterModel `json:"data"`
		}{
			Data: []openRouterModel{{
				ID:            "anthropic/claude-sonnet-4-6",
				ContextLength: &ctx,
				TopProvider: &struct {
					ContextLength       *int `json:"context_length"`
					MaxCompletionTokens *int `json:"max_completion_tokens"`
				}{MaxCompletionTokens: &maxComp},
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchOpenRouter(map[string]string{
		"OPENROUTER_API_KEY":  "test-key-12345",
		"OPENROUTER_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestFetchOllama_EmptyModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	}))
	defer srv.Close()

	entries, err := FetchOllama(map[string]string{"OLLAMA_BASE_URL": srv.URL + "/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
