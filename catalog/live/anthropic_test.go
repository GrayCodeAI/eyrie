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
	if len(entries) != 17 {
		t.Fatalf("expected 17 models, got %d", len(entries))
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	sonnet, ok := byID["claude-sonnet-4-20250514"]
	if !ok {
		t.Fatal("missing claude-sonnet-4-20250514")
	}
	if sonnet.DisplayName != "Claude Sonnet 4 (deprecated)" {
		t.Fatalf("display name = %q", sonnet.DisplayName)
	}
	if sonnet.MaxInputTokens != 200000 {
		t.Fatalf("max_input_tokens = %d (expected 200000)", sonnet.MaxInputTokens)
	}
	if sonnet.MaxOutput != 64000 {
		t.Fatalf("max_output = %d (expected 64000)", sonnet.MaxOutput)
	}
	if !sonnet.ThinkingEnabled {
		t.Fatal("expected thinking_enabled = true")
	}
	if !sonnet.ThinkingAdaptive {
		t.Fatal("expected thinking_adaptive = true")
	}
	if sonnet.EffortSupported {
		t.Fatal("expected effort_supported = false for sonnet-4")
	}
	if !sonnet.StructuredOutput {
		t.Fatal("expected structured_output = true")
	}
	if !sonnet.CodeExecution {
		t.Fatal("expected code_execution = true")
	}
	if !sonnet.CitationsSupported {
		t.Fatal("expected citations_supported = true")
	}
	if !sonnet.PDFInput {
		t.Fatal("expected pdf_input = true")
	}
	if !sonnet.ImageInput {
		t.Fatal("expected image_input = true")
	}
	if len(sonnet.RawJSON) == 0 {
		t.Fatal("expected RawJSON to be preserved")
	}
	// Check Features list
	featureSet := map[string]bool{}
	for _, f := range sonnet.Features {
		featureSet[f] = true
	}
	for _, want := range []string{"thinking:enabled", "thinking:adaptive", "structured_output", "code_execution", "citations", "pdf_input", "image_input"} {
		if !featureSet[want] {
			t.Fatalf("expected feature %q in Features list", want)
		}
	}

	haiku, ok := byID["claude-haiku-4-5-20251001"]
	if !ok {
		t.Fatal("missing claude-haiku-4-5-20251001")
	}
	if haiku.MaxInputTokens != 200000 {
		t.Fatalf("haiku max_input_tokens = %d (expected 200000)", haiku.MaxInputTokens)
	}
	if haiku.CodeExecution {
		t.Fatal("expected haiku code_execution = false")
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
