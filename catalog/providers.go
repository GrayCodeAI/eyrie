package catalog

// Embedded model data per provider.

var AnthropicModels = []ModelCatalogEntry{
	{ID: "claude-opus-4-6", InputPricePer1M: 15, OutputPricePer1M: 75, ContextWindow: 200000, MaxOutput: 32000, ServerTools: []string{"web_search"}},
	{ID: "claude-sonnet-4-6", InputPricePer1M: 3, OutputPricePer1M: 15, ContextWindow: 200000, MaxOutput: 32000, ServerTools: []string{"web_search"}},
	{ID: "claude-haiku-4-5-20251001", InputPricePer1M: 1, OutputPricePer1M: 5, ContextWindow: 200000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
}

var OpenAIModels = []ModelCatalogEntry{
	{ID: "gpt-4o", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
	{ID: "gpt-4o-mini", InputPricePer1M: 0.15, OutputPricePer1M: 0.6, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
}

var GrokModels = []ModelCatalogEntry{
	{ID: "grok-2", InputPricePer1M: 2, OutputPricePer1M: 10, ContextWindow: 128000, MaxOutput: 8000, ServerTools: []string{"web_search"}},
}

var GeminiModels = []ModelCatalogEntry{
	{ID: "gemini-2.5-pro-preview-03-25", InputPricePer1M: 1.25, OutputPricePer1M: 5, ContextWindow: 1000000, MaxOutput: 65536, ServerTools: []string{"web_search"}},
	{ID: "gemini-2.0-flash", InputPricePer1M: 0.1, OutputPricePer1M: 0.4, ContextWindow: 1000000, MaxOutput: 8192, ServerTools: []string{"web_search"}},
	{ID: "gemini-2.0-flash-lite", InputPricePer1M: 0.075, OutputPricePer1M: 0.3, ContextWindow: 1000000, MaxOutput: 8192, ServerTools: []string{"web_search"}},
}

var OpenRouterModels = []ModelCatalogEntry{
	{ID: "openai/gpt-4o", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
	{ID: "openai/gpt-4o-mini", InputPricePer1M: 0.15, OutputPricePer1M: 0.6, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
	{ID: "anthropic/claude-sonnet-4-6", InputPricePer1M: 3, OutputPricePer1M: 15, ContextWindow: 200000, MaxOutput: 32000, ServerTools: []string{"web_search"}},
}

var CanopyWaveModels = []ModelCatalogEntry{
	{ID: "zai/glm-4.6", InputPricePer1M: 0, OutputPricePer1M: 0, ContextWindow: 128000, MaxOutput: 8192},
}

var OllamaModels = []ModelCatalogEntry{}

var OpenCodeGoModels = []ModelCatalogEntry{
	{ID: "glm-5.1", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "GLM-5.1", Description: "Zhipu GLM-5.1 · Advanced reasoning model"},
	{ID: "glm-5", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "GLM-5", Description: "Zhipu GLM-5 · Powerful general-purpose model"},
	{ID: "kimi-k2.5", InputPricePer1M: 3, OutputPricePer1M: 10, ContextWindow: 256000, MaxOutput: 8000, DisplayName: "Kimi K2.5", Description: "Moonshot Kimi K2.5 · Long-context specialist"},
	{ID: "kimi-k2.6", InputPricePer1M: 3, OutputPricePer1M: 10, ContextWindow: 256000, MaxOutput: 8000, DisplayName: "Kimi K2.6", Description: "Moonshot Kimi K2.6 · Enhanced long-context model"},
	{ID: "mimo-v2-pro", InputPricePer1M: 3, OutputPricePer1M: 10, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "MiMo V2 Pro", Description: "MiMo V2 Pro · Professional-grade model"},
	{ID: "mimo-v2-omni", InputPricePer1M: 2, OutputPricePer1M: 8, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "MiMo V2 Omni", Description: "MiMo V2 Omni · Versatile multimodal model"},
	{ID: "minimax-m2.7", InputPricePer1M: 1, OutputPricePer1M: 3, ContextWindow: 1000000, MaxOutput: 8000, DisplayName: "MiniMax M2.7", Description: "MiniMax M2.7 · Latest generation with 1M context"},
	{ID: "minimax-m2.5", InputPricePer1M: 0.5, OutputPricePer1M: 1.5, ContextWindow: 1000000, MaxOutput: 8000, DisplayName: "MiniMax M2.5", Description: "MiniMax M2.5 · Cost-effective with 1M context"},
	{ID: "qwen3.6-plus", InputPricePer1M: 0.3, OutputPricePer1M: 1.7, ContextWindow: 1000000, MaxOutput: 65536, DisplayName: "Qwen3.6 Plus", Description: "Alibaba Qwen3.6 Plus · Latest Qwen with 1M context"},
	{ID: "qwen3.5-plus", InputPricePer1M: 0.26, OutputPricePer1M: 1.56, ContextWindow: 1000000, MaxOutput: 65536, DisplayName: "Qwen3.5 Plus", Description: "Alibaba Qwen3.5 Plus · Strong coding capabilities"},
}

// DefaultProviderCatalogs returns the embedded catalog data for all providers.
func DefaultProviderCatalogs() map[string][]ModelCatalogEntry {
	return map[string][]ModelCatalogEntry{
		"anthropic":  AnthropicModels,
		"openai":     OpenAIModels,
		"grok":       GrokModels,
		"gemini":     GeminiModels,
		"openrouter": OpenRouterModels,
		"canopywave": CanopyWaveModels,
		"ollama":     OllamaModels,
		"opencodego": OpenCodeGoModels,
	}
}
