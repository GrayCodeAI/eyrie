package credentials

import (
	"context"
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

// DefaultStore returns the process-wide credential store (OS secret service).
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
	case "zai_api_key":
		return "ZAI_API_KEY"
	case "canopywave_api_key":
		return "CANOPYWAVE_API_KEY"
	case "opencodego_api_key":
		return "OPENCODEGO_API_KEY"
	case "kimi_api_key", "moonshot_api_key":
		return "KIMI_API_KEY"
	case "xiaomi_api_key", "mimo_api_key":
		return "XIAOMI_API_KEY"
	case "ollama_base_url":
		return "OLLAMA_BASE_URL"
	default:
		return strings.ToUpper(strings.ReplaceAll(account, "-", "_"))
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
		"MOONSHOT_API_KEY", "KIMI_API_KEY", "MIMO_API_KEY",
		"CANOPYWAVE_API_KEY", "OPENCODEGO_API_KEY",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
		"VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN",
	}
}
