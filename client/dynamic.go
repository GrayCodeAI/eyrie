package client

import (
	"os"
	"sync"
)

// dynamicMu protects the OpenAICompatibleProviders map from concurrent access.
var dynamicMu sync.RWMutex

// RegisterDynamicProvider adds a user-defined OpenAI-compatible provider at runtime.
// name is the provider key (e.g. "my-local-llm"), baseURL is the API base
// (e.g. "http://localhost:8080/v1"), and envKey is the environment variable
// that holds the API key (e.g. "MY_LLM_API_KEY"). If envKey is empty, the
// provider is treated like ollama (no key required).
func RegisterDynamicProvider(name, baseURL, envKey string) {
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
