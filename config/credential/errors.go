package credential

import "errors"

var (
	errEmptyCredential       = errors.New("paste an API key")
	errPlaceholderCredential = errors.New("API key looks like a placeholder — paste your real key")
	errCredentialTooShort    = errors.New("API key appears too short")
)
