package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func body(s string) io.ReadCloser { return io.NopCloser(strings.NewReader(s)) }

func TestParseProviderError(t *testing.T) {
	d := parseProviderError(body(`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`))
	if d.Message != "Incorrect API key provided" {
		t.Errorf("Message = %q", d.Message)
	}
	if d.Type != "invalid_request_error" {
		t.Errorf("Type = %q", d.Type)
	}
	if d.Code != "invalid_api_key" {
		t.Errorf("Code = %q", d.Code)
	}
}

func TestParseProviderError_NumericCode(t *testing.T) {
	// Some providers send a bare numeric code.
	d := parseProviderError(body(`{"error":{"message":"nope","code":404}}`))
	if d.Code != "404" {
		t.Errorf("Code = %q, want 404", d.Code)
	}
}

func TestParseProviderError_Unstructured(t *testing.T) {
	d := parseProviderError(body(`upstream proxy: 502 bad gateway`))
	if d.Message != "" {
		t.Errorf("Message should be empty for unstructured body, got %q", d.Message)
	}
	if !strings.Contains(d.Raw, "bad gateway") {
		t.Errorf("Raw = %q", d.Raw)
	}
}

func TestClassifyProviderError(t *testing.T) {
	cases := []struct {
		name   string
		status int
		detail providerErrorDetail
		want   string // substring expected in the hint
	}{
		{"invalid key by code", 400, providerErrorDetail{Code: "invalid_api_key"}, "invalid API key"},
		{"model not found by code", 404, providerErrorDetail{Code: "model_not_found"}, "model not found"},
		{"quota by message", 429, providerErrorDetail{Message: "You exceeded your current quota"}, "billing/quota"},
		{"content filter", 400, providerErrorDetail{Type: "content_filter"}, "content filter"},
		{"context length", 400, providerErrorDetail{Message: "maximum context length is 8192"}, "context window"},
		{"bare 401", http.StatusUnauthorized, providerErrorDetail{}, "unauthorized"},
		{"bare 429", http.StatusTooManyRequests, providerErrorDetail{}, "rate limited"},
		{"bare 503", http.StatusServiceUnavailable, providerErrorDetail{}, "transient"},
		{"no hint", 400, providerErrorDetail{Message: "weird"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyProviderError(tc.status, tc.detail)
			if tc.want == "" {
				if got != "" {
					t.Errorf("expected no hint, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("hint = %q, want substring %q", got, tc.want)
			}
		})
	}
}

func TestFormatAPIError(t *testing.T) {
	err := formatAPIError("openai", 401, "req_123",
		providerErrorDetail{Message: "bad key", Type: "auth_error", Code: "invalid_api_key"})
	s := err.Error()
	for _, want := range []string{"openai", "status=401", "request_id=req_123", "invalid API key", "bad key"} {
		if !strings.Contains(s, want) {
			t.Errorf("error %q missing %q", s, want)
		}
	}
}

func TestFormatAPIError_NoHintFallsBackToDetail(t *testing.T) {
	err := formatAPIError("vertex", 400, "", providerErrorDetail{Raw: "totally opaque"})
	s := err.Error()
	if !strings.Contains(s, "totally opaque") {
		t.Errorf("error %q should include raw detail", s)
	}
	if !strings.Contains(s, "request_id=") {
		t.Errorf("error %q should still include request_id field", s)
	}
}
