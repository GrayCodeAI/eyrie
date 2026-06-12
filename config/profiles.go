package config

// APIProvider is the type for supported LLM providers.
type APIProvider = string

const (
	ProviderAnthropic           APIProvider = "anthropic"
	ProviderOpenAI              APIProvider = "openai"
	ProviderAzure               APIProvider = "azure"
	ProviderCanopyWave          APIProvider = "canopywave"
	ProviderDeepSeek            APIProvider = "deepseek"
	ProviderZAI                 APIProvider = "z-ai"
	ProviderOpenRouter          APIProvider = "openrouter"
	ProviderGrok                APIProvider = "grok"
	ProviderGemini              APIProvider = "gemini"
	ProviderBedrock             APIProvider = "bedrock"
	ProviderVertex              APIProvider = "vertex"
	ProviderOllama              APIProvider = "ollama"
	ProviderOpenCodeGo          APIProvider = "opencodego"
	ProviderKimi                APIProvider = "kimi"
	ProviderXiaomiMimoPayg      APIProvider = "xiaomi_mimo_payg"
	ProviderXiaomiMimoTokenPlan APIProvider = "xiaomi_mimo_token_plan"
)

// RuntimeProviderProfile defines how a provider is detected and configured at runtime.
type RuntimeProviderProfile struct {
	Mode           string      `json:"mode"`
	DefaultBaseURL string      `json:"default_base_url"`
	DefaultModel   string      `json:"default_model"`
	DetectionEnv   []string    `json:"detection_env"`
	ModelEnv       []string    `json:"model_env"`
	BaseURLEnv     []string    `json:"base_url_env"`
	APIKeys        []APIKeyDef `json:"api_keys"`
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
		DetectionEnv: []string{"XAI_API_KEY"},
		ModelEnv:     []string{"XAI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"XAI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "XAI_API_KEY", Source: "grok"}},
	}
	GeminiRuntimeProfile = RuntimeProviderProfile{
		Mode: "gemini", DefaultBaseURL: DefaultGeminiOpenAIBaseURL, DefaultModel: "gemini-2.0-flash",
		DetectionEnv: []string{"GEMINI_API_KEY"},
		ModelEnv:     []string{"GEMINI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"GEMINI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "GEMINI_API_KEY", Source: "gemini"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	VertexRuntimeProfile = RuntimeProviderProfile{
		Mode: "gemini-vertex", DefaultBaseURL: "", DefaultModel: "gemini-2.0-flash",
		DetectionEnv: []string{"VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"},
		ModelEnv:     []string{"VERTEX_MODEL", "GEMINI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"VERTEX_PROJECT_ID", "VERTEX_REGION"},
		APIKeys:      []APIKeyDef{{Env: "VERTEX_ACCESS_TOKEN", Source: "vertex"}, {Env: "GOOGLE_OAUTH_ACCESS_TOKEN", Source: "google"}},
	}
	AzureRuntimeProfile = RuntimeProviderProfile{
		Mode: "azure", DefaultBaseURL: "", DefaultModel: "",
		DetectionEnv: []string{"AZURE_OPENAI_API_KEY"},
		ModelEnv:     []string{"AZURE_OPENAI_DEPLOYMENT", "AZURE_OPENAI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"AZURE_OPENAI_ENDPOINT"},
		APIKeys:      []APIKeyDef{{Env: "AZURE_OPENAI_API_KEY", Source: "azure"}},
	}
	BedrockRuntimeProfile = RuntimeProviderProfile{
		Mode: "bedrock", DefaultBaseURL: "", DefaultModel: "",
		DetectionEnv: []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
		ModelEnv:     []string{"BEDROCK_MODEL", "ANTHROPIC_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"AWS_REGION", "AWS_DEFAULT_REGION"},
		APIKeys:      []APIKeyDef{{Env: "AWS_SECRET_ACCESS_KEY", Source: "bedrock"}, {Env: "AWS_ACCESS_KEY_ID", Source: "bedrock"}},
	}
	OpenRouterRuntimeProfile = RuntimeProviderProfile{
		Mode: "openrouter", DefaultBaseURL: DefaultOpenRouterOpenAIBaseURL,
		DetectionEnv: []string{"OPENROUTER_API_KEY"},
		ModelEnv:     []string{"OPENROUTER_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"OPENROUTER_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "OPENROUTER_API_KEY", Source: "openrouter"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	ZAIRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: DefaultZAIOpenAIBaseURL,
		DetectionEnv: []string{"ZAI_API_KEY"},
		ModelEnv:     []string{"ZAI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"ZAI_BASE_URL", "ZAI_API_BASE", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "ZAI_API_KEY", Source: "z-ai"}},
	}
	CanopyWaveRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: DefaultCanopyWaveOpenAIBaseURL,
		DetectionEnv: []string{"CANOPYWAVE_API_KEY"},
		ModelEnv:     []string{"CANOPYWAVE_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"CANOPYWAVE_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "CANOPYWAVE_API_KEY", Source: "canopywave"}, {Env: "OPENAI_API_KEY", Source: "openai"}},
	}
	DeepSeekRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: "https://api.deepseek.com/v1", DefaultModel: "deepseek-v4-flash",
		DetectionEnv: []string{"DEEPSEEK_API_KEY"},
		ModelEnv:     []string{"DEEPSEEK_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"DEEPSEEK_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "DEEPSEEK_API_KEY", Source: "deepseek"}},
	}
	OpenCodeGoRuntimeProfile = RuntimeProviderProfile{
		Mode: "opencodego", DefaultBaseURL: DefaultOpenCodeGoBaseURL, DefaultModel: "kimi-k2.5",
		DetectionEnv: []string{"OPENCODEGO_API_KEY"},
		ModelEnv:     []string{"OPENCODEGO_MODEL"},
		BaseURLEnv:   []string{"OPENCODEGO_BASE_URL"},
		APIKeys:      []APIKeyDef{{Env: "OPENCODEGO_API_KEY", Source: "opencodego"}},
	}
	KimiRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: DefaultKimiOpenAIBaseURL, DefaultModel: "kimi-k2.6",
		DetectionEnv: []string{"MOONSHOT_API_KEY"},
		ModelEnv:     []string{"MOONSHOT_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{"MOONSHOT_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: "MOONSHOT_API_KEY", Source: "kimi"}},
	}
	XiaomiPaygRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: DefaultXiaomiOpenAIBaseURL, DefaultModel: "mimo-v2.5-pro",
		DetectionEnv: []string{EnvXiaomiPaygAPIKey},
		ModelEnv:     []string{"XIAOMI_MIMO_PAYG_MODEL", "XIAOMI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{EnvXiaomiPaygBaseURL, "XIAOMI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: EnvXiaomiPaygAPIKey, Source: "xiaomi_mimo_payg"}},
	}
	XiaomiTokenPlanRuntimeProfile = RuntimeProviderProfile{
		Mode: "openai", DefaultBaseURL: "", DefaultModel: "mimo-v2.5-pro",
		DetectionEnv: []string{EnvXiaomiTokenPlanAPIKey},
		ModelEnv:     []string{"XIAOMI_MIMO_TOKEN_PLAN_MODEL", "XIAOMI_MODEL", "OPENAI_MODEL"},
		BaseURLEnv:   []string{EnvXiaomiTokenPlanBaseURL, "OPENAI_BASE_URL", "OPENAI_API_BASE"},
		APIKeys:      []APIKeyDef{{Env: EnvXiaomiTokenPlanAPIKey, Source: "xiaomi_mimo_token_plan"}},
	}
)

// APIProviderDetectionOrder is the priority order for provider detection.
var APIProviderDetectionOrder = []APIProvider{
	ProviderAnthropic, ProviderOpenRouter, ProviderGrok, ProviderGemini,
	ProviderVertex, ProviderBedrock, ProviderZAI, ProviderCanopyWave, ProviderDeepSeek, ProviderAzure, ProviderOpenAI, ProviderOpenCodeGo,
	ProviderKimi, ProviderXiaomiMimoPayg, ProviderXiaomiMimoTokenPlan, ProviderOllama,
}

// ProviderModelEnvKeys maps each provider to its model env var keys.
var ProviderModelEnvKeys = map[APIProvider][]string{
	ProviderAnthropic:           AnthropicRuntimeProfile.ModelEnv,
	ProviderOpenAI:              OpenAIRuntimeProfile.ModelEnv,
	ProviderAzure:               AzureRuntimeProfile.ModelEnv,
	ProviderCanopyWave:          CanopyWaveRuntimeProfile.ModelEnv,
	ProviderDeepSeek:            DeepSeekRuntimeProfile.ModelEnv,
	ProviderZAI:                 ZAIRuntimeProfile.ModelEnv,
	ProviderOpenRouter:          OpenRouterRuntimeProfile.ModelEnv,
	ProviderGrok:                GrokRuntimeProfile.ModelEnv,
	ProviderGemini:              GeminiRuntimeProfile.ModelEnv,
	ProviderBedrock:             BedrockRuntimeProfile.ModelEnv,
	ProviderVertex:              VertexRuntimeProfile.ModelEnv,
	ProviderOllama:              {"OLLAMA_MODEL", "OPENAI_MODEL"},
	ProviderOpenCodeGo:          OpenCodeGoRuntimeProfile.ModelEnv,
	ProviderKimi:                KimiRuntimeProfile.ModelEnv,
	ProviderXiaomiMimoPayg:      XiaomiPaygRuntimeProfile.ModelEnv,
	ProviderXiaomiMimoTokenPlan: XiaomiTokenPlanRuntimeProfile.ModelEnv,
}

const (
	OllamaDefaultBaseURL     = "http://localhost:11434/v1"
	OllamaDefaultModel       = "llama3.1:8b"
	OpenCodeGoDefaultBaseURL = "https://opencode.ai/zen/go/v1"
	OpenCodeGoDefaultModel   = "kimi-k2.5"
	KimiDefaultModel         = "kimi-k2.6"
	XiaomiDefaultModel       = "mimo-v2-flash"
)

// OpenAICompatibleRuntimeProfileOrder is the detection order for runtime profiles.
var OpenAICompatibleRuntimeProfileOrder = []string{
	"openrouter", "grok", "gemini", "anthropic", "z-ai", "canopywave", "deepseek", "openai", "opencodego", "kimi", "xiaomi_mimo_payg", "xiaomi_mimo_token_plan",
}

// OpenAICompatibleRuntimeProfiles maps profile key to its runtime profile.
var OpenAICompatibleRuntimeProfiles = map[string]RuntimeProviderProfile{
	"anthropic":              AnthropicRuntimeProfile,
	"grok":                   GrokRuntimeProfile,
	"gemini":                 GeminiRuntimeProfile,
	"z-ai":                   ZAIRuntimeProfile,
	"canopywave":             CanopyWaveRuntimeProfile,
	"deepseek":               DeepSeekRuntimeProfile,
	"openai":                 OpenAIRuntimeProfile,
	"openrouter":             OpenRouterRuntimeProfile,
	"opencodego":             OpenCodeGoRuntimeProfile,
	"kimi":                   KimiRuntimeProfile,
	"xiaomi_mimo_payg":       XiaomiPaygRuntimeProfile,
	"xiaomi_mimo_token_plan": XiaomiTokenPlanRuntimeProfile,
}
