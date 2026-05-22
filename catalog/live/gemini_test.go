package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFetchGemini_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/gemini_models.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1beta/models") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	entries, err := FetchGemini(map[string]string{
		"GEMINI_API_KEY":  "AIzaTest123",
		"GEMINI_BASE_URL": srv.URL,
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
	flash, ok := byID["gemini-2.0-flash"]
	if !ok {
		t.Fatal("missing gemini-2.0-flash")
	}
	if flash.DisplayName != "Gemini 2.0 Flash" {
		t.Fatalf("display name = %q", flash.DisplayName)
	}
	if flash.ContextWindow != 128000 || flash.MaxOutput != 8192 {
		t.Fatalf("context/max = %d/%d", flash.ContextWindow, flash.MaxOutput)
	}
}

func TestFetchGemini_NoKey(t *testing.T) {
	entries, err := FetchGemini(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchGemini_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	_, err := FetchGemini(map[string]string{
		"GEMINI_API_KEY":  "AIzaBad",
		"GEMINI_BASE_URL": srv.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestFetchGemini_FiltersNonGenerateModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"models": []map[string]any{
				{
					"name":                       "models/gemini-embedding-exp-03-07",
					"displayName":                "Gemini Embedding Experimental",
					"supportedGenerationMethods": []string{"embedContent"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchGemini(map[string]string{
		"GEMINI_API_KEY":  "AIzaTest",
		"GEMINI_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for non-generateContent models, got %d", len(entries))
	}
}
