package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFetchOpenRouter_Mock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{json.RawMessage(`{
				"id":"anthropic/claude-sonnet-4-6",
				"context_length":200000,
				"top_provider":{"max_completion_tokens":32000},
				"architecture":{"modality":"text"}
			}`)},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchOpenRouter(map[string]string{
		"OPENROUTER_API_KEY":  "test-key-12345",
		"OPENROUTER_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].RawJSON) == 0 {
		t.Fatal("expected full provider JSON preserved")
	}
	if !strings.Contains(string(entries[0].RawJSON), "architecture") {
		t.Fatalf("expected raw metadata, got %s", entries[0].RawJSON)
	}
}

func TestFetchCanopyWave_ParsesProviderFields(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "vendor/sample",
		"display_name": "Sample Model",
		"description": "Synthetic parser test",
		"context_size": 2048,
		"status": 1,
		"max_output_tokens": 512,
		"input_token_price_per_m": 11,
		"output_token_price_per_m": 22,
		"features": ["function-calling","vision"],
		"tags": ["synthetic-test-only"]
	}`)
	entry, ok := entryFromOpenAICompatJSON(raw)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if entry.ID != "vendor/sample" {
		t.Fatalf("id = %q", entry.ID)
	}
	if entry.DisplayName != "Sample Model" {
		t.Fatalf("display = %q", entry.DisplayName)
	}
	if entry.ContextWindow != 2048 || entry.MaxOutput != 512 {
		t.Fatalf("context/max = %d/%d", entry.ContextWindow, entry.MaxOutput)
	}
	if entry.InputPricePer1M != 11 || entry.OutputPricePer1M != 22 {
		t.Fatalf("pricing = %v/%v", entry.InputPricePer1M, entry.OutputPricePer1M)
	}
	if len(entry.Features) != 2 {
		t.Fatalf("features = %v", entry.Features)
	}
	if !strings.Contains(string(entry.RawJSON), "synthetic-test-only") {
		t.Fatalf("raw json not preserved: %s", entry.RawJSON)
	}
}

func TestFetchCanopyWave_MockHTTPServer(t *testing.T) {
	body, err := os.ReadFile("testdata/canopywave_models.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	entries, err := FetchCanopyWave(map[string]string{
		"CANOPYWAVE_API_KEY":  "test-key-12345678",
		"CANOPYWAVE_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 models, got %d", len(entries))
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}

	// Verify pricing conversion: API returns cents, we convert to dollars.
	// GLM-5.1: 140 cents = $1.40, 440 cents = $4.40
	glm, ok := byID["zai/glm-5.1"]
	if !ok {
		t.Fatal("missing zai/glm-5.1")
	}
	if glm.InputPricePer1M != 1.40 {
		t.Errorf("glm-5.1 input price = %.2f, want 1.40", glm.InputPricePer1M)
	}
	if glm.OutputPricePer1M != 4.40 {
		t.Errorf("glm-5.1 output price = %.2f, want 4.40", glm.OutputPricePer1M)
	}
	if glm.ContextWindow != 2048000 {
		t.Errorf("glm-5.1 context = %d, want 2048000", glm.ContextWindow)
	}
	if glm.MaxOutput != 8192 {
		t.Errorf("glm-5.1 max output = %d, want 8192", glm.MaxOutput)
	}

	// DeepSeek-V4-Flash: 14 cents = $0.14, 28 cents = $0.28
	ds, ok := byID["deepseek/deepseek-v4-flash"]
	if !ok {
		t.Fatal("missing deepseek/deepseek-v4-flash")
	}
	if ds.InputPricePer1M != 0.14 {
		t.Errorf("deepseek input price = %.2f, want 0.14", ds.InputPricePer1M)
	}
	if ds.OutputPricePer1M != 0.28 {
		t.Errorf("deepseek output price = %.2f, want 0.28", ds.OutputPricePer1M)
	}

	for _, e := range entries {
		if len(e.RawJSON) == 0 {
			t.Fatalf("%s missing raw json", e.ID)
		}
	}
}

func TestFetchOllama_EmptyModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	}))
	defer srv.Close()

	entries, err := FetchOllama(map[string]string{"OLLAMA_BASE_URL": srv.URL + "/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
