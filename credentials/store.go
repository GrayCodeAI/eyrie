package credentials

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/GrayCodeAI/eyrie/catalog"
)

const (
	ServiceName = "hawk"
)

// Store persists provider API secrets outside provider.json.
type Store interface {
	Set(ctx context.Context, account, secret string) error
	Get(ctx context.Context, account string) (string, error)
	Delete(ctx context.Context, account string) error
}

var (
	defaultStore   Store
	defaultStoreMu sync.RWMutex
)

// DefaultStore returns the process-wide credential store (keychain + env fallback).
func DefaultStore() Store {
	defaultStoreMu.RLock()
	s := defaultStore
	defaultStoreMu.RUnlock()
	if s != nil {
		return s
	}
	defaultStoreMu.Lock()
	defer defaultStoreMu.Unlock()
	if defaultStore == nil {
		defaultStore = NewCombinedStore()
	}
	return defaultStore
}

// SetDefaultStore replaces the process-wide store (tests).
func SetDefaultStore(s Store) {
	defaultStoreMu.Lock()
	defaultStore = s
	defaultStoreMu.Unlock()
}

// AccountForEnv maps a standard env var to a stable keychain account name.
func AccountForEnv(envKey string) string {
	return strings.ToLower(strings.TrimSpace(envKey))
}

// EnvForAccount is a best-effort reverse map for loading into process env.
func EnvForAccount(account string) string {
	switch strings.ToLower(account) {
	case "anthropic_api_key":
		return "ANTHROPIC_API_KEY"
	case "openai_api_key":
		return "OPENAI_API_KEY"
	case "openrouter_api_key":
		return "OPENROUTER_API_KEY"
	case "gemini_api_key":
		return "GEMINI_API_KEY"
	case "grok_api_key", "xai_api_key":
		return "GROK_API_KEY"
	default:
		return strings.ToUpper(strings.ReplaceAll(account, "-", "_"))
	}
}

// ApplyToProcess sets env vars from the store for catalog-defined credential accounts.
func ApplyToProcess(ctx context.Context, store Store) {
	if store == nil {
		store = DefaultStore()
	}
	keys := discoveryEnvKeys(ctx)
	for _, envKey := range keys {
		if strings.TrimSpace(os.Getenv(envKey)) != "" {
			continue
		}
		secret, err := store.Get(ctx, AccountForEnv(envKey))
		if err != nil || strings.TrimSpace(secret) == "" {
			continue
		}
		_ = os.Setenv(envKey, secret)
	}
}

// APIKeysMap returns env-keyed secrets for catalog discovery.
func APIKeysMap(ctx context.Context, store Store) map[string]string {
	if store == nil {
		store = DefaultStore()
	}
	out := map[string]string{}
	for _, envKey := range discoveryEnvKeys(ctx) {
		secret, err := store.Get(ctx, AccountForEnv(envKey))
		if err != nil || strings.TrimSpace(secret) == "" {
			continue
		}
		out[envKey] = secret
	}
	return out
}

func discoveryEnvKeys(ctx context.Context) []string {
	if ctx == nil {
		ctx = context.Background()
	}
	if keys := catalog.DiscoveryEnvKeyNames(ctx); len(keys) > 0 {
		return keys
	}
	return []string{
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY",
		"GEMINI_API_KEY", "GROK_API_KEY", "XAI_API_KEY",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
		"VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN",
	}
}
