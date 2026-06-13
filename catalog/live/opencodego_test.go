package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchOpenCodeGo_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-ocg-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"kimi-k2.6","owned_by":"opencode"}`),
				json.RawMessage(`{"id":"minimax-m2.7","owned_by":"opencode"}`),
				json.RawMessage(`{"id":"qwen3.7-max","owned_by":"opencode"}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchOpenCodeGo(map[string]string{
		"OPENCODEGO_API_KEY":  "test-ocg-key",
		"OPENCODEGO_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected all 3 models, got %d", len(entries))
	}
	if entries[0].ID != "kimi-k2.6" {
		t.Fatalf("id = %q", entries[0].ID)
	}
}

func TestFetchOpenCodeGo_NoKey(t *testing.T) {
	entries, err := FetchOpenCodeGo(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}
