package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchGroq_MockHTTPServer(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("testdata/groq_models.json")
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

	entries, err := FetchGroq(map[string]string{
		"GROQ_API_KEY":  "gsk-test123",
		"GROQ_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 models, got %d", len(entries))
	}
}

func TestFetchGroq_NoKey(t *testing.T) {
	t.Parallel()
	entries, err := FetchGroq(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchGroq_ParsesProviderFields(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{
		"id": "llama-3.3-70b-versatile",
		"owned_by": "meta-llama",
		"display_name": "Llama 3.3 70B Versatile"
	}`)
	entry, ok := entryFromOpenAICompatJSON(raw)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if entry.ID != "llama-3.3-70b-versatile" {
		t.Fatalf("id = %q", entry.ID)
	}
	if entry.OwnedBy != "meta-llama" {
		t.Fatalf("owned_by = %q", entry.OwnedBy)
	}
	if entry.DisplayName != "Llama 3.3 70B Versatile" {
		t.Fatalf("display_name = %q", entry.DisplayName)
	}
}
