package catalog

import "time"

// deploymentsForProvider returns all deployment IDs belonging to the given provider.
func deploymentsForProvider(cat *Catalog, providerID string) []string {
	var ids []string
	for id, dep := range cat.Deployments {
		if dep.ProviderID == providerID {
			ids = append(ids, id)
		}
	}
	return ids
}

// ModelID constants for well-known models used across tests.
const (
	ClaudeOpusV4_6    = "claude-opus-4-6"
	ClaudeSonnetV4_6  = "claude-sonnet-4-6"
	ClaudeHaikuV4_5   = "claude-haiku-4-5-20251001"
	GPT_4o            = "gpt-4o"
	GPT_4oMini        = "gpt-4o-mini"
	Grokk_2           = "grok-2"
	Glm51             = "glm-5.1"
)

// Fixture model lists — used exclusively by SeedCatalog to validate catalog
// construction and model registry. All provider model lists are defined here so
// that producers and consumers stay in sync.

var seedAnthropicModels = []ModelCatalogEntry{
	{ID: ClaudeOpusV4_6, InputPricePer1M: 15, OutputPricePer1M: 75, ContextWindow: 200000, MaxOutput: 32000, DisplayName: "Claude Opus 4.6", ServerTools: []string{"web_search"}},
	{ID: ClaudeSonnetV4_6, InputPricePer1M: 3, OutputPricePer1M: 15, ContextWindow: 200000, MaxOutput: 32000, DisplayName: "Claude Sonnet 4.6", ServerTools: []string{"web_search"}},
	{ID: ClaudeHaikuV4_5, InputPricePer1M: 1, OutputPricePer1M: 5, ContextWindow: 200000, MaxOutput: 16000, DisplayName: "Claude Haiku 4.5", ServerTools: []string{"web_search"}},
}

var seedOpenAIModels = []ModelCatalogEntry{
	{ID: GPT_4o, InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 16000, DisplayName: "GPT-4o", ServerTools: []string{"web_search"}},
	{ID: GPT_4oMini, InputPricePer1M: 0.15, OutputPricePer1M: 0.6, ContextWindow: 128000, MaxOutput: 16000, DisplayName: "GPT-4o Mini", ServerTools: []string{"web_search"}},
}

var seedGrokModels = []ModelCatalogEntry{
	{ID: Grokk_2, InputPricePer1M: 2, OutputPricePer1M: 10, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "Grokk 2", ServerTools: []string{"web_search"}},
}

var seedGeminiModels = []ModelCatalogEntry{
	{ID: "gemini-2.5-pro-preview-03-25", InputPricePer1M: 1.25, OutputPricePer1M: 5, ContextWindow: 1000000, MaxOutput: 65536, DisplayName: "Gemini 2.5 Pro", ServerTools: []string{"web_search"}},
	{ID: "gemini-2.0-flash", InputPricePer1M: 0.1, OutputPricePer1M: 0.4, ContextWindow: 1000000, MaxOutput: 8192, DisplayName: "Gemini 2.0 Flash", ServerTools: []string{"web_search"}},
	{ID: "gemini-2.0-flash-lite", InputPricePer1M: 0.075, OutputPricePer1M: 0.3, ContextWindow: 1000000, MaxOutput: 8192, DisplayName: "Gemini 2.0 Flash Lite", ServerTools: []string{"web_search"}},
}

var seedOpenRouterModels []ModelCatalogEntry

var seedCanopyWaveModels []ModelCatalogEntry

var seedOpenCodeGoModels = []ModelCatalogEntry{
	{ID: Glm51, InputPricePer1M: 5, OutputPricePer1M: 15, ContextWindow: 128000, MaxOutput: 8000, DisplayName: "GLM-5.1"},
}

// SeedModelCatalog returns the embedded test model catalog.
func SeedModelCatalog() ModelCatalog {
	return ModelCatalog{
		Source: "test",
		Providers: map[string][]ModelCatalogEntry{
			"anthropic":   seedAnthropicModels,
			"openai":      seedOpenAIModels,
			"grok":        seedGrokModels,
			"gemini":      seedGeminiModels,
			"openrouter":  seedOpenRouterModels,
			"canopywave":  seedCanopyWaveModels,
			"ollama":      nil,
			"opencodego": seedOpenCodeGoModels,
		},
	}
}

// seedCatalogFromDefault returns a Catalog built from the embedded test fixtures,
// using the current default provider/deployment/protocol registries.
func seedCatalogFromDefault() Catalog {
	now := time.Now().UTC()
	cat := Catalog{
		SchemaVersion: CatalogSchemaVersion,
		GeneratedAt:  now,
		StaleAfter:   now.Add(30 * 24 * time.Hour),
		Providers:    defaultProviders(),
		Protocols:    defaultProtocols(),
		Deployments:  defaultDeployments(),
		Models:       map[string]Model{},
		Aliases:      map[string]string{},
		Provenance:   &Provenance{Source: "test", ObservedAt: now},
	}
	EnsureDeploymentEnvFallbacks(&cat)
	EnsureCredentialRegistryInCatalog(&cat)

	for _, e := range seedAnthropicModels {
		addModel(&cat, "anthropic", e)
	}
	for _, e := range seedOpenAIModels {
		addModel(&cat, "openai", e)
	}
	for _, e := range seedGrokModels {
		addModel(&cat, "xai", e)
	}
	for _, e := range seedGeminiModels {
		addModel(&cat, "google", e)
	}
	for _, e := range seedOpenRouterModels {
		addModel(&cat, "openrouter", e)
	}
	for _, e := range seedCanopyWaveModels {
		addModel(&cat, "canopywave", e)
	}
	for _, e := range seedOpenCodeGoModels {
		addModel(&cat, "opencodego", e)
	}

	return cat
}

func addModel(cat *Catalog, providerID string, e ModelCatalogEntry) {
	modelID := providerID + "/" + e.ID
	cat.Models[modelID] = Model{
		ID:            modelID,
		ProviderID:    providerID,
		Name:          e.DisplayName,
		ContextWindow: e.ContextWindow,
		MaxOutput:     e.MaxOutput,
	}
	cat.Aliases[e.ID] = modelID
	for _, deploymentID := range deploymentsForProvider(cat, providerID) {
		cat.Offerings = append(cat.Offerings, ModelOffering{
			ID:               deploymentID + ":" + e.ID,
			CanonicalModelID: modelID,
			DeploymentID:     deploymentID,
			NativeModelID:    e.ID,
			Capabilities:     capabilitySetFromLegacy(e),
			Pricing:          pricingFromLegacy(e, time.Now().UTC()),
		})
	}
}

// SeedCatalog returns a catalog built from the embedded test fixtures.
func SeedCatalog() Catalog {
	return seedCatalogFromDefault()
}

// CompileTestCatalog builds a compiled catalog from the embedded test fixtures.
func CompileTestCatalog() (*CompiledCatalog, error) {
	c := SeedCatalog()
	return CompileCatalog(&c)
}

// testModelCatalog is an alias for backward compatibility.
func testModelCatalog() ModelCatalog {
	return SeedModelCatalog()
}
