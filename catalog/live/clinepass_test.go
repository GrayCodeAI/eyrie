package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchClinePass_MockHTTPServer(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("testdata/clinepass_models.json")
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

	entries, err := FetchClinePass(map[string]string{
		"CLINE_API_KEY":  "cp-test123",
		"CLINE_API_BASE": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 models, got %d", len(entries))
	}
}

func TestFetchClinePass_NoKey(t *testing.T) {
	t.Parallel()
	entries, err := FetchClinePass(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchClinePass_ParsesProviderFields(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{
		"id": "cline-pass/deepseek-v4-pro",
		"owned_by": "deepseek",
		"display_name": "DeepSeek V4 Pro"
	}`)
	entry, ok := entryFromOpenAICompatJSON(raw)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if entry.ID != "cline-pass/deepseek-v4-pro" {
		t.Fatalf("id = %q", entry.ID)
	}
	if entry.OwnedBy != "deepseek" {
		t.Fatalf("owned_by = %q", entry.OwnedBy)
	}
	if entry.DisplayName != "DeepSeek V4 Pro" {
		t.Fatalf("display_name = %q", entry.DisplayName)
	}
}
