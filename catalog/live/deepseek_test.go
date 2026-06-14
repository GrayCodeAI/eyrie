package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchDeepSeek_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/deepseek_models.json")
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

	entries, err := FetchDeepSeek(map[string]string{
		"DEEPSEEK_API_KEY":  "sk-test12345678",
		"DEEPSEEK_BASE_URL": srv.URL,
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
	v4flash, ok := byID["deepseek-v4-flash"]
	if !ok {
		t.Fatal("missing deepseek-v4-flash")
	}
	if v4flash.OwnedBy != "deepseek" {
		t.Fatalf("owned_by = %q", v4flash.OwnedBy)
	}
	v4pro, ok := byID["deepseek-v4-pro"]
	if !ok {
		t.Fatal("missing deepseek-v4-pro")
	}
	if v4pro.OwnedBy != "deepseek" {
		t.Fatalf("owned_by = %q", v4pro.OwnedBy)
	}
	for _, e := range entries {
		if len(e.RawJSON) == 0 {
			t.Fatalf("%s missing raw json", e.ID)
		}
	}
}

func TestFetchDeepSeek_NoKey(t *testing.T) {
	entries, err := FetchDeepSeek(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchDeepSeek_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	_, err := FetchDeepSeek(map[string]string{
		"DEEPSEEK_API_KEY":  "sk-bad",
		"DEEPSEEK_BASE_URL": srv.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}
