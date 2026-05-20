package catalog

import (
	"github.com/GrayCodeAI/eyrie/catalog/live"
)

func liveEntriesToCatalog(in []live.Entry) []ModelCatalogEntry {
	return LiveEntriesToCatalog(in)
}

// LiveEntriesToCatalog converts live fetch rows to catalog entries.
func LiveEntriesToCatalog(in []live.Entry) []ModelCatalogEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]ModelCatalogEntry, len(in))
	for i, e := range in {
		out[i] = ModelCatalogEntry{
			ID:                e.ID,
			DisplayName:       e.DisplayName,
			ContextWindow:     e.ContextWindow,
			MaxOutput:         e.MaxOutput,
			InputPricePer1M:   e.InputPricePer1M,
			OutputPricePer1M:  e.OutputPricePer1M,
		}
	}
	return out
}

// FetchOllamaModels lists models installed on a running Ollama instance.
func FetchOllamaModels(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchOllama(env)
	if err != nil {
		return nil, err
	}
	return liveEntriesToCatalog(entries), nil
}

func fetchOpenRouterCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchOpenRouter(env)
	return liveEntriesToCatalog(entries), err
}

func fetchCanopyWaveCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchCanopyWave(env)
	return liveEntriesToCatalog(entries), err
}

func fetchOllamaCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	return FetchOllamaModels(env)
}

func fetchAnthropicCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchAnthropic(env)
	return liveEntriesToCatalog(entries), err
}

func fetchOpenAICatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchOpenAI(env)
	return liveEntriesToCatalog(entries), err
}

func fetchGeminiCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchGemini(env)
	return liveEntriesToCatalog(entries), err
}

func fetchGrokCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchGrok(env)
	return liveEntriesToCatalog(entries), err
}

func fetchOpenCodeGoCatalog(env map[string]string) ([]ModelCatalogEntry, error) {
	entries, err := live.FetchOpenCodeGo(env)
	return liveEntriesToCatalog(entries), err
}
