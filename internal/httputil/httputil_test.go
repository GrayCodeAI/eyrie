package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConstantTimeEqual(t *testing.T) {
	key := "super-secret-api-key"
	if !ConstantTimeEqual(key, key) {
		t.Fatal("expected equal tokens to match")
	}
	if ConstantTimeEqual(key, key+"x") {
		t.Fatal("expected different-length tokens to not match")
	}
	if ConstantTimeEqual("short", "much-longer-token") {
		t.Fatal("expected mismatched tokens to not match")
	}
	if !ConstantTimeEqual("", "") {
		t.Fatal("expected empty strings to match")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.1.2.3", true},
		{"::1", true},
		{"localhost", true},
		{"", false},
		{"0.0.0.0", false},
		{"10.0.0.1", false},
		{"example.com", false},
	}
	for _, tc := range cases {
		if got := IsLoopbackHost(tc.host); got != tc.want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestValidateAuthConfig(t *testing.T) {
	if err := ValidateAuthConfig("127.0.0.1:8080", ""); err != nil {
		t.Errorf("loopback with no key should be allowed: %v", err)
	}
	if err := ValidateAuthConfig("0.0.0.0:8080", "secret"); err != nil {
		t.Errorf("non-loopback with key should be allowed: %v", err)
	}
	if err := ValidateAuthConfig("0.0.0.0:8080", ""); err == nil {
		t.Error("non-loopback with no key should be rejected")
	}
}

func TestExtractBearerToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	if got := ExtractBearerToken(req); got != "my-token" {
		t.Errorf("got %q, want %q", got, "my-token")
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-API-Key", "api-key-token")
	if got := ExtractBearerToken(req2); got != "api-key-token" {
		t.Errorf("got %q, want %q", got, "api-key-token")
	}

	req3 := httptest.NewRequest("GET", "/", nil)
	if got := ExtractBearerToken(req3); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want %q", ct, "application/json")
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body = %q, want to contain status:ok", body)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("missing Cache-Control")
	}
}
