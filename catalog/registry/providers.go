package registry

import "github.com/GrayCodeAI/eyrie/catalog/opencodego"

func init() {
	for _, spec := range providerSpecs() {
		DefaultRegistry.Register(spec)
	}
}

// All returns every registered provider spec (sorted by SortOrder at use sites).
func All() []ProviderSpec {
	return DefaultRegistry.All()
}

func providerSpecs() []ProviderSpec {
	return []ProviderSpec{
		// ── Direct API providers ──────────────────────────────────────────
		{
			ProviderID: "anthropic", DisplayName: "Anthropic", DeploymentID: "anthropic-direct", SortOrder: 1,
			RequiresKey: true, CredentialEnv: "ANTHROPIC_API_KEY",
			BaseURLEnv:     []string{"ANTHROPIC_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:      ProbeAnthropic,
			LiveFetcherKey: "anthropic", LiveCatalogKey: "anthropic",
			APIProtocolID: "anthropic-messages", AdapterID: "anthropic",
		},
		{
			ProviderID: "openai", DisplayName: "OpenAI", DeploymentID: "openai-direct", SortOrder: 2,
			RequiresKey: true, CredentialEnv: "OPENAI_API_KEY",
			BaseURLEnv: []string{"OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.openai.com/v1",
			LiveFetcherKey: "openai", LiveCatalogKey: "openai",
			APIProtocolID: "openai-chat-completions", AdapterID: "openai",
		},
		{
			ProviderID: "gemini", DisplayName: "Gemini API", DeploymentID: "gemini-direct", SortOrder: 3,
			RequiresKey: true, CredentialEnv: "GEMINI_API_KEY",
			BaseURLEnv:     []string{"GEMINI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:      ProbeGemini,
			LiveFetcherKey: "gemini", LiveCatalogKey: "gemini",
			APIProtocolID: "gemini-generate-content", AdapterID: "gemini",
		},
		{
			ProviderID: "deepseek", DisplayName: "DeepSeek", DeploymentID: "deepseek-direct", SortOrder: 4,
			RequiresKey: true, CredentialEnv: "DEEPSEEK_API_KEY",
			BaseURLEnv: []string{"DEEPSEEK_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.deepseek.com/v1",
			LiveFetcherKey: "deepseek", LiveCatalogKey: "deepseek",
			APIProtocolID: "openai-chat-completions", AdapterID: "deepseek",
		},
		{
			ProviderID: "grok", DisplayName: "xAI", DeploymentID: "grok-direct", SortOrder: 5,
			RequiresKey: true, CredentialEnv: "XAI_API_KEY",
			BaseURLEnv: []string{"XAI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.x.ai/v1",
			LiveFetcherKey: "grok", LiveCatalogKey: "grok",
			APIProtocolID: "openai-chat-completions", AdapterID: "grok",
		},
		{
			ProviderID: "kimi", DisplayName: "Kimi", DeploymentID: "kimi-direct", SortOrder: 6,
			RequiresKey: true, CredentialEnv: "MOONSHOT_API_KEY",
			BaseURLEnv: []string{"MOONSHOT_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.moonshot.ai/v1",
			LiveFetcherKey: "kimi", LiveCatalogKey: "kimi",
			APIProtocolID: "openai-chat-completions", AdapterID: "kimi",
		},
		{
			ProviderID: "z-ai", DisplayName: "Z.AI", DeploymentID: "z-ai-direct", SortOrder: 7,
			RequiresKey: true, CredentialEnv: "ZAI_API_KEY",
			BaseURLEnv: []string{"ZAI_BASE_URL", "ZAI_API_BASE", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.z.ai/api/paas/v4",
			LiveFetcherKey: "z-ai", LiveCatalogKey: "z-ai",
			APIProtocolID: "openai-chat-completions", AdapterID: "z-ai",
		},
		{
			ProviderID: "xiaomi_mimo_token_plan", DisplayName: "Xiaomi MiMo — Token Plan", DeploymentID: "xiaomi_mimo_token_plan-direct", SortOrder: 8,
			RequiresKey: true, CredentialEnv: "XIAOMI_MIMO_TOKEN_PLAN_API_KEY",
			BaseURLEnv: []string{"XIAOMI_MIMO_TOKEN_PLAN_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "",
			LiveFetcherKey: "xiaomi_mimo_token_plan", LiveCatalogKey: "xiaomi_mimo_token_plan",
			APIProtocolID: "openai-chat-completions", AdapterID: "xiaomi_mimo",
		},
		{
			ProviderID: "xiaomi_mimo_payg", DisplayName: "Xiaomi MiMo — Pay-as-you-go", DeploymentID: "xiaomi_mimo_payg-direct", SortOrder: 9,
			RequiresKey: true, CredentialEnv: "XIAOMI_MIMO_PAYG_API_KEY",
			BaseURLEnv: []string{"XIAOMI_MIMO_PAYG_BASE_URL", "XIAOMI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.xiaomimimo.com/v1",
			LiveFetcherKey: "xiaomi_mimo_payg", LiveCatalogKey: "xiaomi_mimo_payg",
			APIProtocolID: "openai-chat-completions", AdapterID: "xiaomi_mimo",
		},
		{
			ProviderID: "minimax_token_plan", DisplayName: "MiniMax — Token Plan", DeploymentID: "minimax_token_plan-direct", SortOrder: 10,
			RequiresKey: true, CredentialEnv: "MINIMAX_TOKEN_PLAN_API_KEY",
			BaseURLEnv: []string{"MINIMAX_TOKEN_PLAN_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.minimax.io/v1",
			LiveFetcherKey: "minimax_token_plan", LiveCatalogKey: "minimax_token_plan",
			APIProtocolID: "openai-chat-completions", AdapterID: "openai",
		},
		{
			ProviderID: "minimax_payg", DisplayName: "MiniMax — Pay-as-you-go", DeploymentID: "minimax_payg-direct", SortOrder: 11,
			RequiresKey: true, CredentialEnv: "MINIMAX_PAYG_API_KEY",
			BaseURLEnv: []string{"MINIMAX_PAYG_BASE_URL", "MINIMAX_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://api.minimax.io/v1",
			LiveFetcherKey: "minimax_payg", LiveCatalogKey: "minimax_payg",
			APIProtocolID: "openai-chat-completions", AdapterID: "openai",
		},

		// ── Cloud platform providers ──────────────────────────────────────
		{
			ProviderID: "azure", DisplayName: "Azure OpenAI", DeploymentID: "openai-azure", SortOrder: 12,
			RequiresKey: true, CredentialEnv: "AZURE_OPENAI_API_KEY",
			BaseURLEnv:     []string{"AZURE_OPENAI_ENDPOINT"},
			ProbeKind:      ProbeNone,
			LiveFetcherKey: "azure", LiveCatalogKey: "azure",
			APIProtocolID: "openai-chat-completions", AdapterID: "openai-azure",
		},
		{
			ProviderID: "bedrock", DisplayName: "Amazon Bedrock", DeploymentID: "anthropic-bedrock", SortOrder: 13,
			RequiresKey: true, CredentialEnv: "AWS_SECRET_ACCESS_KEY",
			CredentialEnvFallbacks: []string{"AWS_ACCESS_KEY_ID", "AWS_SESSION_TOKEN"},
			BaseURLEnv:             []string{"AWS_REGION", "AWS_DEFAULT_REGION"},
			ProbeKind:              ProbeNone,
			LiveFetcherKey:         "bedrock", LiveCatalogKey: "bedrock",
			APIProtocolID: "anthropic-messages", AdapterID: "anthropic-bedrock",
		},
		{
			ProviderID: "vertex", DisplayName: "Vertex AI", DeploymentID: "gemini-vertex", SortOrder: 14,
			RequiresKey: true, CredentialEnv: "VERTEX_ACCESS_TOKEN",
			CredentialEnvFallbacks: []string{"GOOGLE_OAUTH_ACCESS_TOKEN"},
			BaseURLEnv:             []string{"VERTEX_PROJECT_ID", "VERTEX_REGION"},
			ProbeKind:              ProbeNone,
			LiveFetcherKey:         "vertex", LiveCatalogKey: "vertex",
			APIProtocolID: "gemini-generate-content", AdapterID: "gemini-vertex",
		},

		// ── Aggregators ───────────────────────────────────────────────────
		{
			ProviderID: "openrouter", DisplayName: "OpenRouter", DeploymentID: "openrouter", SortOrder: 15,
			RequiresKey: true, CredentialEnv: "OPENROUTER_API_KEY",
			BaseURLEnv: []string{"OPENROUTER_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://openrouter.ai/api/v1",
			LiveFetcherKey: "openrouter", LiveCatalogKey: "openrouter",
			APIProtocolID: "openai-chat-completions", AdapterID: "openrouter",
		},

		// ── Niche ─────────────────────────────────────────────────────────
		{
			ProviderID: "canopywave", DisplayName: "CanopyWave", DeploymentID: "canopywave", SortOrder: 16,
			RequiresKey: true, CredentialEnv: "CANOPYWAVE_API_KEY",
			BaseURLEnv: []string{"CANOPYWAVE_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:  ProbeOpenAIModels, ProbeBaseURL: "https://inference.canopywave.io/v1",
			LiveFetcherKey: "canopywave", LiveCatalogKey: "canopywave",
			APIProtocolID: "openai-chat-completions", AdapterID: "canopywave",
		},
		{
			ProviderID: "opencodego", DisplayName: "OpenCode Go", DeploymentID: "opencodego", SortOrder: 17,
			RequiresKey: true, CredentialEnv: "OPENCODEGO_API_KEY",
			BaseURLEnv:     []string{"OPENCODEGO_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"},
			ProbeKind:      ProbeOpenAIModels,
			ProbeBaseURL:   opencodego.DefaultBaseURL,
			LiveFetcherKey: "opencodego", LiveCatalogKey: "opencodego",
			APIProtocolID: "openai-chat-completions", AdapterID: "opencodego",
		},

		// ── Local ─────────────────────────────────────────────────────────
		{
			ProviderID: "ollama", DisplayName: "Ollama", DeploymentID: "ollama-local", SortOrder: 18,
			RequiresKey: false, CredentialEnv: "OLLAMA_BASE_URL",
			BaseURLEnv:     []string{"OLLAMA_BASE_URL"},
			ProbeKind:      ProbeOllama,
			LiveFetcherKey: "ollama", LiveCatalogKey: "ollama",
			APIProtocolID: "openai-chat-completions", AdapterID: "openai",
			IsLocal: true,
			RetryConfig: &RetryConfig{
				BaseDelayMs: 2000, MaxDelayMs: 10000, MaxRetries: 3,
				BackoffMultiplier: 1.5, JitterPct: 10,
				RetryOnCodes: []int{500, 503},
				AbortOnCodes: []int{400},
			},
		},
	}
}
