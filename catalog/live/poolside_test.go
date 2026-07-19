package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchPoolside_MockHTTPServer(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("testdata/poolside_models.json")
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

	entries, err := FetchPoolside(map[string]string{
		"POOLSIDE_API_KEY":  "ps-test123",
		"POOLSIDE_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 models, got %d", len(entries))
	}
}

func TestFetchPoolside_NoKey(t *testing.T) {
	t.Parallel()
	entries, err := FetchPoolside(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchPoolside_ParsesProviderFields(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{
		"id": "poolside/laguna-m.1",
		"owned_by": "poolside",
		"display_name": "Poolside: Laguna M.1",
		"context_length": 262144,
		"max_completion_tokens": 32768,
		"supported_features": ["tools", "reasoning"]
	}`)
	entry, ok := entryFromOpenAICompatJSON(raw)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if entry.ID != "poolside/laguna-m.1" {
		t.Fatalf("id = %q", entry.ID)
	}
	if entry.OwnedBy != "poolside" {
		t.Fatalf("owned_by = %q", entry.OwnedBy)
	}
	if entry.DisplayName != "Poolside: Laguna M.1" {
		t.Fatalf("display_name = %q", entry.DisplayName)
	}
	if entry.ContextWindow != 262144 {
		t.Fatalf("context_window = %d", entry.ContextWindow)
	}
	if entry.MaxOutput != 32768 {
		t.Fatalf("max_output = %d", entry.MaxOutput)
	}
	if len(entry.Features) != 2 || entry.Features[0] != "tools" || entry.Features[1] != "reasoning" {
		t.Fatalf("features = %v, want Poolside supported_features", entry.Features)
	}
}
