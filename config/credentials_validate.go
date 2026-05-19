package config

import (
	"errors"
	"fmt"
	"strings"
)

// LooksLikePlaceholderSecret reports obvious non-secrets (templates, examples, too short).
func LooksLikePlaceholderSecret(secret string) bool {
	s := strings.TrimSpace(strings.ToLower(secret))
	if s == "" {
		return true
	}
	if len(s) < 8 {
		return true
	}
	switch s {
	case "sua_chave", "changeme", "your-api-key", "your_api_key", "api-key", "api_key",
		"xxx", "test", "dummy", "fake", "placeholder", "insert-key-here", "api_key_here",
		"your-api-key-here":
		return true
	}
	if strings.HasPrefix(s, "your-") && strings.Contains(s, "key") {
		return true
	}
	if strings.HasPrefix(s, "sk-xxx") || strings.HasPrefix(s, "sk-your") || strings.HasPrefix(s, "sk-test") {
		return true
	}
	if strings.Contains(s, "your_api") || strings.Contains(s, "api_key_here") || strings.Contains(s, "<") {
		return true
	}
	return false
}

// ValidateCredentialSecret rejects placeholders and obviously invalid keys before persistence.
func ValidateCredentialSecret(envKey, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	if LooksLikePlaceholderSecret(secret) {
		return errors.New("API key looks like a placeholder — paste your real key")
	}
	label := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(envKey), "_API_KEY"))
	if label == "" {
		label = "provider"
	}
	if msg := ValidateAPIKey(secret, label); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	return nil
}
