package credentials

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// LookupSecret returns an API key from the OS secret store.
// It does not read the process environment — use for strict isolation from agents and shell dumps.
func LookupSecret(ctx context.Context, envKey string) string {
	if ctx == nil {
		ctx = context.Background()
	}
	envKey = strings.TrimSpace(envKey)
	if envKey == "" {
		return ""
	}
	secret, err := DefaultStore().Get(ctx, AccountForEnv(envKey))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(secret)
}

// HasSecret reports whether the store has a non-empty secret for an env key name.
func HasSecret(ctx context.Context, envKey string) bool {
	return LookupSecret(ctx, envKey) != ""
}

// DeleteSecret removes a stored secret for an env key name.
func DeleteSecret(ctx context.Context, envKey string) error {
	envKey = strings.TrimSpace(envKey)
	if envKey == "" {
		return fmt.Errorf("credentials: env key required")
	}
	return DefaultStore().Delete(ctx, AccountForEnv(envKey))
}

// ScrubProcessEnv removes API key env vars from the process (shell inheritance / legacy Setenv).
func ScrubProcessEnv(envKeys []string) {
	for _, k := range envKeys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		_ = os.Unsetenv(k)
	}
}
