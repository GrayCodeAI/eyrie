package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchXiaomi_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-xiaomi-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"mimo/model-x","display_name":"MiMo Model X","status":1}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchXiaomi(map[string]string{
		"XIAOMI_API_KEY":  "test-xiaomi-key",
		"XIAOMI_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 model, got %d", len(entries))
	}
	if entries[0].ID != "mimo/model-x" {
		t.Fatalf("id = %q", entries[0].ID)
	}
	if len(entries[0].RawJSON) == 0 {
		t.Fatal("expected RawJSON to be preserved")
	}
}

func TestFetchXiaomi_NoKey(t *testing.T) {
	entries, err := FetchXiaomi(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}
