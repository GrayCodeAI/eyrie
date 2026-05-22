package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchAnthropic_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/anthropic_models.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	entries, err := FetchAnthropic(map[string]string{
		"ANTHROPIC_API_KEY":  "sk-ant-test123",
		"ANTHROPIC_BASE_URL": srv.URL,
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
	sonnet, ok := byID["claude-sonnet-4-20250514"]
	if !ok {
		t.Fatal("missing claude-sonnet-4-20250514")
	}
	if sonnet.DisplayName != "Claude Sonnet 4" {
		t.Fatalf("display name = %q", sonnet.DisplayName)
	}
	if sonnet.ContextWindow != 200000 || sonnet.MaxOutput != 8192 {
		t.Fatalf("context/max = %d/%d", sonnet.ContextWindow, sonnet.MaxOutput)
	}
}

func TestFetchAnthropic_NoKey(t *testing.T) {
	entries, err := FetchAnthropic(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchAnthropic_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	_, err := FetchAnthropic(map[string]string{
		"ANTHROPIC_API_KEY":  "sk-ant-bad",
		"ANTHROPIC_BASE_URL": srv.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}
