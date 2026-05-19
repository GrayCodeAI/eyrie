package catalog

// CompileTestCatalog builds a compiled catalog from built-in provider model lists (tests and dev fixtures).
func CompileTestCatalog() (*CompiledCatalogV1, error) {
	legacy := ModelCatalog{
		UpdatedAt: "2026-04-09T00:00:00.000Z",
		Source:    "test",
		Providers: map[string][]ModelCatalogEntry{
			"anthropic":  AnthropicModels,
			"openai":     OpenAIModels,
			"grok":       GrokModels,
			"gemini":     GeminiModels,
			"openrouter": OpenRouterModels,
			"canopywave": CanopyWaveModels,
			"ollama":     OllamaModels,
			"opencodego": OpenCodeGoModels,
		},
	}
	c := CatalogV1FromLegacy(legacy)
	return CompileCatalogV1(&c)
}
