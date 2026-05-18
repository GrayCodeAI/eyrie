package config

import (
	"os"
	"strings"
)

// OpenAICompatibleRuntimeMode identifies the runtime mode.
type OpenAICompatibleRuntimeMode = string

// OpenAICompatibleApiKeySource identifies where the API key came from.
type OpenAICompatibleApiKeySource = string

// ResolvedOpenAICompatibleRuntime holds the resolved runtime config.
type ResolvedOpenAICompatibleRuntime struct {
	Mode         string                  `json:"mode"`
	Request      ResolvedProviderRequest `json:"request"`
	APIKey       string                  `json:"api_key"`
	APIKeySource string                  `json:"api_key_source"`
}

// IsOpenAICompatibleRuntimeEnabled checks if any provider API key is set.
func IsOpenAICompatibleRuntimeEnabled() bool {
	keys := []string{
		"OPENROUTER_API_KEY", "GROK_API_KEY", "XAI_API_KEY", "GEMINI_API_KEY",
		"ANTHROPIC_API_KEY", "CANOPYWAVE_API_KEY", "OPENAI_API_KEY",
		"OPENCODEGO_API_KEY", "OLLAMA_BASE_URL",
	}
	for _, k := range keys {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

func asTrimmedEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func firstEnvValue(keys []string) string {
	for _, k := range keys {
		if v := asTrimmedEnv(k); v != "" {
			return v
		}
	}
	return ""
}

func resolveRuntimeProvider() RuntimeProviderProfile {
	for _, key := range OpenAICompatibleRuntimeProfileOrder {
		profile := OpenAICompatibleRuntimeProfiles[key]
		for _, envKey := range profile.DetectionEnv {
			if asTrimmedEnv(envKey) != "" {
				return profile
			}
		}
	}
	if os.Getenv("OLLAMA_BASE_URL") != "" {
		base := OpenAIRuntimeProfile
		base.DefaultBaseURL = os.Getenv("OLLAMA_BASE_URL")
		if base.DefaultBaseURL == "" {
			base.DefaultBaseURL = OllamaDefaultBaseURL
		}
		base.DefaultModel = OllamaDefaultModel
		base.ModelEnv = []string{"OLLAMA_MODEL"}
		base.BaseURLEnv = []string{"OLLAMA_BASE_URL"}
		base.APIKeys = nil
		return base
	}
	return OpenAIRuntimeProfile
}

func resolveProviderAPIKey(profile RuntimeProviderProfile) (string, string) {
	for _, k := range profile.APIKeys {
		if v := asTrimmedEnv(k.Env); v != "" {
			return v, k.Source
		}
	}
	return "", "none"
}

// ResolveOpenAICompatibleRuntime resolves the full OpenAI-compatible runtime config.
func ResolveOpenAICompatibleRuntime(model, baseURL, fallbackModel string) ResolvedOpenAICompatibleRuntime {
	provider := resolveRuntimeProvider()
	runtimeModel := model
	if runtimeModel == "" {
		runtimeModel = firstEnvValue(provider.ModelEnv)
	}
	runtimeBaseURL := baseURL
	if runtimeBaseURL == "" {
		runtimeBaseURL = firstEnvValue(provider.BaseURLEnv)
	}
	if runtimeBaseURL == "" {
		runtimeBaseURL = provider.DefaultBaseURL
	}

	fb := fallbackModel
	if fb == "" {
		fb = firstEnvValue(provider.ModelEnv)
	}
	if fb == "" {
		fb = provider.DefaultModel
	}

	request := ResolveProviderRequest(runtimeModel, runtimeBaseURL, fb)
	apiKey, apiKeySource := resolveProviderAPIKey(provider)

	return ResolvedOpenAICompatibleRuntime{
		Mode:         provider.Mode,
		Request:      request,
		APIKey:       apiKey,
		APIKeySource: apiKeySource,
	}
}
