package errors

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	APIErrorMessagePrefix                  = "API Error"
	PromptTooLongErrorMessage              = "Prompt is too long"
	CreditBalanceTooLowErrorMessage        = "Credit balance is too low"
	InvalidAPIKeyErrorMessage              = "Not logged in · Please run /login"               // #nosec G101 -- static error message text, not a secret value
	InvalidAPIKeyErrorMessageExternal      = "Invalid API key · Please check your credentials" // #nosec G101 -- static error message text, not a secret value
	OrgDisabledErrorMessageEnvKeyWithOAuth = "Organization disabled"
	OrgDisabledErrorMessageEnvKey          = "Organization disabled"
	TokenRevokedErrorMessage               = "Token has been revoked" // #nosec G101 -- static error message text, not a secret value
	CCRAuthErrorMessage                    = "Authentication error"
	Repeated529ErrorMessage                = "Repeated 529 Overloaded errors"
	CustomOffSwitchMessage                 = "Service temporarily disabled"
	APITimeoutErrorMessage                 = "Request timed out"
	OAuthOrgNotAllowedErrorMessage         = "Organization not allowed for OAuth"
)

var promptTokensRe = regexp.MustCompile(`(?i)prompt is too long[^0-9]*(\d+)\s*tokens?\s*>\s*(\d+)`)

func StartsWithApiErrorPrefix(text string) bool {
	return strings.HasPrefix(text, APIErrorMessagePrefix) ||
		strings.HasPrefix(text, "Please run /login · "+APIErrorMessagePrefix)
}

func ParsePromptTooLongTokenCounts(rawMessage string) (actualTokens *int, limitTokens *int) {
	m := promptTokensRe.FindStringSubmatch(rawMessage)
	if m == nil {
		return nil, nil
	}
	a, err1 := strconv.Atoi(m[1])
	l, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &a, &l
}

func IsMediaSizeError(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "image is too large") ||
		strings.Contains(lower, "pdf is too large") ||
		strings.Contains(lower, "file is too large") ||
		strings.Contains(lower, "request is too large")
}

func GetPdfTooLargeErrorMessage() string {
	return "PDF is too large · Try splitting into smaller files or reducing quality"
}

func GetPdfPasswordProtectedErrorMessage() string {
	return "PDF is password protected · Please remove the password and try again"
}

func GetPdfInvalidErrorMessage() string {
	return "PDF is invalid · Please check the file and try again"
}

func GetImageTooLargeErrorMessage() string {
	return "Image is too large · Please reduce the file size and try again"
}

func GetRequestTooLargeErrorMessage() string {
	return "Request is too large · Please reduce the size and try again"
}

func GetTokenRevokedErrorMessage() string {
	return "Token has been revoked · Please run /login to reconnect"
}

func GetOauthOrgNotAllowedErrorMessage() string {
	return "Organization not allowed · Please ask IT to allowlist"
}
