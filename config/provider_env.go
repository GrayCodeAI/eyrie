package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
)

// ProviderConfig mirrors ~/.hawk/provider.json.
type ProviderConfig struct {
	Version          string `json:"_version,omitempty"`
	ActiveProvider   string `json:"active_provider,omitempty"`
	AnthropicAPIKey  string `json:"anthropic_api_key,omitempty"`
	GrokAPIKey       string `json:"grok_api_key,omitempty"`
	XAIAPIKey        string `json:"xai_api_key,omitempty"`
	OpenAIAPIKey     string `json:"openai_api_key,omitempty"`
	CanopyWaveAPIKey string `json:"canopywave_api_key,omitempty"`
	OpenRouterAPIKey string `json:"openrouter_api_key,omitempty"`
	GeminiAPIKey     string `json:"gemini_api_key,omitempty"`
	OllamaBaseURL    string `json:"ollama_base_url,omitempty"`
	OpenCodeGoAPIKey string `json:"opencodego_api_key,omitempty"`
	AnthropicBaseURL string `json:"anthropic_base_url,omitempty"`
	CanopyWaveBaseURL string `json:"canopywave_base_url,omitempty"`
	GrokBaseURL      string `json:"grok_base_url,omitempty"`
	XAIBaseURL       string `json:"xai_base_url,omitempty"`
	OpenAIBaseURL    string `json:"openai_base_url,omitempty"`
	OpenRouterBaseURL string `json:"openrouter_base_url,omitempty"`
	GeminiBaseURL    string `json:"gemini_base_url,omitempty"`
	OpenCodeGoBaseURL string `json:"opencodego_base_url,omitempty"`
	AnthropicModel   string `json:"anthropic_model,omitempty"`
	OpenAIModel      string `json:"openai_model,omitempty"`
	CanopyWaveModel  string `json:"canopywave_model,omitempty"`
	GrokModel        string `json:"grok_model,omitempty"`
	XAIModel         string `json:"xai_model,omitempty"`
	OpenRouterModel  string `json:"openrouter_model,omitempty"`
	GeminiModel      string `json:"gemini_model,omitempty"`
	OllamaModel      string `json:"ollama_model,omitempty"`
	OpenCodeGoModel  string `json:"opencodego_model,omitempty"`
	ActiveModel      string `json:"active_model,omitempty"`
	ExplorationModel string `json:"exploration_model,omitempty"`
	AnthropicVersion string `json:"anthropic_version,omitempty"`
}

// providerFieldMap defines which config fields map to each provider.
type providerFieldMap struct {
	APIKeys  func(c *ProviderConfig) []string
	Models   func(c *ProviderConfig) []string
	BaseURL  func(c *ProviderConfig) string
}

var providerFields = map[string]providerFieldMap{
	ProviderAnthropic: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.AnthropicAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.AnthropicModel} },
		BaseURL: func(c *ProviderConfig) string { return c.AnthropicBaseURL },
	},
	ProviderOpenAI: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.OpenAIAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.OpenAIModel} },
		BaseURL: func(c *ProviderConfig) string { return c.OpenAIBaseURL },
	},
	ProviderCanopyWave: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.CanopyWaveAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.CanopyWaveModel} },
		BaseURL: func(c *ProviderConfig) string { return c.CanopyWaveBaseURL },
	},
	ProviderOpenRouter: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.OpenRouterAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.OpenRouterModel} },
		BaseURL: func(c *ProviderConfig) string { return c.OpenRouterBaseURL },
	},
	ProviderGrok: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.GrokAPIKey, c.XAIAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.GrokModel, c.XAIModel} },
		BaseURL: func(c *ProviderConfig) string { return firstNonEmpty(c.GrokBaseURL, c.XAIBaseURL) },
	},
	ProviderGemini: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.GeminiAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.GeminiModel} },
		BaseURL: func(c *ProviderConfig) string { return c.GeminiBaseURL },
	},
	ProviderOllama: {
		APIKeys: func(c *ProviderConfig) []string { return nil },
		Models:  func(c *ProviderConfig) []string { return []string{c.OllamaModel} },
		BaseURL: func(c *ProviderConfig) string { return c.OllamaBaseURL },
	},
	ProviderOpenCodeGo: {
		APIKeys: func(c *ProviderConfig) []string { return []string{c.OpenCodeGoAPIKey} },
		Models:  func(c *ProviderConfig) []string { return []string{c.OpenCodeGoModel} },
		BaseURL: func(c *ProviderConfig) string { return c.OpenCodeGoBaseURL },
	},
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

// AsNonEmptyString returns trimmed string or empty.
func AsNonEmptyString(v string) string {
	if t := strings.TrimSpace(v); t != "" {
		return t
	}
	return ""
}

// NormalizeOllamaOpenAIBaseURL ensures the URL ends with /v1.
func NormalizeOllamaOpenAIBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	trimmed := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed
	}
	return trimmed + "/v1"
}

// SetEnvValue sets an env var if value is non-empty and overwrite is allowed.
func SetEnvValue(key, value string, overwrite bool) {
	if value == "" {
		return
	}
	if !overwrite && os.Getenv(key) != "" {
		return
	}
	_ = os.Setenv(key, value)
}

// ApplyOpenAICompatibleProvider sets env vars for an OpenAI-compatible provider.
func ApplyOpenAICompatibleProvider(prefix, apiKey, model, baseURL string, overwrite bool) {
	SetEnvValue(prefix+"_API_KEY", apiKey, overwrite)
	SetEnvValue(prefix+"_MODEL", model, overwrite)
	SetEnvValue(prefix+"_BASE_URL", baseURL, overwrite)
	SetEnvValue("OPENAI_API_KEY", apiKey, overwrite)
	SetEnvValue("OPENAI_MODEL", model, overwrite)
	SetEnvValue("OPENAI_BASE_URL", baseURL, overwrite)
}

// GetProviderModel returns the configured model for a provider.
func GetProviderModel(config *ProviderConfig, provider string) string {
	if f, ok := providerFields[provider]; ok {
		for _, m := range f.Models(config) {
			if v := AsNonEmptyString(m); v != "" {
				return v
			}
		}
	}
	return ""
}

// GetProviderAPIKey returns the configured API key for a provider.
func GetProviderAPIKey(config *ProviderConfig, provider string) string {
	if f, ok := providerFields[provider]; ok {
		for _, k := range f.APIKeys(config) {
			if v := AsNonEmptyString(k); v != "" {
				return v
			}
		}
	}
	return ""
}

// ValidateAPIKey validates an API key.
func ValidateAPIKey(apiKey, providerName string) string {
	if apiKey == "" {
		return providerName + " requires an API key"
	}
	if apiKey == "SUA_CHAVE" {
		return providerName + " API key cannot be placeholder value 'SUA_CHAVE'"
	}
	if len(apiKey) < 10 {
		return providerName + " API key appears invalid (too short)"
	}
	return ""
}

// ValidateBaseURL validates a base URL.
func ValidateBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	if _, err := os.Stat(baseURL); err == nil {
		return "Invalid base URL: " + baseURL
	}
	return ""
}

// ProviderDetectionOrder is the priority order for provider detection.
var ProviderDetectionOrder = APIProviderDetectionOrder

// GetProviderConfigDir returns the config directory path.
func GetProviderConfigDir() string {
	if d := os.Getenv("HAWK_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hawk")
}

// GetProviderConfigPath returns the full path to provider.json.
func GetProviderConfigPath() string {
	return filepath.Join(GetProviderConfigDir(), "provider.json")
}

// LoadProviderConfig loads provider config from disk.
// Returns nil if file doesn't exist. Returns error for corrupt JSON or permission issues.
func LoadProviderConfig(path string) *ProviderConfig {
	if path == "" {
		path = GetProviderConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg ProviderConfig
	if json.Unmarshal(data, &cfg) != nil {
		return nil
	}
	return &cfg
}

// LoadProviderConfigWithError loads provider config from disk with detailed error reporting.
// Returns (nil, nil) if file doesn't exist.
// Returns (nil, error) for corrupt JSON or permission issues.
func LoadProviderConfigWithError(path string) (*ProviderConfig, error) {
	if path == "" {
		path = GetProviderConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("eyrie: failed to read provider config at %s: %w", path, err)
	}
	var cfg ProviderConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("eyrie: corrupt provider config at %s: %w", path, err)
	}
	if cfg.Version != "" && cfg.Version != "1" {
		return nil, fmt.Errorf("eyrie: unsupported provider config version %q at %s", cfg.Version, path)
	}
	return &cfg, nil
}

// SaveProviderConfig saves provider config to disk.
func SaveProviderConfig(config *ProviderConfig, path string) error {
	if path == "" {
		path = GetProviderConfigPath()
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// IsProviderConfigured checks if a provider has valid configuration.
func IsProviderConfigured(config *ProviderConfig, provider string) bool {
	if provider == ProviderOllama {
		return AsNonEmptyString(config.OllamaBaseURL) != ""
	}
	return GetProviderAPIKey(config, provider) != ""
}

// DefaultProviderFromConfig determines the default provider from config.
func DefaultProviderFromConfig(config *ProviderConfig) string {
	if config == nil {
		return ""
	}
	if ep := AsNonEmptyString(config.ActiveProvider); ep != "" {
		if IsProviderConfigured(config, ep) {
			return ep
		}
	}
	for _, p := range ProviderDetectionOrder {
		if IsProviderConfigured(config, p) {
			return p
		}
	}
	return ""
}

// GetProviderActiveModel gets the active model for a provider from config.
func GetProviderActiveModel(config *ProviderConfig, provider string) string {
	specific := GetProviderModel(config, provider)
	if specific != "" {
		return specific
	}
	// Check if any provider has a scoped model
	for _, p := range ProviderDetectionOrder {
		if GetProviderModel(config, p) != "" {
			return ""
		}
	}
	legacy := AsNonEmptyString(config.ActiveModel)
	if legacy == "" {
		return ""
	}
	if DefaultProviderFromConfig(config) == provider {
		return legacy
	}
	return ""
}

// ClearProviderRuntimeEnv clears all provider-related env vars.
func ClearProviderRuntimeEnv() {
	keys := []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "ANTHROPIC_BASE_URL", "ANTHROPIC_VERSION",
		"OPENAI_API_KEY", "OPENAI_MODEL", "OPENAI_BASE_URL",
		"OPENROUTER_API_KEY", "OPENROUTER_MODEL", "OPENROUTER_BASE_URL",
		"CANOPYWAVE_API_KEY", "CANOPYWAVE_MODEL", "CANOPYWAVE_BASE_URL",
		"GROK_API_KEY", "GROK_MODEL", "GROK_BASE_URL",
		"XAI_API_KEY", "XAI_MODEL", "XAI_BASE_URL",
		"GEMINI_API_KEY", "GEMINI_MODEL", "GEMINI_BASE_URL",
		"OLLAMA_BASE_URL",
		"OPENCODEGO_API_KEY", "OPENCODEGO_MODEL", "OPENCODEGO_BASE_URL",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}

// collectEnvValue adds a key/value to the env map if value is non-empty and
// overwrite is allowed (or the key is not already set in the process env).
func collectEnvValue(env map[string]string, key, value string, overwrite bool) {
	if value == "" {
		return
	}
	if !overwrite && os.Getenv(key) != "" {
		return
	}
	env[key] = value
}

// collectOpenAICompatibleProvider adds env vars for an OpenAI-compatible provider to the map.
func collectOpenAICompatibleProvider(env map[string]string, prefix, apiKey, model, baseURL string, overwrite bool) {
	collectEnvValue(env, prefix+"_API_KEY", apiKey, overwrite)
	collectEnvValue(env, prefix+"_MODEL", model, overwrite)
	collectEnvValue(env, prefix+"_BASE_URL", baseURL, overwrite)
	collectEnvValue(env, "OPENAI_API_KEY", apiKey, overwrite)
	collectEnvValue(env, "OPENAI_MODEL", model, overwrite)
	collectEnvValue(env, "OPENAI_BASE_URL", baseURL, overwrite)
}

// ApplyProviderEnv computes the env vars for a specific provider and returns
// them as a map without modifying the process environment.
func ApplyProviderEnv(provider string, config *ProviderConfig, activeModel string, overwrite bool, cat *catalog.ModelCatalog) map[string]string {
	env := make(map[string]string)
	switch provider {
	case ProviderAnthropic:
		collectEnvValue(env, "ANTHROPIC_API_KEY", AsNonEmptyString(config.AnthropicAPIKey), overwrite)
		m := activeModel
		if m == "" {
			m = catalog.GetPreferredProviderModel("anthropic", catalog.TierSonnet, cat)
		}
		collectEnvValue(env, "ANTHROPIC_MODEL", m, overwrite)
		collectEnvValue(env, "ANTHROPIC_BASE_URL", AsNonEmptyString(config.AnthropicBaseURL), overwrite)
		collectEnvValue(env, "ANTHROPIC_VERSION", AsNonEmptyString(config.AnthropicVersion), overwrite)
	case ProviderOpenAI:
		collectEnvValue(env, "OPENAI_API_KEY", AsNonEmptyString(config.OpenAIAPIKey), overwrite)
		m := activeModel
		if m == "" {
			m = catalog.GetProviderDefaultModel("openai", cat)
		}
		collectEnvValue(env, "OPENAI_MODEL", m, overwrite)
		base := AsNonEmptyString(config.OpenAIBaseURL)
		if base == "" {
			base = DefaultOpenAIBaseURL
		}
		collectEnvValue(env, "OPENAI_BASE_URL", base, overwrite)
	case ProviderGemini:
		apiKey := AsNonEmptyString(config.GeminiAPIKey)
		base := firstNonEmpty(config.GeminiBaseURL, DefaultGeminiOpenAIBaseURL)
		m := activeModel
		if m == "" {
			m = catalog.GetProviderDefaultModel("gemini", cat)
		}
		collectOpenAICompatibleProvider(env, "GEMINI", apiKey, m, base, overwrite)
	case ProviderGrok:
		apiKey := firstNonEmpty(config.GrokAPIKey, config.XAIAPIKey)
		base := firstNonEmpty(config.GrokBaseURL, config.XAIBaseURL, DefaultGrokOpenAIBaseURL)
		m := activeModel
		if m == "" {
			m = catalog.GetProviderDefaultModel("grok", cat)
		}
		collectEnvValue(env, "GROK_API_KEY", AsNonEmptyString(config.GrokAPIKey), overwrite)
		collectEnvValue(env, "XAI_API_KEY", AsNonEmptyString(config.XAIAPIKey), overwrite)
		collectOpenAICompatibleProvider(env, "GROK", apiKey, m, base, overwrite)
	case ProviderCanopyWave:
		apiKey := AsNonEmptyString(config.CanopyWaveAPIKey)
		base := firstNonEmpty(config.CanopyWaveBaseURL, DefaultCanopyWaveOpenAIBaseURL)
		m := activeModel
		if m == "" {
			m = catalog.GetProviderDefaultModel("canopywave", cat)
		}
		collectOpenAICompatibleProvider(env, "CANOPYWAVE", apiKey, m, base, overwrite)
	case ProviderOpenRouter:
		apiKey := AsNonEmptyString(config.OpenRouterAPIKey)
		base := firstNonEmpty(config.OpenRouterBaseURL, DefaultOpenRouterOpenAIBaseURL)
		m := activeModel
		if m == "" {
			m = catalog.GetProviderDefaultModel("openrouter", cat)
		}
		collectOpenAICompatibleProvider(env, "OPENROUTER", apiKey, m, base, overwrite)
	case ProviderOllama:
		m := activeModel
		if m == "" {
			m = OllamaDefaultModel
		}
		collectEnvValue(env, "OPENAI_MODEL", m, overwrite)
		base := NormalizeOllamaOpenAIBaseURL(AsNonEmptyString(config.OllamaBaseURL))
		if base == "" {
			base = OllamaDefaultBaseURL
		}
		collectEnvValue(env, "OPENAI_BASE_URL", base, overwrite)
	case ProviderOpenCodeGo:
		apiKey := AsNonEmptyString(config.OpenCodeGoAPIKey)
		base := firstNonEmpty(config.OpenCodeGoBaseURL, DefaultOpenCodeGoBaseURL)
		m := activeModel
		if m == "" {
			m = catalog.GetProviderDefaultModel("opencodego", cat)
		}
		collectOpenAICompatibleProvider(env, "OPENCODEGO", apiKey, m, base, overwrite)
	}
	return env
}

// ApplyProviderEnvToProcess applies the env vars for a specific provider
// directly to the process environment via os.Setenv.
func ApplyProviderEnvToProcess(provider string, config *ProviderConfig, activeModel string, overwrite bool, cat *catalog.ModelCatalog) {
	for k, v := range ApplyProviderEnv(provider, config, activeModel, overwrite, cat) {
		_ = os.Setenv(k, v)
	}
}

// ApplyProviderConfigToEnv applies the full provider config to env vars.
// Returns the detected provider or empty string.
func ApplyProviderConfigToEnv(config *ProviderConfig, overwrite bool, cat *catalog.ModelCatalog) string {
	if config == nil {
		config = LoadProviderConfig("")
	}
	if config == nil {
		return ""
	}
	provider := DefaultProviderFromConfig(config)
	if provider == "" {
		return ""
	}
	if overwrite {
		ClearProviderRuntimeEnv()
	}
	activeModel := GetProviderActiveModel(config, provider)
	SetEnvValue("GRAYCODE_SMALL_FAST_MODEL", AsNonEmptyString(config.ExplorationModel), overwrite)
	ApplyProviderEnvToProcess(provider, config, activeModel, overwrite, cat)
	return provider
}
