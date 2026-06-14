package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchOpenCodeGo_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-ocg-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"kimi-k2.6","owned_by":"opencode"}`),
				json.RawMessage(`{"id":"minimax-m2.7","owned_by":"opencode"}`),
				json.RawMessage(`{"id":"qwen3.7-max","owned_by":"opencode"}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchOpenCodeGo(map[string]string{
		"OPENCODEGO_API_KEY":  "test-ocg-key",
		"OPENCODEGO_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected all 3 models, got %d", len(entries))
	}
	if entries[0].ID != "kimi-k2.6" {
		t.Fatalf("id = %q", entries[0].ID)
	}
	// Verify cached pricing.erify protocol is derived from model name heuristic.
	if entries[0].Protocol != "openai" {
		t.Errorf("kimi-k2.6 protocol = %q, want openai", entries[0].Protocol)
	}
	if entries[1].Protocol != "anthropic" {
		t.Errorf("minimax-m2.7 protocol = %q, want anthropic", entries[1].Protocol)
	}
	if entries[2].Protocol != "anthropic" {
		t.Errorf("qwen3.7-max protocol = %q, want anthropic", entries[2].Protocol)
	}
}

func TestFetchOpenCodeGo_WithAPIProtocolMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				// Verify cached pricing.imulate API returning api_type field (future-proof).
				json.RawMessage(`{"id":"new-model-x","owned_by":"opencode","api_type":"anthropic"}`),
				json.RawMessage(`{"id":"kimi-k2.6","owned_by":"opencode","features":["openai-compat"]}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchOpenCodeGo(map[string]string{
		"OPENCODEGO_API_KEY":  "test-ocg-key",
		"OPENCODEGO_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 models, got %d", len(entries))
	}
	// Verify cached pricing.pi_type should take precedence.
	if entries[0].Protocol != "anthropic" {
		t.Errorf("new-model-x protocol = %q, want anthropic (from api_type)", entries[0].Protocol)
	}
	// Verify cached pricing.eatures hint should work.
	if entries[1].Protocol != "openai" {
		t.Errorf("kimi-k2.6 protocol = %q, want openai (from features)", entries[1].Protocol)
	}
}

func TestFetchOpenCodeGo_ActualAPIFormat(t *testing.T) {
	// Verify cached pricing.est with the actual API response format (minimal fields only).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"minimax-m3","object":"model","created":1781385301,"owned_by":"opencode"}`),
				json.RawMessage(`{"id":"kimi-k2.6","object":"model","created":1781385301,"owned_by":"opencode"}`),
				json.RawMessage(`{"id":"qwen3.7-max","object":"model","created":1781385301,"owned_by":"opencode"}`),
				json.RawMessage(`{"id":"mimo-v2.5-pro","object":"model","created":1781385301,"owned_by":"opencode"}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchOpenCodeGo(map[string]string{
		"OPENCODEGO_API_KEY":  "test-ocg-key",
		"OPENCODEGO_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 models, got %d", len(entries))
	}
	// Verify cached pricing.erify protocol derived from heuristic (no api_type in response).
	if entries[0].Protocol != "anthropic" {
		t.Errorf("minimax-m3 protocol = %q, want anthropic", entries[0].Protocol)
	}
	if entries[1].Protocol != "openai" {
		t.Errorf("kimi-k2.6 protocol = %q, want openai", entries[1].Protocol)
	}
	if entries[2].Protocol != "anthropic" {
		t.Errorf("qwen3.7-max protocol = %q, want anthropic", entries[2].Protocol)
	}
	if entries[3].Protocol != "openai" {
		t.Errorf("mimo-v2.5-pro protocol = %q, want openai", entries[3].Protocol)
	}
}

func TestFetchOpenCodeGo_NoKey(t *testing.T) {
	entries, err := FetchOpenCodeGo(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchOpenCodeGo_FullActualResponse(t *testing.T) {
	// Verify cached pricing.xact response from https://opencode.ai/zen/go/v1/models (2026-06-14).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[
			{"id":"minimax-m3","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"minimax-m2.7","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"minimax-m2.5","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"kimi-k2.7-code","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"kimi-k2.6","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"kimi-k2.5","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"glm-5.1","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"glm-5","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"deepseek-v4-pro","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"deepseek-v4-flash","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"qwen3.7-max","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"qwen3.7-plus","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"qwen3.6-plus","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"qwen3.5-plus","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"mimo-v2-pro","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"mimo-v2-omni","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"mimo-v2.5-pro","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"mimo-v2.5","object":"model","created":1781385301,"owned_by":"opencode"},
			{"id":"hy3-preview","object":"model","created":1781385301,"owned_by":"opencode"}
		]}`))
	}))
	defer srv.Close()

	entries, err := FetchOpenCodeGo(map[string]string{
		"OPENCODEGO_API_KEY":  "test-ocg-key",
		"OPENCODEGO_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 19 {
		t.Fatalf("expected 19 models, got %d", len(entries))
	}

	// Verify cached pricing.uild lookup map.
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}

	// Verify cached pricing.erify protocol routing (from static metadata, not heuristic).
	anthropicModels := []string{"minimax-m3", "minimax-m2.7", "minimax-m2.5", "qwen3.7-max", "qwen3.7-plus", "qwen3.6-plus", "qwen3.5-plus"}
	for _, id := range anthropicModels {
		if e, ok := byID[id]; !ok {
			t.Errorf("missing model %s", id)
		} else if e.Protocol != "anthropic" {
			t.Errorf("%s protocol = %q, want anthropic", id, e.Protocol)
		}
	}
	openaiModels := []string{"kimi-k2.6", "kimi-k2.7-code", "kimi-k2.5", "glm-5.1", "glm-5", "deepseek-v4-pro", "deepseek-v4-flash", "mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-pro", "mimo-v2-omni", "hy3-preview"}
	for _, id := range openaiModels {
		if e, ok := byID[id]; !ok {
			t.Errorf("missing model %s", id)
		} else if e.Protocol != "openai" {
			t.Errorf("%s protocol = %q, want openai", id, e.Protocol)
		}
	}

	// Verify cached pricing.erify pricing is populated.
	if e := byID["glm-5.1"]; e.InputPricePer1M <= 0 {
		t.Errorf("glm-5.1 input price = %.2f, want >0", e.InputPricePer1M)
	}
	if e := byID["deepseek-v4-flash"]; e.InputPricePer1M <= 0 {
		t.Errorf("deepseek-v4-flash input price = %.2f, want >0", e.InputPricePer1M)
	}
	if e := byID["minimax-m3"]; e.OutputPricePer1M <= 0 {
		t.Errorf("minimax-m3 output price = %.2f, want >0", e.OutputPricePer1M)
	}

	// Verify cached pricing.
	if e := byID["glm-5.1"]; e.CachedReadPricePer1M <= 0 {
		t.Errorf("glm-5.1 cached read = %.3f, want >0", e.CachedReadPricePer1M)
	}

	// Verify cached pricing.erify context window.
	if e := byID["kimi-k2.6"]; e.ContextWindow <= 0 {
		t.Errorf("kimi-k2.6 context = %d, want >0", e.ContextWindow)
	}

	// Verify cached pricing.erify ID normalization (no prefixes).
	for _, e := range entries {
		if e.ID != strings.TrimSpace(e.ID) {
			t.Errorf("ID %q has leading/trailing whitespace", e.ID)
		}
	}
}

func TestFetchOpenCodeGo_UnknownModelStillWorks(t *testing.T) {
	// Verify cached pricing. brand new model not in static metadata should still work.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"object":"list","data":[
			{"id":"brand-new-model","object":"model","created":1781385301,"owned_by":"opencode"}
		]}`))
	}))
	defer srv.Close()

	entries, err := FetchOpenCodeGo(map[string]string{
		"OPENCODEGO_API_KEY":  "test-ocg-key",
		"OPENCODEGO_BASE_URL": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 model, got %d", len(entries))
	}
	// Verify cached pricing.nknown model — protocol derived from heuristic (defaults to openai).
	if entries[0].Protocol != "openai" {
		t.Errorf("unknown model protocol = %q, want openai", entries[0].Protocol)
	}
	// Verify cached pricing.ricing should be zero (not in static metadata).
	if entries[0].InputPricePer1M != 0 {
		t.Errorf("unknown model input price = %.2f, want 0", entries[0].InputPricePer1M)
	}
}
