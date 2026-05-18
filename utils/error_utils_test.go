package utils

import (
	"errors"
	"testing"
)

type codedError struct {
	code string
	msg  string
}

func (e *codedError) Error() string { return e.msg }
func (e *codedError) Code() string  { return e.code }

func TestExtractConnectionErrorDetails(t *testing.T) {
	err := &codedError{code: "CERT_HAS_EXPIRED", msg: "certificate expired"}
	details := ExtractConnectionErrorDetails(err)
	if details == nil {
		t.Fatal("expected details")
	}
	if !details.IsSSLError {
		t.Error("expected SSL error")
	}
	if details.Code != "CERT_HAS_EXPIRED" {
		t.Errorf("expected CERT_HAS_EXPIRED, got %s", details.Code)
	}

	nonSSL := &codedError{code: "ECONNREFUSED", msg: "connection refused"}
	details = ExtractConnectionErrorDetails(nonSSL)
	if details == nil {
		t.Fatal("expected details")
	}
	if details.IsSSLError {
		t.Error("expected non-SSL error")
	}
}

func TestExtractConnectionErrorDetailsNil(t *testing.T) {
	if ExtractConnectionErrorDetails(nil) != nil {
		t.Error("expected nil for nil error")
	}
	if ExtractConnectionErrorDetails(errors.New("plain error")) != nil {
		t.Error("expected nil for plain error without Code()")
	}
}

func TestGetSSLErrorHint(t *testing.T) {
	err := &codedError{code: "DEPTH_ZERO_SELF_SIGNED_CERT", msg: "self signed"}
	hint := GetSSLErrorHint(err)
	if hint == nil {
		t.Fatal("expected hint")
	}
	if *hint == "" {
		t.Error("expected non-empty hint")
	}

	nonSSL := errors.New("plain error")
	if GetSSLErrorHint(nonSSL) != nil {
		t.Error("expected nil hint for non-SSL error")
	}
}

func TestSanitizeAPIError(t *testing.T) {
	if got := SanitizeAPIError("plain error"); got != "plain error" {
		t.Errorf("expected plain error, got %q", got)
	}
	html := `<!DOCTYPE html><html><head><title>502 Bad Gateway</title></head><body>error</body></html>`
	if got := SanitizeAPIError(html); got != "502 Bad Gateway" {
		t.Errorf("expected '502 Bad Gateway', got %q", got)
	}
	if got := SanitizeAPIError(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
