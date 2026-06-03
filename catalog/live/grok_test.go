package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchGrok_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/grok_models.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	entries, err := FetchGrok(map[string]string{
		"XAI_API_KEY":  "xai-test123",
		"XAI_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 models, got %d", len(entries))
	}
}

func TestFetchGrok_NoKey(t *testing.T) {
	entries, err := FetchGrok(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchGrok_ParsesProviderFields(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "grok-3-beta",
		"owned_by": "xai",
		"display_name": "Grok 3 Beta"
	}`)
	entry, ok := entryFromOpenAICompatJSON(raw)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if entry.ID != "grok-3-beta" {
		t.Fatalf("id = %q", entry.ID)
	}
	if entry.OwnedBy != "xai" {
		t.Fatalf("owned_by = %q", entry.OwnedBy)
	}
	if entry.DisplayName != "Grok 3 Beta" {
		t.Fatalf("display_name = %q", entry.DisplayName)
	}
}
