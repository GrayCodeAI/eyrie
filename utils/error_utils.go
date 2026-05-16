package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var sslErrorCodes = map[string]bool{
	"UNABLE_TO_VERIFY_LEAF_SIGNATURE":             true,
	"UNABLE_TO_GET_ISSUER_CERT":                   true,
	"UNABLE_TO_GET_ISSUER_CERT_LOCALLY":           true,
	"CERT_SIGNATURE_FAILURE":                      true,
	"CERT_NOT_YET_VALID":                          true,
	"CERT_HAS_EXPIRED":                            true,
	"CERT_REVOKED":                                true,
	"CERT_REJECTED":                               true,
	"CERT_UNTRUSTED":                              true,
	"DEPTH_ZERO_SELF_SIGNED_CERT":                 true,
	"SELF_SIGNED_CERT_IN_CHAIN":                   true,
	"CERT_CHAIN_TOO_LONG":                         true,
	"PATH_LENGTH_EXCEEDED":                        true,
	"ERR_TLS_CERT_ALTNAME_INVALID":                true,
	"HOSTNAME_MISMATCH":                           true,
	"ERR_TLS_HANDSHAKE_TIMEOUT":                   true,
	"ERR_SSL_WRONG_VERSION_NUMBER":                true,
	"ERR_SSL_DECRYPTION_FAILED_OR_BAD_RECORD_MAC": true,
}

// ConnectionErrorDetails holds details about a connection error.
type ConnectionErrorDetails struct {
	Code       string
	Message    string
	IsSSLError bool
}

type coder interface {
	Code() string
}

var titleRe = regexp.MustCompile(`(?i)<title>(.*?)</title>`)

// ExtractConnectionErrorDetails walks the error chain up to 5 levels deep,
// looking for an error with a Code() string method, and checks if the code
// is a known SSL error.
func ExtractConnectionErrorDetails(err error) *ConnectionErrorDetails {
	current := err
	for i := 0; i < 5 && current != nil; i++ {
		if c, ok := current.(coder); ok {
			code := c.Code()
			return &ConnectionErrorDetails{
				Code:       code,
				Message:    current.Error(),
				IsSSLError: sslErrorCodes[code],
			}
		}
		current = errors.Unwrap(current)
	}
	return nil
}

// GetSSLErrorHint returns a hint string if the error is an SSL error,
// suggesting corporate proxy or NODE_EXTRA_CA_CERTS configuration.
func GetSSLErrorHint(err error) *string {
	details := ExtractConnectionErrorDetails(err)
	if details == nil || !details.IsSSLError {
		return nil
	}
	hint := fmt.Sprintf(
		"SSL error (%s): if you are behind a corporate proxy, ensure your certificates are properly configured. "+
			"You can set NODE_EXTRA_CA_CERTS to point to your CA bundle.",
		details.Code,
	)
	return &hint
}

// SanitizeAPIError extracts the <title> content from HTML error responses,
// or returns the message as-is if it is not HTML.
func SanitizeAPIError(message string) string {
	if !strings.Contains(message, "<!DOCTYPE html") && !strings.Contains(message, "<html") {
		return message
	}
	if m := titleRe.FindStringSubmatch(message); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return message
}
