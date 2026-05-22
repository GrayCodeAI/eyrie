package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchKimi_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-kimi-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"moonshotai/kimi-k2","display_name":"Kimi K2","status":1,"owned_by":"moonshot"}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchKimi(map[string]string{
		"MOONSHOT_API_KEY": "test-kimi-key",
		"MOONSHOT_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 model, got %d", len(entries))
	}
	if entries[0].ID != "moonshotai/kimi-k2" {
		t.Fatalf("id = %q", entries[0].ID)
	}
	if entries[0].OwnedBy != "moonshot" {
		t.Fatalf("owned_by = %q", entries[0].OwnedBy)
	}
	if len(entries[0].RawJSON) == 0 {
		t.Fatal("expected RawJSON to be preserved")
	}
}

func TestFetchKimi_NoKey(t *testing.T) {
	entries, err := FetchKimi(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}
