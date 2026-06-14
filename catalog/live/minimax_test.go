package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchMiniMaxTokenPlan_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/minimax_token_plan_models.json")
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

	entries, err := FetchMiniMaxTokenPlan(map[string]string{
		"MINIMAX_TOKEN_PLAN_API_KEY": "test-key",
		"MINIMAX_TOKEN_PLAN_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 8 {
		t.Fatalf("expected 8 models, got %d", len(entries))
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	m3, ok := byID["MiniMax-M3"]
	if !ok {
		t.Fatal("missing MiniMax-M3")
	}
	if m3.DisplayName != "MiniMax-M3" {
		t.Fatalf("display name = %q", m3.DisplayName)
	}
	if m3.OwnedBy != "minimax" {
		t.Fatalf("owned_by = %q, want minimax", m3.OwnedBy)
	}
	if len(m3.RawJSON) == 0 {
		t.Fatal("expected RawJSON to be preserved")
	}
}

func TestFetchMiniMaxTokenPlan_NoKey(t *testing.T) {
	entries, err := FetchMiniMaxTokenPlan(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchMiniMaxTokenPlan_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	_, err := FetchMiniMaxTokenPlan(map[string]string{
		"MINIMAX_TOKEN_PLAN_API_KEY": "bad-key",
		"MINIMAX_TOKEN_PLAN_BASE_URL": srv.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestFetchMiniMaxPayg_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/minimax_payg_models.json")
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

	entries, err := FetchMiniMaxPayg(map[string]string{
		"MINIMAX_PAYG_API_KEY": "test-key",
		"MINIMAX_PAYG_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 8 {
		t.Fatalf("expected 8 models, got %d", len(entries))
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	m3, ok := byID["MiniMax-M3"]
	if !ok {
		t.Fatal("missing MiniMax-M3")
	}
	if m3.DisplayName != "MiniMax-M3" {
		t.Fatalf("display name = %q", m3.DisplayName)
	}
}

func TestFetchMiniMaxPayg_NoKey(t *testing.T) {
	entries, err := FetchMiniMaxPayg(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchMiniMaxPayg_NoGenericFallback(t *testing.T) {
	// Generic MINIMAX_API_KEY should NOT be used — only plan-specific keys
	entries, err := FetchMiniMaxPayg(map[string]string{
		"MINIMAX_API_KEY": "generic-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries without plan-specific key, got %d", len(entries))
	}
}
