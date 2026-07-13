package engine

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, machine-readable engine failure category.
type ErrorCode string

const (
	ErrorInvalidRequest      ErrorCode = "invalid_request"
	ErrorCredentialMissing   ErrorCode = "credential_missing"
	ErrorAuthentication      ErrorCode = "authentication_failed"
	ErrorCatalogUnavailable  ErrorCode = "catalog_unavailable"
	ErrorModelUnavailable    ErrorCode = "model_unavailable"
	ErrorCapabilityMismatch  ErrorCode = "capability_mismatch"
	ErrorContextExceeded     ErrorCode = "context_exceeded"
	ErrorRateLimited         ErrorCode = "rate_limited"
	ErrorBudgetExceeded      ErrorCode = "budget_exceeded"
	ErrorProviderUnavailable ErrorCode = "provider_unavailable"
	ErrorCancelled           ErrorCode = "cancelled"
	ErrorInternal            ErrorCode = "internal"
)

// Error is the typed error returned across the host boundary.
type Error struct {
	Code      ErrorCode
	Operation string
	Provider  string
	Model     string
	Message   string
	Retryable bool
	Cause     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Operation != "" {
		return fmt.Sprintf("eyrie engine: %s failed", e.Operation)
	}
	return "eyrie engine error"
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsCode reports whether err or an error in its chain has the supplied code.
func IsCode(err error, code ErrorCode) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func invalid(operation, message string) error {
	return &Error{Code: ErrorInvalidRequest, Operation: operation, Message: message}
}
