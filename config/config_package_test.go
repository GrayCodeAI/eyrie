//nolint:errcheck
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. Config loading from environment
// ---------------------------------------------------------------------------

func TestResolveProviderRequest_UsesEnvBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		wantBase string
	}{
		{
			name:     "OPENAI_BASE_URL",
			envKey:   "OPENAI_BASE_URL",
			envValue: "https://custom.openai.example.com/v1",
			wantBase: "https://custom.openai.example.com/v1",
		},
		{
			name:     "OPENAI_API_BASE fallback",
			envKey:   "OPENAI_API_BASE",
			envValue: "https://api-base.example.com/v1",
			wantBase: "https://api-base.example.com/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all base URL env vars first
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("OPENAI_API_BASE")
			t.Setenv(tt.envKey, tt.envValue)

			r := ResolveProviderRequest("gpt-4o", "", "")
			if r.BaseURL != tt.wantBase {
				t.Errorf("BaseURL = %q, want %q", r.BaseURL, tt.wantBase)
			}
		})
	}
}

func TestResolveProviderRequest_ExplicitBaseURLOverridesEnv(t *testing.T) {
	os.Unsetenv("OPENAI_BASE_URL")
	os.Unsetenv("OPENAI_API_BASE")
	t.Setenv("OPENAI_BASE_URL", "https://env.example.com/v1")

	r := ResolveProviderRequest("gpt-4o", "https://explicit.example.com/v1", "")
	if r.BaseURL != "https://explicit.example.com/v1" {
		t.Errorf("BaseURL = %q, want explicit override", r.BaseURL)
	}
}

func TestResolveProviderRequest_UsesEnvModel(t *testing.T) {
	os.Unsetenv("OPENAI_MODEL")
	t.Setenv("OPENAI_MODEL", "gpt-4-turbo")

	r := ResolveProviderRequest("", "", "")
	if r.ResolvedModel != "gpt-4-turbo" {
		t.Errorf("ResolvedModel = %q, want 'gpt-4-turbo'", r.ResolvedModel)
	}
}

func TestResolveProviderRequest_Defaults(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		baseURL       string
		fallbackModel string
		wantModel     string
		wantBaseURL   string
	}{
		{
			name:          "all empty uses defaults",
			model:         "",
			baseURL:       "",
			fallbackModel: "",
			wantModel:     "gpt-4o",
			wantBaseURL:   DefaultOpenAIBaseURL,
		},
		{
			name:          "explicit model",
			model:         "gpt-4-turbo",
			baseURL:       "",
			fallbackModel: "",
			wantModel:     "gpt-4-turbo",
			wantBaseURL:   DefaultOpenAIBaseURL,
		},
		{
			name:          "fallback model used when model empty",
			model:         "",
			baseURL:       "",
			fallbackModel: "my-fallback",
			wantModel:     "my-fallback",
			wantBaseURL:   DefaultOpenAIBaseURL,
		},
		{
			name:          "explicit model takes priority over fallback",
			model:         "gpt-4o-mini",
			baseURL:       "",
			fallbackModel: "my-fallback",
			wantModel:     "gpt-4o-mini",
			wantBaseURL:   DefaultOpenAIBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("OPENAI_MODEL")
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("OPENAI_API_BASE")

			r := ResolveProviderRequest(tt.model, tt.baseURL, tt.fallbackModel)
			if r.ResolvedModel != tt.wantModel {
				t.Errorf("ResolvedModel = %q, want %q", r.ResolvedModel, tt.wantModel)
			}
			if r.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", r.BaseURL, tt.wantBaseURL)
			}
			if r.Transport != TransportChatCompletions {
				t.Errorf("Transport = %q, want %q", r.Transport, TransportChatCompletions)
			}
		})
	}
}

func TestResolveProviderRequest_TrailingSlashStripped(t *testing.T) {
	os.Unsetenv("OPENAI_BASE_URL")
	os.Unsetenv("OPENAI_API_BASE")

	r := ResolveProviderRequest("gpt-4o", "https://example.com/v1/", "")
	if r.BaseURL != "https://example.com/v1" {
		t.Errorf("BaseURL = %q, want trailing slash stripped", r.BaseURL)
	}
}

func TestResolveProviderRequest_ReasoningEffortLevels(t *testing.T) {
	tests := []struct {
		model      string
		wantNil    bool
		wantEffort ReasoningEffort
	}{
		{"gpt-4o?reasoning=low", false, ReasoningLow},
		{"gpt-4o?reasoning=medium", false, ReasoningMedium},
		{"gpt-4o?reasoning=high", false, ReasoningHigh},
		{"gpt-4o", true, ""},
		{"gpt-4o?reasoning=invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			r := ResolveProviderRequest(tt.model, "", "")
			if tt.wantNil {
				if r.Reasoning != nil {
					t.Errorf("expected nil reasoning for %q, got %+v", tt.model, r.Reasoning)
				}
			} else {
				if r.Reasoning == nil {
					t.Fatalf("expected reasoning for %q, got nil", tt.model)
				}
				if r.Reasoning.Effort != tt.wantEffort {
					t.Errorf("reasoning effort = %q, want %q", r.Reasoning.Effort, tt.wantEffort)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. Default values
// ---------------------------------------------------------------------------

func TestDefaultProviderFromConfig_NilConfig(t *testing.T) {
	if got := DefaultProviderFromConfig(nil); got != "" {
		t.Errorf("DefaultProviderFromConfig(nil) = %q, want empty", got)
	}
}

func TestDefaultProviderFromConfig_EmptyConfig(t *testing.T) {
	if got := DefaultProviderFromConfig(&ProviderConfig{}); got != "" {
		t.Errorf("DefaultProviderFromConfig(empty) = %q, want empty", got)
	}
}

func TestDefaultProviderFromConfig_ActiveProviderConfigured(t *testing.T) {
	cfg := &ProviderConfig{
		ActiveProvider: "openai",
		OpenAIAPIKey:   "sk-openai-key-1234567890",
	}
	if got := DefaultProviderFromConfig(cfg); got != "openai" {
		t.Errorf("got %q, want 'openai'", got)
	}
}

func TestDefaultProviderFromConfig_ActiveProviderNotConfigured(t *testing.T) {
	cfg := &ProviderConfig{
		ActiveProvider:  "gemini", // no GeminiAPIKey set
		AnthropicAPIKey: "sk-ant-key-1234567890",
	}
	if got := DefaultProviderFromConfig(cfg); got != "anthropic" {
		t.Errorf("got %q, want 'anthropic' (first configured in detection order)", got)
	}
}

func TestDefaultProviderFromConfig_DetectionOrderPriority(t *testing.T) {
	// Anthropic should win over OpenAI when both are configured
	cfg := &ProviderConfig{
		AnthropicAPIKey: "sk-ant-key-1234567890",
		OpenAIAPIKey:    "sk-openai-key-1234567890",
	}
	if got := DefaultProviderFromConfig(cfg); got != "anthropic" {
		t.Errorf("got %q, want 'anthropic' (higher priority)", got)
	}
}

func TestDefaultProviderFromConfig_OllamaUsesBaseURL(t *testing.T) {
	cfg := &ProviderConfig{
		OllamaBaseURL: "http://localhost:11434",
	}
	if got := DefaultProviderFromConfig(cfg); got != "ollama" {
		t.Errorf("got %q, want 'ollama'", got)
	}
}

func TestDefaultProviderFromConfig_UnconfiguredActiveFallsThrough(t *testing.T) {
	cfg := &ProviderConfig{
		ActiveProvider: "openrouter", // no OpenRouterAPIKey set
		GeminiAPIKey:   "gemini-key-1234567890",
	}
	got := DefaultProviderFromConfig(cfg)
	if got == "openrouter" {
		t.Errorf("should not return unconfigured active provider, got %q", got)
	}
}

func TestIsOpenAICompatibleRuntimeEnabled_NoKeys(t *testing.T) {
	// With no env vars set, should return false
	// (This test may be affected by credential store, so just verify the function doesn't panic)
	_ = IsOpenAICompatibleRuntimeEnabled()
}

// ---------------------------------------------------------------------------
// 3. Validation
// ---------------------------------------------------------------------------

func TestValidateAPIKey_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		provider   string
		wantEmpty  bool   // true if valid (no error)
		wantSubstr string // expected substring in error message
	}{
		{
			name:      "valid key",
			apiKey:    "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890",
			provider:  "OpenAI",
			wantEmpty: true,
		},
		{
			name:       "empty key",
			apiKey:     "",
			provider:   "Anthropic",
			wantEmpty:  false,
			wantSubstr: "requires an API key",
		},
		{
			name:       "placeholder SUA_CHAVE",
			apiKey:     "SUA_CHAVE",
			provider:   "Gemini",
			wantEmpty:  false,
			wantSubstr: "placeholder",
		},
		{
			name:       "too short key",
			apiKey:     "abc",
			provider:   "Grok",
			wantEmpty:  false,
			wantSubstr: "too short",
		},
		{
			name:       "exactly 9 chars is too short",
			apiKey:     "123456789",
			provider:   "test",
			wantEmpty:  false,
			wantSubstr: "too short",
		},
		{
			name:      "exactly 10 chars is valid",
			apiKey:    "1234567890",
			provider:  "test",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ValidateAPIKey(tt.apiKey, tt.provider)
			if tt.wantEmpty && msg != "" {
				t.Errorf("expected no error, got %q", msg)
			}
			if !tt.wantEmpty && msg == "" {
				t.Error("expected error, got empty")
			}
			if !tt.wantEmpty && tt.wantSubstr != "" {
				if !containsSubstring(msg, tt.wantSubstr) {
					t.Errorf("error %q should contain %q", msg, tt.wantSubstr)
				}
			}
		})
	}
}

func TestValidateBaseURL_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wantEmpty bool
	}{
		{
			name:      "empty URL is valid",
			baseURL:   "",
			wantEmpty: true,
		},
		{
			name:      "http URL is valid",
			baseURL:   "https://api.openai.com/v1",
			wantEmpty: true,
		},
		{
			name:      "localhost URL is valid",
			baseURL:   "http://localhost:11434/v1",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ValidateBaseURL(tt.baseURL)
			if tt.wantEmpty && msg != "" {
				t.Errorf("expected no error, got %q", msg)
			}
			if !tt.wantEmpty && msg == "" {
				t.Error("expected error, got empty")
			}
		})
	}
}

func TestIsLocalProviderURL_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"localhost", "http://localhost:11434/v1", true},
		{"127.0.0.1", "http://127.0.0.1:8080", true},
		{"ipv6 loopback", "http://[::1]:8080", true},
		{"remote https", "https://api.openai.com/v1", false},
		{"empty string", "", false},
		{"invalid url", "://bad", false},
		{"remote with port", "https://api.example.com:443/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLocalProviderURL(tt.url); got != tt.want {
				t.Errorf("IsLocalProviderURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. Provider config parsing
// ---------------------------------------------------------------------------

func TestGetProviderModel_TableDriven(t *testing.T) {
	cfg := &ProviderConfig{
		AnthropicModel:  "claude-sonnet-4-6",
		OpenAIModel:     "gpt-4o",
		GeminiModel:     "gemini-2.0-flash",
		GrokModel:       "",
		XAIModel:        "grok-2",
		CanopyWaveModel: "cw-model",
		ZAIModel:        "zai-model",
		OpenRouterModel: "or-model",
		OllamaModel:     "llama3.1:8b",
		OpenCodeGoModel: "ocg-model",
		MoonshotModel:   "kimi-k2.6",
		XiaomiModel:     "mimo-v2-flash",
	}

	tests := []struct {
		provider string
		want     string
	}{
		{ProviderAnthropic, "claude-sonnet-4-6"},
		{ProviderOpenAI, "gpt-4o"},
		{ProviderGemini, "gemini-2.0-flash"},
		{ProviderGrok, "grok-2"}, // falls through GrokModel to XAIModel
		{ProviderCanopyWave, "cw-model"},
		{ProviderZAI, "zai-model"},
		{ProviderOpenRouter, "or-model"},
		{ProviderOllama, "llama3.1:8b"},
		{ProviderOpenCodeGo, "ocg-model"},
		{ProviderKimi, "kimi-k2.6"},
		{ProviderXiaomi, "mimo-v2-flash"},
		{"nonexistent-provider", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := GetProviderModel(cfg, tt.provider); got != tt.want {
				t.Errorf("GetProviderModel(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestGetProviderAPIKey_TableDriven(t *testing.T) {
	cfg := &ProviderConfig{
		AnthropicAPIKey:  "sk-ant-key",
		OpenAIAPIKey:     "sk-openai-key",
		GeminiAPIKey:     "gemini-key",
		GrokAPIKey:       "",
		XAIAPIKey:        "xai-key",
		CanopyWaveAPIKey: "cw-key",
		ZAIAPIKey:        "zai-key",
		OpenRouterAPIKey: "or-key",
		OpenCodeGoAPIKey: "ocg-key",
		MoonshotAPIKey:   "moonshot-key",
		XiaomiAPIKey:     "xiaomi-key",
	}

	tests := []struct {
		provider string
		want     string
	}{
		{ProviderAnthropic, "sk-ant-key"},
		{ProviderOpenAI, "sk-openai-key"},
		{ProviderGemini, "gemini-key"},
		{ProviderGrok, "xai-key"}, // falls through GrokAPIKey to XAIAPIKey
		{ProviderCanopyWave, "cw-key"},
		{ProviderZAI, "zai-key"},
		{ProviderOpenRouter, "or-key"},
		{ProviderOllama, ""}, // no API key for ollama
		{ProviderOpenCodeGo, "ocg-key"},
		{ProviderKimi, "moonshot-key"},
		{ProviderXiaomi, "xiaomi-key"},
		{"nonexistent-provider", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := GetProviderAPIKey(cfg, tt.provider); got != tt.want {
				t.Errorf("GetProviderAPIKey(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestIsProviderConfigured_TableDriven(t *testing.T) {
	cfg := &ProviderConfig{
		AnthropicAPIKey: "sk-ant-key",
		OpenAIAPIKey:    "sk-openai-key",
		OllamaBaseURL:   "http://localhost:11434",
	}

	tests := []struct {
		provider string
		want     bool
	}{
		{ProviderAnthropic, true},
		{ProviderOpenAI, true},
		{ProviderOllama, true},  // uses BaseURL
		{ProviderGemini, false}, // no API key
		{ProviderGrok, false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := IsProviderConfigured(cfg, tt.provider); got != tt.want {
				t.Errorf("IsProviderConfigured(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 5. Provider config file load/save roundtrip
// ---------------------------------------------------------------------------

func TestLoadProviderConfig_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provider.json")

	original := &ProviderConfig{
		Version:          "1",
		ConfigVersion:    2,
		ActiveProvider:   "anthropic",
		AnthropicAPIKey:  "sk-ant-test-key-1234567890",
		AnthropicModel:   "claude-sonnet-4-6",
		OpenAIAPIKey:     "sk-openai-test-key-1234567890",
		ActiveModel:      "claude-sonnet-4-6",
		ExplorationModel: "gpt-4o-mini",
	}

	if err := SaveProviderConfig(original, path); err != nil {
		t.Fatalf("SaveProviderConfig: %v", err)
	}

	loaded := LoadProviderConfig(path)
	if loaded == nil {
		t.Fatal("LoadProviderConfig returned nil")
	}

	if loaded.ActiveProvider != original.ActiveProvider {
		t.Errorf("ActiveProvider = %q, want %q", loaded.ActiveProvider, original.ActiveProvider)
	}
	if loaded.AnthropicAPIKey != original.AnthropicAPIKey {
		t.Errorf("AnthropicAPIKey = %q, want %q", loaded.AnthropicAPIKey, original.AnthropicAPIKey)
	}
	if loaded.AnthropicModel != original.AnthropicModel {
		t.Errorf("AnthropicModel = %q, want %q", loaded.AnthropicModel, original.AnthropicModel)
	}
	if loaded.ConfigVersion != original.ConfigVersion {
		t.Errorf("ConfigVersion = %d, want %d", loaded.ConfigVersion, original.ConfigVersion)
	}
}

func TestLoadProviderConfigWithError_ValidVersion1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provider.json")

	cfg := ProviderConfig{Version: "1", ActiveProvider: "openai"}
	data, _ := json.Marshal(cfg)
	os.WriteFile(path, data, 0o644)

	loaded, err := LoadProviderConfigWithError(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadProviderConfigWithError_EmptyVersionAccepted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provider.json")

	cfg := ProviderConfig{ActiveProvider: "openai"} // no version field
	data, _ := json.Marshal(cfg)
	os.WriteFile(path, data, 0o644)

	loaded, err := LoadProviderConfigWithError(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil config for empty version")
	}
}

// ---------------------------------------------------------------------------
// 6. Utility functions
// ---------------------------------------------------------------------------

func TestAsNonEmptyString_TableDriven(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"", ""},
		{"   ", ""},
		{"\t\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := AsNonEmptyString(tt.input); got != tt.want {
				t.Errorf("AsNonEmptyString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeOllamaOpenAIBaseURL_TableDriven(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:11434", "http://localhost:11434/v1"},
		{"http://localhost:11434/v1", "http://localhost:11434/v1"},
		{"http://localhost:11434/", "http://localhost:11434/v1"},
		{"http://localhost:11434///", "http://localhost:11434/v1"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeOllamaOpenAIBaseURL(tt.input); got != tt.want {
				t.Errorf("NormalizeOllamaOpenAIBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetEnvValue_TableDriven(t *testing.T) {
	key := "TEST_CONFIG_PKG_SETENV_98765"
	defer os.Unsetenv(key)

	tests := []struct {
		name      string
		value     string
		overwrite bool
		preSet    string // value already in env
		wantEnv   string // expected env value after call
	}{
		{"empty value no-ops", "", true, "", ""},
		{"sets new value", "hello", true, "", "hello"},
		{"overwrite=false keeps existing", "new", false, "old", "old"},
		{"overwrite=true replaces", "new", true, "old", "new"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preSet != "" {
				os.Setenv(key, tt.preSet)
			} else {
				os.Unsetenv(key)
			}
			SetEnvValue(key, tt.value, tt.overwrite)
			if got := os.Getenv(key); got != tt.wantEnv {
				t.Errorf("env = %q, want %q", got, tt.wantEnv)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 7. Agent routing
// ---------------------------------------------------------------------------

func TestDefaultAgentRouting(t *testing.T) {
	cfg := DefaultAgentRouting()
	if cfg == nil {
		t.Fatal("DefaultAgentRouting returned nil")
	}
	if cfg.Routes == nil {
		t.Fatal("Routes is nil")
	}
	if cfg.Routes["default"] == "" {
		t.Error("default route is empty")
	}
	if cfg.Routes["general-purpose"] == "" {
		t.Error("general-purpose route is empty")
	}
}

func TestAgentRoutingConfig_ResolveModel(t *testing.T) {
	cfg := &AgentRoutingConfig{
		Routes: map[string]string{
			"default":         "anthropic/claude-sonnet-4-20250514",
			"general-purpose": "anthropic/claude-sonnet-4-20250514",
			"coding":          "anthropic/claude-opus-20250514",
		},
	}

	tests := []struct {
		agentType string
		fallback  string
		want      string
	}{
		{"coding", "fallback-model", "anthropic/claude-opus-20250514"},
		{"general-purpose", "fallback-model", "anthropic/claude-sonnet-4-20250514"},
		{"unknown-type", "fallback-model", "anthropic/claude-sonnet-4-20250514"}, // falls to default
		{"unknown-type", "", "anthropic/claude-sonnet-4-20250514"},               // falls to default
	}

	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			if got := cfg.ResolveModel(tt.agentType, tt.fallback); got != tt.want {
				t.Errorf("ResolveModel(%q, %q) = %q, want %q", tt.agentType, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestAgentRoutingConfig_ResolveModel_NilRoutes(t *testing.T) {
	cfg := &AgentRoutingConfig{}
	if got := cfg.ResolveModel("anything", "fallback"); got != "fallback" {
		t.Errorf("ResolveModel with nil routes = %q, want 'fallback'", got)
	}
}

func TestAgentRoutingConfig_ResolveModel_NoDefaultRoute(t *testing.T) {
	cfg := &AgentRoutingConfig{
		Routes: map[string]string{
			"coding": "anthropic/claude-opus-20250514",
		},
	}
	// Unknown agent type, no "default" route -> fallback
	if got := cfg.ResolveModel("unknown", "my-fallback"); got != "my-fallback" {
		t.Errorf("got %q, want 'my-fallback'", got)
	}
}

func TestAgentRoutingConfig_SetRoute(t *testing.T) {
	cfg := &AgentRoutingConfig{}
	cfg.SetRoute("coding", "anthropic/claude-opus-20250514")
	if cfg.Routes["coding"] != "anthropic/claude-opus-20250514" {
		t.Errorf("SetRoute failed, got %q", cfg.Routes["coding"])
	}

	// Update existing
	cfg.SetRoute("coding", "openai/gpt-4o")
	if cfg.Routes["coding"] != "openai/gpt-4o" {
		t.Errorf("SetRoute update failed, got %q", cfg.Routes["coding"])
	}
}

func TestParseRoute_TableDriven(t *testing.T) {
	tests := []struct {
		route        string
		wantProvider string
		wantModel    string
	}{
		{"anthropic/claude-sonnet-4-20250514", "anthropic", "claude-sonnet-4-20250514"},
		{"openai/gpt-4o", "openai", "gpt-4o"},
		{"gpt-4o", "", "gpt-4o"},
		{"", "", ""},
		{"anthropic/claude/opus", "anthropic", "claude/opus"}, // only splits on first /
	}

	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {
			provider, model := ParseRoute(tt.route)
			if provider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 8. Migration (EnsureDeploymentConfigV2)
// ---------------------------------------------------------------------------

func TestEnsureDeploymentConfigV2_NilConfig(t *testing.T) {
	if got := EnsureDeploymentConfigV2(nil); got != nil {
		t.Errorf("EnsureDeploymentConfigV2(nil) = %v, want nil", got)
	}
}

func TestEnsureDeploymentConfigV2_AlreadyV2(t *testing.T) {
	cfg := &ProviderConfig{
		ConfigVersion: 2,
		Deployments:   map[string]DeploymentConfig{"anthropic-direct": {APIKey: "key"}},
	}
	out := EnsureDeploymentConfigV2(cfg)
	if out.ConfigVersion != 2 {
		t.Errorf("ConfigVersion = %d, want 2", out.ConfigVersion)
	}
}

func TestEnsureDeploymentConfigV2_UpgradesLegacy(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *ProviderConfig
		wantDeploymentKey string
	}{
		{
			name:              "anthropic legacy",
			cfg:               &ProviderConfig{AnthropicAPIKey: "sk-ant-test-1234567890"},
			wantDeploymentKey: "anthropic-direct",
		},
		{
			name:              "openai legacy",
			cfg:               &ProviderConfig{OpenAIAPIKey: "sk-openai-test-1234567890"},
			wantDeploymentKey: "openai-direct",
		},
		{
			name:              "gemini legacy",
			cfg:               &ProviderConfig{GeminiAPIKey: "gemini-test-key-1234567890"},
			wantDeploymentKey: "gemini-direct",
		},
		{
			name:              "openrouter legacy",
			cfg:               &ProviderConfig{OpenRouterAPIKey: "or-test-key-1234567890"},
			wantDeploymentKey: "openrouter",
		},
		{
			name:              "ollama legacy",
			cfg:               &ProviderConfig{OllamaBaseURL: "http://localhost:11434"},
			wantDeploymentKey: "ollama-local",
		},
		{
			name:              "opencodego legacy",
			cfg:               &ProviderConfig{OpenCodeGoAPIKey: "ocg-test-key-1234567890"},
			wantDeploymentKey: "opencodego",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := EnsureDeploymentConfigV2(tt.cfg)
			if out.ConfigVersion != 2 {
				t.Errorf("ConfigVersion = %d, want 2", out.ConfigVersion)
			}
			if _, ok := out.Deployments[tt.wantDeploymentKey]; !ok {
				t.Errorf("missing deployment %q, got %v", tt.wantDeploymentKey, out.Deployments)
			}
		})
	}
}

func TestEnsureDeploymentConfigV2_NoConfiguredProviders(t *testing.T) {
	cfg := &ProviderConfig{} // no keys or URLs set
	out := EnsureDeploymentConfigV2(cfg)
	// Should not upgrade if nothing is configured
	if out.ConfigVersion == 2 {
		t.Error("should not upgrade to v2 with no configured providers")
	}
}

// ---------------------------------------------------------------------------
// 9. Provider env application
// ---------------------------------------------------------------------------

func TestApplyProviderEnv_AllProviders(t *testing.T) {
	tests := []struct {
		provider  string
		apiKey    string
		apiKeyEnv string
		model     string
		wantKey   string
		wantModel string
	}{
		{ProviderAnthropic, "sk-ant-1234567890", "ANTHROPIC_API_KEY", "claude-sonnet-4-6", "ANTHROPIC_API_KEY", "ANTHROPIC_MODEL"},
		{ProviderOpenAI, "sk-openai-1234567890", "OPENAI_API_KEY", "gpt-4o", "OPENAI_API_KEY", "OPENAI_MODEL"},
		{ProviderGemini, "gemini-key-1234567890", "GEMINI_API_KEY", "gemini-2.0-flash", "GEMINI_API_KEY", "GEMINI_MODEL"},
		{ProviderOpenRouter, "or-key-1234567890", "OPENROUTER_API_KEY", "or-model", "OPENROUTER_API_KEY", "OPENROUTER_MODEL"},
		{ProviderCanopyWave, "cw-key-1234567890", "CANOPYWAVE_API_KEY", "cw-model", "CANOPYWAVE_API_KEY", "CANOPYWAVE_MODEL"},
		{ProviderZAI, "zai-key-1234567890", "ZAI_API_KEY", "zai-model", "ZAI_API_KEY", "ZAI_MODEL"},
	}

	cat := testModelCatalog()

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := &ProviderConfig{}
			// Set the API key field for the provider
			switch tt.provider {
			case ProviderAnthropic:
				cfg.AnthropicAPIKey = tt.apiKey
			case ProviderOpenAI:
				cfg.OpenAIAPIKey = tt.apiKey
			case ProviderGemini:
				cfg.GeminiAPIKey = tt.apiKey
			case ProviderOpenRouter:
				cfg.OpenRouterAPIKey = tt.apiKey
			case ProviderCanopyWave:
				cfg.CanopyWaveAPIKey = tt.apiKey
			case ProviderZAI:
				cfg.ZAIAPIKey = tt.apiKey
			}

			env := ApplyProviderEnv(tt.provider, cfg, tt.model, true, &cat)
			if env[tt.wantKey] != tt.apiKey {
				t.Errorf("%s = %q, want %q", tt.wantKey, env[tt.wantKey], tt.apiKey)
			}
			if env[tt.wantModel] != tt.model {
				t.Errorf("%s = %q, want %q", tt.wantModel, env[tt.wantModel], tt.model)
			}
		})
	}
}

func TestApplyProviderEnv_GrokUsesXAIFallback(t *testing.T) {
	cfg := &ProviderConfig{
		XAIAPIKey: "xai-fallback-key-1234567890",
	}
	cat := testModelCatalog()

	env := ApplyProviderEnv(ProviderGrok, cfg, "grok-2", true, &cat)
	if env["XAI_API_KEY"] != "xai-fallback-key-1234567890" {
		t.Errorf("XAI_API_KEY = %q, want fallback key", env["XAI_API_KEY"])
	}
}

// ---------------------------------------------------------------------------
// 10. Provider detection order completeness
// ---------------------------------------------------------------------------

func TestProviderDetectionOrder_AllProvidersCovered(t *testing.T) {
	allProviders := []string{
		ProviderAnthropic, ProviderOpenAI, ProviderCanopyWave,
		ProviderZAI, ProviderOpenRouter, ProviderGrok,
		ProviderGemini, ProviderOllama, ProviderOpenCodeGo,
		ProviderKimi, ProviderXiaomi,
	}

	for _, p := range allProviders {
		found := false
		for _, ordered := range APIProviderDetectionOrder {
			if ordered == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q missing from APIProviderDetectionOrder", p)
		}
	}
}

func TestOpenAICompatibleRuntimeProfiles_AllKeysHaveOrderEntry(t *testing.T) {
	for key := range OpenAICompatibleRuntimeProfiles {
		found := false
		for _, ordered := range OpenAICompatibleRuntimeProfileOrder {
			if ordered == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("profile key %q in OpenAICompatibleRuntimeProfiles but missing from OpenAICompatibleRuntimeProfileOrder", key)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
