package catalog

// Test fixture model slices — not used in production (cache/discovery only).

var testAnthropicModels = []ModelCatalogEntry{
	{ID: "claude-opus-4-6", InputPricePer1M: 15, OutputPricePer1M: 75, ContextWindow: 200000, MaxOutput: 32000, ServerTools: []string{"web_search"}},
	{ID: "claude-sonnet-4-6", InputPricePer1M: 3, OutputPricePer1M: 15, ContextWindow: 200000, MaxOutput: 32000, ServerTools: []string{"web_search"}},
	{ID: "claude-haiku-4-5-20251001", InputPricePer1M: 1, OutputPricePer1M: 5, ContextWindow: 200000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
}

var testOpenAIModels = []ModelCatalogEntry{
	{ID: "gpt-4o", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
	{ID: "gpt-4o-mini", InputPricePer1M: 0.15, OutputPricePer1M: 0.6, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
}

var testGrokModels = []ModelCatalogEntry{
	{ID: "grok-2", InputPricePer1M: 2, OutputPricePer1M: 10, ContextWindow: 128000, MaxOutput: 8000, ServerTools: []string{"web_search"}},
}

var testGeminiModels = []ModelCatalogEntry{
	{ID: "gemini-2.5-pro-preview-03-25", InputPricePer1M: 1.25, OutputPricePer1M: 5, ContextWindow: 1000000, MaxOutput: 65536, ServerTools: []string{"web_search"}},
	{ID: "gemini-2.0-flash", InputPricePer1M: 0.1, OutputPricePer1M: 0.4, ContextWindow: 1000000, MaxOutput: 8192, ServerTools: []string{"web_search"}},
	{ID: "gemini-2.0-flash-lite", InputPricePer1M: 0.075, OutputPricePer1M: 0.3, ContextWindow: 1000000, MaxOutput: 8192, ServerTools: []string{"web_search"}},
}

var testOpenRouterModels = []ModelCatalogEntry{
	{ID: "openai/gpt-4o", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
	{ID: "openai/gpt-4o-mini", InputPricePer1M: 0.15, OutputPricePer1M: 0.6, ContextWindow: 128000, MaxOutput: 16000, ServerTools: []string{"web_search"}},
	{ID: "anthropic/claude-sonnet-4-6", InputPricePer1M: 3, OutputPricePer1M: 15, ContextWindow: 200000, MaxOutput: 32000, ServerTools: []string{"web_search"}},
}

var testCanopyWaveModels = []ModelCatalogEntry{
	{ID: "zai/glm-4.6", InputPricePer1M: 0, OutputPricePer1M: 0, ContextWindow: 128000, MaxOutput: 8192},
}

var testOpenCodeGoModels = []ModelCatalogEntry{
	{ID: "glm-5.1", InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "GLM-5.1"},
}

func testLegacyModelCatalog() ModelCatalog {
	return ModelCatalog{
		UpdatedAt: "2026-04-09T00:00:00.000Z",
		Source:    "test",
		Providers: map[string][]ModelCatalogEntry{
			"anthropic":  testAnthropicModels,
			"openai":     testOpenAIModels,
			"grok":       testGrokModels,
			"gemini":     testGeminiModels,
			"openrouter": testOpenRouterModels,
			"canopywave": testCanopyWaveModels,
			"ollama":     nil,
			"opencodego": testOpenCodeGoModels,
		},
	}
}

func testLegacyCatalogV1() CatalogV1 {
	return CatalogV1FromLegacy(testLegacyModelCatalog())
}

// TestSeedCatalogV1 returns a v1 catalog built from embedded test fixtures.
func TestSeedCatalogV1() CatalogV1 {
	return testLegacyCatalogV1()
}

// CompileTestCatalog builds a compiled catalog from built-in provider model lists (tests and dev fixtures).
func CompileTestCatalog() (*CompiledCatalogV1, error) {
	c := testLegacyCatalogV1()
	return CompileCatalogV1(&c)
}
