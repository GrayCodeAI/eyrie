package config

// APIProvider is the type for supported LLM providers.
type APIProvider = string

const (
	ProviderAnthropic  APIProvider = "anthropic"
	ProviderOpenAI     APIProvider = "openai"
	ProviderCanopyWave APIProvider = "canopywave"
	ProviderOpenRouter APIProvider = "openrouter"
	ProviderGrok       APIProvider = "grok"
	ProviderGemini     APIProvider = "gemini"
	ProviderOllama     APIProvider = "ollama"
	ProviderOpenCodeGo APIProvider = "opencodego"
)

// RuntimeProviderProfile defines how a provider is detected and configured at runtime.
type RuntimeProviderProfile struct {
	Mode           string       `json:"mode"`
	DefaultBaseURL string       `json:"default_base_url"`
	DefaultModel   string       `json:"default_model"`
	DetectionEnv   []string     `json:"detection_env"`
	ModelEnv       []string     `json:"model_env"`
	BaseURLEnv     []string     `json:"base_url_env"`
	APIKeys        []APIKeyDef  `json:"api_keys"`
}

// APIKeyDef maps an env var to a key source name.
type APIKeyDef struct {
	Env    string `json:"env"`
	Source string `json:"source"`
}

// Provider runtime profiles.
var (
	AnthropicRuntimeProfile = RuntimeProviderProfile{
		Mode: "anthropic", DefaultBaseURL: DefaultAnthropicOpenAIBaseURL, DefaultModel: "claude-3-5-sonnet-latest",
		DetectionEnv: []string{"ANTHROPIC_API_KEY"},
		ModelEnv:     []string{"ANTHROPIC_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"ANTHROPIC_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "ANTHROPIC_API_KEY", Source: "anthropic"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	OpenAIRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: DefaultOpenAIBaseURL, DefaultModel: "gpt-4o",
		DetectionEnv: []string{"OPENAI_API_KEY"},
		ModelEnv:     []string{"OPENAI_MODEL"},
		BaseURLEnv:   []string{"OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	GrokRuntimeProfile = RuntimeProviderProfile{
		Mode: "grok", DefaultBaseURL: DefaultGrokOpenAIBaseURL, DefaultModel: "grok-2",
		DetectionEnv: []string{"GROK_API_KEY", "XAI_API_KEY"},
		ModelEnv:     []string{"GROK_MODEL", "XAI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"GROK_BASE_URL", "XAI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "GROK_API_KEY", Source: "grok"}, {Env: "XAI_API_KEY", Source: "xai"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	GeminiRuntimeProfile = RuntimeProviderProfile{
		Mode: "gemini", DefaultBaseURL: DefaultGeminiOpenAIBaseURL, DefaultModel: "gemini-2.0-flash",
		DetectionEnv: []string{"GEMINI_API_KEY"},
		ModelEnv:     []string{"GEMINI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"GEMINI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "GEMINI_API_KEY", Source: "gemini"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	OpenRouterRuntimeProfile = RuntimeProviderProfile{
		Mode: "openrouter", DefaultBaseURL: DefaultOpenRouterOpenAIBaseURL, DefaultModel: "openai/gpt-4o-mini",
		DetectionEnv: []string{"OPENROUTER_API_KEY"},
		ModelEnv:     []string{"OPENROUTER_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"OPENROUTER_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "OPENROUTER_API_KEY", Source: "openrouter"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	CanopyWaveRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: DefaultCanopyWaveOpenAIBaseURL, DefaultModel: "zai/glm-4.6",
		DetectionEnv: []string{"CANOPYWAVE_API_KEY"},
		ModelEnv:     []string{"CANOPYWAVE_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"CANOPYWAVE_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "CANOPYWAVE_API_KEY", Source: "canopywave"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	OpenCodeGoRuntimeProfile = RuntimeProviderProfile{
		Mode: "opencodego", DefaultBaseURL: DefaultOpenCodeGoBaseURL, DefaultModel: "kimi-k2.5",
		DetectionEnv: []string{"OPENCODEGO_API_KEY"},
		ModelEnv:     []string{"OPENCODEGO_MODEL"},
		BaseURLEnv:   []string{"OPENCODEGO_BASE_URL"},
		APIKeys:      []APIKeyDef{{Env: "OPENCODEGO_API_KEY", Source: "opencodego"}},
	}
)

// APIProviderDetectionOrder is the priority order for provider detection.
var APIProviderDetectionOrder = []APIProvider{
	ProviderAnthropic, ProviderOpenRouter, ProviderGrok, ProviderGemini,
	ProviderCanopyWave, ProviderOpenAI, ProviderOpenCodeGo, ProviderOllama,
}

// ProviderModelEnvKeys maps each provider to its model env var keys.
var ProviderModelEnvKeys = map[APIProvider][]string{
	ProviderAnthropic:  AnthropicRuntimeProfile.ModelEnv,
	ProviderOpenAI:     OpenAIRuntimeProfile.ModelEnv,
	ProviderCanopyWave: CanopyWaveRuntimeProfile.ModelEnv,
	ProviderOpenRouter: OpenRouterRuntimeProfile.ModelEnv,
	ProviderGrok:       GrokRuntimeProfile.ModelEnv,
	ProviderGemini:     GeminiRuntimeProfile.ModelEnv,
	ProviderOllama:     {"OLLAMA_MODEL", "OPENAI_MODEL"},
	ProviderOpenCodeGo: OpenCodeGoRuntimeProfile.ModelEnv,
}

const (
	OllamaDefaultBaseURL  = "http://localhost:11434/v1"
	OllamaDefaultModel    = "llama3.1:8b"
	OpenCodeGoDefaultBaseURL = "https://opencode.ai/zen/go/v1"
	OpenCodeGoDefaultModel   = "kimi-k2.5"
)

// OpenAICompatibleRuntimeProfileOrder is the detection order for runtime profiles.
var OpenAICompatibleRuntimeProfileOrder = []string{
	"openrouter", "grok", "gemini", "anthropic", "canopywave", "openai", "opencodego",
}

// OpenAICompatibleRuntimeProfiles maps profile key to its runtime profile.
var OpenAICompatibleRuntimeProfiles = map[string]RuntimeProviderProfile{
	"anthropic":  AnthropicRuntimeProfile,
	"grok":       GrokRuntimeProfile,
	"gemini":     GeminiRuntimeProfile,
	"canopywave": CanopyWaveRuntimeProfile,
	"openai":     OpenAIRuntimeProfile,
	"openrouter": OpenRouterRuntimeProfile,
	"opencodego": OpenCodeGoRuntimeProfile,
}
