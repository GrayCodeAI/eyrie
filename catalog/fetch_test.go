package catalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchModelCatalog_MockOpenRouter(t *testing.T) {
	// Create a mock OpenRouter server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		ctx := 200000
		maxComp := 32000
		resp := struct {
			Data []openRouterModel `json:"data"`
		}{
			Data: []openRouterModel{
				{
					ID:            "anthropic/claude-sonnet-4-6",
					ContextLength: &ctx,
					TopProvider: &struct {
						ContextLength       *int `json:"context_length"`
						MaxCompletionTokens *int `json:"max_completion_tokens"`
					}{MaxCompletionTokens: &maxComp},
					Pricing: &struct {
						Prompt     interface{} `json:"prompt"`
						Completion interface{} `json:"completion"`
					}{Prompt: "0.000003", Completion: "0.000015"},
				},
				{
					ID:            "openai/gpt-4o",
					ContextLength: &ctx,
					TopProvider: &struct {
						ContextLength       *int `json:"context_length"`
						MaxCompletionTokens *int `json:"max_completion_tokens"`
					}{MaxCompletionTokens: &maxComp},
					Pricing: &struct {
						Prompt     interface{} `json:"prompt"`
						Completion interface{} `json:"completion"`
					}{Prompt: 0.000005, Completion: 0.000015},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	env := map[string]string{
		"OPENROUTER_API_KEY":  "test-key-12345",
		"OPENROUTER_BASE_URL": srv.URL,
	}

	entries, err := fetchOpenRouterCatalog(env)
	if err != nil {
		t.Fatalf("fetchOpenRouterCatalog failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "anthropic/claude-sonnet-4-6" {
		t.Errorf("expected first entry ID 'anthropic/claude-sonnet-4-6', got %q", entries[0].ID)
	}
	if entries[0].ContextWindow != 200000 {
		t.Errorf("expected context window 200000, got %d", entries[0].ContextWindow)
	}
	if entries[0].MaxOutput != 32000 {
		t.Errorf("expected max output 32000, got %d", entries[0].MaxOutput)
	}
	// String pricing: "0.000003" * 1_000_000 = 3.0
	if entries[0].InputPricePer1M != 3.0 {
		t.Errorf("expected input price 3.0, got %f", entries[0].InputPricePer1M)
	}
	// Float pricing: 0.000005 * 1_000_000 = 5.0
	if entries[1].InputPricePer1M != 5.0 {
		t.Errorf("expected input price 5.0 for gpt-4o, got %f", entries[1].InputPricePer1M)
	}
}

func TestFetchModelCatalog_MockCanopyWave(t *testing.T) {
	ctx := 128000
	maxComp := 8192
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		resp := struct {
			Data []openAICompatModel `json:"data"`
		}{
			Data: []openAICompatModel{
				{
					ID:                  "zai/glm-4.6",
					ContextLength:       &ctx,
					MaxCompletionTokens: &maxComp,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	env := map[string]string{
		"CANOPYWAVE_API_KEY":  "test-key-12345",
		"CANOPYWAVE_BASE_URL": srv.URL,
	}

	entries, err := fetchCanopyWaveCatalog(env)
	if err != nil {
		t.Fatalf("fetchCanopyWaveCatalog failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "zai/glm-4.6" {
		t.Errorf("expected ID 'zai/glm-4.6', got %q", entries[0].ID)
	}
	if entries[0].ContextWindow != 128000 {
		t.Errorf("expected context window 128000, got %d", entries[0].ContextWindow)
	}
	if entries[0].MaxOutput != 8192 {
		t.Errorf("expected max output 8192, got %d", entries[0].MaxOutput)
	}
}

func TestFetchModelCatalog_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	env := map[string]string{
		"OPENROUTER_API_KEY":  "test-key-12345",
		"OPENROUTER_BASE_URL": srv.URL,
	}

	entries, err := fetchOpenRouterCatalog(env)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if entries != nil {
		t.Errorf("expected nil entries on error, got %v", entries)
	}
}

func TestFetchModelCatalog_NoAPIKey(t *testing.T) {
	env := map[string]string{}

	entries, err := fetchOpenRouterCatalog(env)
	if err != nil {
		t.Fatalf("expected no error with empty key, got %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries with no API key, got %v", entries)
	}

	entries, err = fetchCanopyWaveCatalog(env)
	if err != nil {
		t.Fatalf("expected no error with empty key, got %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries with no API key, got %v", entries)
	}
}

func TestFetchModelCatalog_CacheFileWritten(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "catalog_cache.json")

	// FetchModelCatalog with no API keys still writes the embedded catalog to cache
	env := map[string]string{}
	cat, err := FetchModelCatalog(cachePath, env)
	if err != nil {
		t.Fatalf("FetchModelCatalog failed: %v", err)
	}

	// Verify the cache file exists
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("expected cache file to be written, got error: %v", err)
	}

	// Verify it's valid JSON and contains expected data
	var cached ModelCatalog
	if err := json.Unmarshal(data, &cached); err != nil {
		t.Fatalf("cache file contains invalid JSON: %v", err)
	}
	if cached.Providers == nil {
		t.Fatal("cached catalog has nil providers")
	}
	if len(cached.Providers["anthropic"]) == 0 {
		t.Error("cached catalog missing anthropic models")
	}

	// Verify returned catalog has providers
	if cat.Providers == nil || len(cat.Providers["anthropic"]) == 0 {
		t.Error("returned catalog missing anthropic models")
	}
}

func TestFetchModelCatalog_EmptyCachePath(t *testing.T) {
	env := map[string]string{}
	cat, err := FetchModelCatalog("", env)
	if err != nil {
		t.Fatalf("FetchModelCatalog with empty cache path failed: %v", err)
	}
	if cat.Providers == nil {
		t.Error("expected non-nil providers")
	}
}

func TestLoadModelCatalogSync_ValidCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	cat := ModelCatalog{
		UpdatedAt: "2026-01-01T00:00:00Z",
		Source:    "test",
		Providers: map[string][]ModelCatalogEntry{
			"test_provider": {{ID: "test-model", ContextWindow: 100000, MaxOutput: 8000}},
		},
	}
	data, _ := json.Marshal(cat)
	_ = os.WriteFile(cachePath, data, 0o644)

	loaded := LoadModelCatalogSync(cachePath)
	if loaded.Source != "test" {
		t.Errorf("expected source 'test', got %q", loaded.Source)
	}
	if len(loaded.Providers["test_provider"]) != 1 {
		t.Error("expected 1 model in test_provider")
	}
}

func TestLoadModelCatalogSync_InvalidCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	_ = os.WriteFile(cachePath, []byte("invalid json!!!"), 0o644)

	loaded := LoadModelCatalogSync(cachePath)
	// Should fall back to default
	if loaded.Source != "embedded" {
		t.Errorf("expected fallback to embedded, got source %q", loaded.Source)
	}
}

func TestLoadModelCatalogSync_MissingFile(t *testing.T) {
	loaded := LoadModelCatalogSync("/nonexistent/path/cache.json")
	if loaded.Source != "embedded" {
		t.Errorf("expected fallback to embedded, got source %q", loaded.Source)
	}
}

func TestFetchOpenRouterCatalog_EmptyModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Data []openRouterModel `json:"data"`
		}{
			Data: []openRouterModel{
				{ID: ""},            // empty ID should be skipped
				{ID: "valid-model"}, // valid entry
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	env := map[string]string{
		"OPENROUTER_API_KEY":  "test-key-12345",
		"OPENROUTER_BASE_URL": srv.URL,
	}

	entries, err := fetchOpenRouterCatalog(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (empty ID skipped), got %d", len(entries))
	}
	if entries[0].ID != "valid-model" {
		t.Errorf("expected 'valid-model', got %q", entries[0].ID)
	}
}
