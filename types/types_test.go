package types

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSessionIdAndAgentId(t *testing.T) {
	sid := AsSessionId("session-123")
	if string(sid) != "session-123" {
		t.Error("AsSessionId failed")
	}
	aid := AsAgentId("a1234567890abcdef")
	if string(aid) != "a1234567890abcdef" {
		t.Error("AsAgentId failed")
	}
}

func TestToAgentId(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"a1234567890abcdef", true},
		{"alabel-1234567890abcdef", true},
		{"invalid", false},
		{"", false},
		{"a123", false},
	}
	for _, tt := range tests {
		result, _ := ToAgentId(tt.input)
		if tt.valid && result == nil {
			t.Errorf("ToAgentId(%q) = nil, expected valid", tt.input)
		}
		if !tt.valid && result != nil {
			t.Errorf("ToAgentId(%q) = %v, expected nil", tt.input, result)
		}
	}
}

func TestIsTextBlock(t *testing.T) {
	tb := TextBlock{Type: "text", Text: "hello"}
	if b, ok := IsTextBlock(tb); !ok || b.Text != "hello" {
		t.Error("IsTextBlock failed for valid TextBlock")
	}
	if _, ok := IsTextBlock("not a block"); ok {
		t.Error("IsTextBlock should fail for string")
	}
}

func TestIsToolUseBlock(t *testing.T) {
	tub := ToolUseBlock{Type: "tool_use", ID: "1", Name: "test", Input: nil}
	if b, ok := IsToolUseBlock(tub); !ok || b.Name != "test" {
		t.Error("IsToolUseBlock failed")
	}
}

func TestCreateMessages(t *testing.T) {
	um := CreateUserMessage("hello")
	if um.Role != "user" || um.Content != "hello" {
		t.Error("CreateUserMessage failed")
	}
	am := CreateAssistantMessage("hi")
	if am.Role != "assistant" || am.Content != "hi" {
		t.Error("CreateAssistantMessage failed")
	}
	sm := CreateSystemMessage("sys")
	if sm.Role != "system" || sm.Content != "sys" {
		t.Error("CreateSystemMessage failed")
	}
}

func TestEmptyUsage(t *testing.T) {
	u := EmptyUsage()
	if u.InputTokens != 0 || u.OutputTokens != 0 {
		t.Error("EmptyUsage should have zero tokens")
	}
	if u.Iterations == nil {
		t.Error("expected non-nil iterations slice")
	}
}

func TestAPIError(t *testing.T) {
	err := NewAPIError(429, nil, nil, "rate limited")
	if err.Error() != "rate limited" {
		t.Errorf("expected 'rate limited', got %q", err.Error())
	}
	if err.Status != 429 {
		t.Errorf("expected status 429, got %d", err.Status)
	}

	connErr := NewAPIConnectionError("timeout")
	if connErr.Error() != "timeout" {
		t.Error("connection error message wrong")
	}
}

func TestIsConnectorTextBlock(t *testing.T) {
	valid := map[string]interface{}{"type": "connector_text", "text": "hello"}
	if !IsConnectorTextBlock(valid) {
		t.Error("expected true for valid connector text block")
	}
	invalid := map[string]interface{}{"type": "text", "text": "hello"}
	if IsConnectorTextBlock(invalid) {
		t.Error("expected false for non-connector block")
	}
}

// --- Retry logic tests ---

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		transient bool
	}{
		// nil
		{"nil error", nil, false},

		// Retriable HTTP status codes: 408, 429, 500, 502, 503, 504, 529
		{"408 Request Timeout", fmt.Errorf("HTTP 408 timeout"), true},
		{"429 Too Many Requests", fmt.Errorf("HTTP 429 rate limited"), true},
		{"500 Internal Server Error", fmt.Errorf("HTTP 500 internal error"), true},
		{"502 Bad Gateway", fmt.Errorf("HTTP 502 bad gateway"), true},
		{"503 Service Unavailable", fmt.Errorf("HTTP 503 service unavailable"), true},
		{"504 Gateway Timeout", fmt.Errorf("HTTP 504 gateway timeout"), true},
		{"529 Overloaded", fmt.Errorf("HTTP 529 overloaded"), true},

		// Non-retriable HTTP status codes: 400, 401, 403, 404, 422
		{"400 Bad Request", fmt.Errorf("HTTP 400 bad request"), false},
		{"401 Unauthorized", fmt.Errorf("HTTP 401 unauthorized"), false},
		{"403 Forbidden", fmt.Errorf("HTTP 403 forbidden"), false},
		{"404 Not Found", fmt.Errorf("HTTP 404 not found"), false},
		{"422 Unprocessable Entity", fmt.Errorf("HTTP 422 unprocessable entity"), false},

		// Context errors
		{"context.DeadlineExceeded", context.DeadlineExceeded, true},
		{"context.Canceled", context.Canceled, false},

		// TransientError type
		{"TransientError 500", &TransientError{StatusCode: 500, Message: "oops"}, true},
		{"TransientError 429", &TransientError{StatusCode: 429, Message: "rate limit"}, true},

		// Wrapped TransientError
		{"wrapped TransientError", fmt.Errorf("outer: %w", &TransientError{StatusCode: 503, Message: "down"}), true},

		// APIConnectionTimeoutError
		{"APIConnectionTimeoutError", NewAPIConnectionTimeoutError("timed out"), true},

		// Message-based pattern matching: timeout variants
		{"timeout message", fmt.Errorf("request timeout occurred"), true},
		{"timed out message", fmt.Errorf("connection timed out"), true},
		{"deadline exceeded message", fmt.Errorf("deadline exceeded"), true},

		// Message-based pattern matching: connection issues
		{"connection refused", fmt.Errorf("connection refused"), true},
		{"connection reset", fmt.Errorf("connection reset by peer"), true},
		{"EOF in message", fmt.Errorf("unexpected EOF"), true},
		{"broken pipe", fmt.Errorf("broken pipe"), true},

		// Message-based pattern matching: overload/retry signals
		{"temporarily unavailable", fmt.Errorf("temporarily unavailable"), true},
		{"overloaded message", fmt.Errorf("server overloaded"), true},
		{"try again", fmt.Errorf("please try again later"), true},

		// Message-based pattern matching: server errors
		{"unavailable message", fmt.Errorf("service unavailable"), true},
		{"server error message", fmt.Errorf("internal server error"), true},
		{"bad gateway message", fmt.Errorf("bad gateway error"), true},

		// Message-based pattern matching: rate limiting
		{"rate limit", fmt.Errorf("rate limit exceeded"), true},
		{"rate_limit", fmt.Errorf("rate_limit hit"), true},

		// Status code embedded in longer message
		{"503 in longer message", fmt.Errorf("provider returned HTTP 503, retrying"), true},
		{"401 in longer message", fmt.Errorf("got HTTP 401 from api"), false},

		// Case insensitivity in message matching
		{"uppercase TIMEOUT", fmt.Errorf("TIMEOUT waiting for response"), true},
		{"mixed case", fmt.Errorf("Connection Refused by server"), true},

		// Unknown errors: NOT retriable (conservative policy)
		{"unknown error", fmt.Errorf("something weird happened"), false},
		{"serialization error", fmt.Errorf("json: cannot unmarshal"), false},
		{"malformed response", fmt.Errorf("malformed response body"), false},
		{"unknown with no status", fmt.Errorf("provider crashed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTransient(tt.err)
			if got != tt.transient {
				t.Errorf("IsTransient(%v) = %v, want %v", tt.err, got, tt.transient)
			}
		})
	}
}

func TestExtractHTTPStatus(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		code  int
		found bool
	}{
		{"nil error", nil, 0, false},
		{"HTTP 503", fmt.Errorf("HTTP 503 from api"), 503, true},
		{"http 429 lowercase", fmt.Errorf("http 429 rate limited"), 429, true},
		{"status 500", fmt.Errorf("server returned status 500"), 500, true},
		{"no code", fmt.Errorf("something went wrong"), 0, false},
		{"200 OK", fmt.Errorf("HTTP 200 OK"), 200, true},
		{"301 redirect", fmt.Errorf("HTTP 301 moved"), 301, true},
		{"embedded 503", fmt.Errorf("provider returned HTTP 503, retrying"), 503, true},
		{"first 3-digit wins", fmt.Errorf("HTTP 502 then HTTP 503"), 502, true},
		{"short message no code", fmt.Errorf("fail"), 0, false},
		{"TransientError extracts status", &TransientError{StatusCode: 504, Message: "timeout"}, 504, true},
		{"APIError with status in message", &APIError{Status: 401, Message: "HTTP 401 unauthorized"}, 401, true},
		{"APIError without status in message", &APIError{Status: 401, Message: "unauthorized"}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, found := ExtractHTTPStatus(tt.err)
			if found != tt.found {
				t.Errorf("ExtractHTTPStatus(%v) found = %v, want %v", tt.err, found, tt.found)
			}
			if found && code != tt.code {
				t.Errorf("ExtractHTTPStatus(%v) code = %d, want %d", tt.err, code, tt.code)
			}
		})
	}
}

func TestTransientErrorType(t *testing.T) {
	te := &TransientError{StatusCode: 503, Message: "service down"}
	msg := te.Error()

	if !strings.Contains(msg, "503") {
		t.Errorf("expected 503 in message, got %q", msg)
	}
	if !strings.Contains(msg, "service down") {
		t.Errorf("expected 'service down' in message, got %q", msg)
	}
	if !strings.Contains(msg, "transient error") {
		t.Errorf("expected 'transient error' prefix in message, got %q", msg)
	}

	// Verify errors.As works for TransientError.
	var target *TransientError
	if !errors.As(te, &target) {
		t.Fatal("errors.As should match *TransientError")
	}
	if target.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", target.StatusCode)
	}
	if target.Message != "service down" {
		t.Errorf("expected message 'service down', got %q", target.Message)
	}

	// Verify errors.As works through wrapping.
	wrapped := fmt.Errorf("outer: %w", te)
	var target2 *TransientError
	if !errors.As(wrapped, &target2) {
		t.Fatal("errors.As should match *TransientError through wrapping")
	}
	if target2.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", target2.StatusCode)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		message       string
		wantTransient bool
	}{
		// Retriable codes produce TransientError
		{"408 produces TransientError", 408, "request timeout", true},
		{"429 produces TransientError", 429, "rate limited", true},
		{"500 produces TransientError", 500, "server error", true},
		{"502 produces TransientError", 502, "bad gateway", true},
		{"503 produces TransientError", 503, "unavailable", true},
		{"504 produces TransientError", 504, "timeout", true},
		{"529 produces TransientError", 529, "overloaded", true},

		// Non-retriable codes produce APIError (not TransientError)
		{"400 produces APIError", 400, "bad request", false},
		{"401 produces APIError", 401, "unauthorized", false},
		{"403 produces APIError", 403, "forbidden", false},
		{"404 produces APIError", 404, "not found", false},
		{"422 produces APIError", 422, "unprocessable", false},
		{"200 produces APIError", 200, "ok", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError(tt.status, tt.message)

			var te *TransientError
			isTransient := errors.As(err, &te)
			if isTransient != tt.wantTransient {
				t.Errorf("ClassifyError(%d, %q): isTransient = %v, want %v",
					tt.status, tt.message, isTransient, tt.wantTransient)
			}

			// Verify the error message contains the original message.
			if !strings.Contains(err.Error(), tt.message) {
				t.Errorf("error %q should contain %q", err.Error(), tt.message)
			}

			// If it's a TransientError, verify the status code.
			if isTransient && te.StatusCode != tt.status {
				t.Errorf("TransientError.StatusCode = %d, want %d", te.StatusCode, tt.status)
			}

			// If it's not a TransientError, verify it's an APIError with correct status.
			if !isTransient {
				var ae *APIError
				if !errors.As(err, &ae) {
					t.Fatal("non-transient ClassifyError should return *APIError")
				}
				if ae.Status != tt.status {
					t.Errorf("APIError.Status = %d, want %d", ae.Status, tt.status)
				}
			}
		})
	}
}

// TestIsTransientVsIsRetriableDivergence documents the intentional divergence
// between IsTransient (conservative) and the client's isRetriableError (optimistic).
// IsTransient returns false for unknown errors; isRetriableError returns true.
// This test ensures the conservative policy is preserved.
func TestIsTransientVsIsRetriableDivergence(t *testing.T) {
	// These errors are unknown to IsTransient — they don't match any known
	// pattern. IsTransient should say "not retriable" (conservative).
	unknownErrors := []error{
		fmt.Errorf("something weird happened"),
		fmt.Errorf("json: cannot unmarshal"),
		fmt.Errorf("malformed response body"),
		fmt.Errorf("provider crashed"),
		fmt.Errorf("internal assertion failed"),
	}

	for _, err := range unknownErrors {
		if IsTransient(err) {
			t.Errorf("IsTransient should be conservative for unknown error %q, got true", err)
		}
	}

	// Known retriable errors should be retriable in both functions.
	retriableErrors := []error{
		context.DeadlineExceeded,
		fmt.Errorf("HTTP 503 service unavailable"),
		fmt.Errorf("rate limit exceeded"),
		&TransientError{StatusCode: 500, Message: "oops"},
	}
	for _, err := range retriableErrors {
		if !IsTransient(err) {
			t.Errorf("IsTransient should be true for known retriable error %q", err)
		}
	}

	// context.Canceled should NOT be retriable in either function.
	if IsTransient(context.Canceled) {
		t.Error("IsTransient(context.Canceled) should be false")
	}
}
