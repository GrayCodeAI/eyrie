package client

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
)

// dynamicMu protects the OpenAICompatibleProviders map from concurrent access.
var dynamicMu sync.RWMutex

// registryFrozen prevents new provider registrations after first use.
var registryFrozen atomic.Bool

// FreezeRegistry prevents further provider registrations.
// Called automatically after first provider lookup.
func FreezeRegistry() {
	registryFrozen.Store(true)
}

// RegisterDynamicProvider adds a user-defined OpenAI-compatible provider at runtime.
// name is the provider key (e.g. "my-local-llm"), baseURL is the API base
// (e.g. "http://localhost:8080/v1"), and envKey is the environment variable
// that holds the API key (e.g. "MY_LLM_API_KEY"). If envKey is empty, the
// provider is treated like ollama (no key required).
// Returns error if registry is frozen (after first provider lookup).
func RegisterDynamicProvider(name, baseURL, envKey string) error {
	if registryFrozen.Load() {
		return fmt.Errorf("eyrie: provider registry is frozen; register providers before first use")
	}
	if baseURL == "" {
		return fmt.Errorf("eyrie: RegisterDynamicProvider: baseURL must not be empty")
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("eyrie: RegisterDynamicProvider: invalid baseURL %q (must be http/https with host)", baseURL)
	}
	dynamicMu.Lock()
	defer dynamicMu.Unlock()

	OpenAICompatibleProviders[name] = ProviderRegistryConfig{
		Name:              name,
		Type:              ProviderTypeOpenAICompatible,
		BaseURL:           baseURL,
		EnvKey:            envKey,
		SupportsStreaming: true,
		SupportsTools:     true,
		SupportsReasoning: false,
		Compat: &OpenAICompatConfig{
			MaxTokensField: "max_tokens",
		},
	}
	return nil
}

// getOrCreateProviderWithFallback extends getOrCreateProvider with a fallback:
// if the provider is not found in any registry map and OPENAI_API_BASE or
// OPENAI_BASE_URL is set, it creates an OpenAI-compatible client using that URL.
// This is invoked automatically inside getOrCreateProvider.
func openaiBaseFallbackURL() string {
	if u := os.Getenv("OPENAI_API_BASE"); u != "" {
		return u
	}
	if u := os.Getenv("OPENAI_BASE_URL"); u != "" {
		return u
	}
	return ""
}
