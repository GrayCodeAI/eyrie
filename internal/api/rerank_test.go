//nolint:bodyclose,noctx
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRerankLexicalFallback(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	body := `{"query":"how do cats purr","documents":[` +
		`"the stock market closed lower today",` +
		`"cats purr when they are content and cats relax",` +
		`"a short note about purr in cats"` +
		`]}`
	resp, err := http.Post(ts.URL+"/rerank", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var out rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("results = %d, want 3", len(out.Results))
	}
	// The unrelated stock-market document (index 0) should rank last.
	if out.Results[len(out.Results)-1].Index != 0 {
		t.Errorf("expected doc 0 ranked last, got order %+v", out.Results)
	}
	// Scores must be sorted descending.
	for i := 1; i < len(out.Results); i++ {
		if out.Results[i-1].RelevanceScore < out.Results[i].RelevanceScore {
			t.Errorf("results not sorted desc: %+v", out.Results)
		}
	}
}

func TestRerankTopN(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	body := `{"query":"alpha","documents":["alpha beta","alpha","gamma"],"top_n":2}`
	resp, err := http.Post(ts.URL+"/rerank", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("results = %d, want 2 (top_n)", len(out.Results))
	}
}

func TestRerankValidation(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	tests := []struct {
		name string
		body string
	}{
		{"missing query", `{"documents":["a"]}`},
		{"missing documents", `{"query":"q"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/rerank", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			defer drainBody(t, resp)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestReady(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ready: expected 200, got %d", resp.StatusCode)
	}
}

func TestReadyNotReady(t *testing.T) {
	// A server without a store/engine must report not-ready (503), while
	// /health (liveness) keeps reporting ok.
	s := &Server{mux: http.NewServeMux()}
	s.routes()

	readyRec := httptest.NewRecorder()
	s.mux.ServeHTTP(readyRec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if readyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready: expected 503 when deps missing, got %d", readyRec.Code)
	}

	healthRec := httptest.NewRecorder()
	s.mux.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health: expected 200 (liveness), got %d", healthRec.Code)
	}
}
