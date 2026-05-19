package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// testLegacyModelCatalog is used by unit tests only — not shipped as production model data.
func testLegacyModelCatalog() ModelCatalog {
	return ModelCatalog{
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
}

func testLegacyCatalogV1() CatalogV1 {
	return CatalogV1FromLegacy(testLegacyModelCatalog())
}

// Run with EXPORT_HAWK_FIXTURE=1 to refresh hawk/internal/catalogtest/testdata/minimal_v1.json
func TestExportHawkCatalogFixture(t *testing.T) {
	if os.Getenv("EXPORT_HAWK_FIXTURE") != "1" {
		t.Skip("set EXPORT_HAWK_FIXTURE=1 to export")
	}
	c := testLegacyCatalogV1()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join("..", "..", "hawk", "internal", "catalogtest", "testdata", "minimal_v1.json")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
