package client

import "fmt"

// EyrieError is a structured error that preserves provider context,
// HTTP metadata, and request identification for debugging.
type EyrieError struct {
	Provider  string
	Op        string // operation that failed (e.g. "chat", "stream", "ping")
	StatusCode int
	RequestID string
	Message   string
	Err       error
}

func (e *EyrieError) Error() string {
	s := fmt.Sprintf("eyrie: %s %s failed", e.Provider, e.Op)
	if e.StatusCode > 0 {
		s += fmt.Sprintf(" (HTTP %d)", e.StatusCode)
	}
	if e.RequestID != "" {
		s += fmt.Sprintf(" [request_id=%s]", e.RequestID)
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	if e.Err != nil {
		s += ": " + e.Err.Error()
	}
	return s
}

func (e *EyrieError) Unwrap() error { return e.Err }

// IsRetriable returns true if the error is likely transient and retrying may help.
func (e *EyrieError) IsRetriable() bool {
	switch e.StatusCode {
	case 429, 500, 502, 503, 529:
		return true
	}
	return false
}

// IsAuthError returns true if the error indicates an authentication/authorization problem.
func (e *EyrieError) IsAuthError() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

// IsRateLimited returns true if the error indicates rate limiting.
func (e *EyrieError) IsRateLimited() bool {
	return e.StatusCode == 429
}
