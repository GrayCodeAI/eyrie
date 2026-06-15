package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchZAI_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-zai-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"zai_payg/model-a","display_name":"ZAI Model A","status":1}`),
				json.RawMessage(`{"id":"zai_payg/model-b","display_name":"ZAI Model B","status":1}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchZAI(map[string]string{
		"ZAI_API_KEY":  "test-zai-key",
		"ZAI_BASE_URL": srv.URL,
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
	a, ok := byID["zai_payg/model-a"]
	if !ok {
		t.Fatal("missing zai_payg/model-a")
	}
	if a.DisplayName != "ZAI Model A" {
		t.Fatalf("display name = %q", a.DisplayName)
	}
	if len(a.RawJSON) == 0 {
		t.Fatal("expected RawJSON to be preserved")
	}
}

func TestFetchZAI_NoKey(t *testing.T) {
	entries, err := FetchZAI(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries with no key, got %d", len(entries))
	}
}

func TestFetchZAI_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer srv.Close()

	_, err := FetchZAI(map[string]string{
		"ZAI_API_KEY":  "bad-key",
		"ZAI_BASE_URL": srv.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestFetchZAICoding_MockHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-coding-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			Data []json.RawMessage `json:"data"`
		}{
			Data: []json.RawMessage{
				json.RawMessage(`{"id":"zai_coding/model-c","display_name":"ZAI Coding Model C","status":1}`),
				json.RawMessage(`{"id":"zai_coding/model-d","display_name":"ZAI Coding Model D","status":1}`),
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	entries, err := FetchZAICoding(map[string]string{
		"ZAI_CODING_API_KEY":  "test-coding-key",
		"ZAI_CODING_BASE_URL": srv.URL,
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
	c, ok := byID["zai_coding/model-c"]
	if !ok {
		t.Fatal("missing zai_coding/model-c")
	}
	if c.DisplayName != "ZAI Coding Model C" {
		t.Fatalf("display name = %q", c.DisplayName)
	}
	if len(c.RawJSON) == 0 {
		t.Fatal("expected RawJSON to be preserved")
	}
}
