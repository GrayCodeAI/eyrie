package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchOpenAI_MockHTTPServer(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("testdata/openai_models.json")
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

	entries, err := FetchOpenAI(map[string]string{
		"OPENAI_API_KEY":  "sk-proj-test123",
		"OPENAI_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 models, got %d", len(entries))
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	gpt4o, ok := byID["gpt-4o"]
	if !ok {
		t.Fatal("missing gpt-4o")
	}
	if gpt4o.OwnedBy != "openai" {
		t.Fatalf("owned_by = %q", gpt4o.OwnedBy)
	}
}

func TestFetchOpenAI_NoKey(t *testing.T) {
	t.Parallel()
	entries, err := FetchOpenAI(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchOpenAI_Unauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	_, err := FetchOpenAI(map[string]string{
		"OPENAI_API_KEY":  "sk-bad",
		"OPENAI_BASE_URL": srv.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}
