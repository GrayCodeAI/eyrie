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
		entry := ModelCatalogEntry{
			ID:               e.ID,
			DisplayName:      DisplayModelLabel(e.ID, e.DisplayName),
			Description:      e.Description,
			Owner:            DisplayModelOwner(e.OwnedBy, e.ID),
			ContextWindow:    e.ContextWindow,
			MaxOutput:        e.MaxOutput,
			InputPricePer1M:  e.InputPricePer1M,
			OutputPricePer1M: e.OutputPricePer1M,
			ServerTools:      append([]string(nil), e.Features...),
			LiveMetadata:     e.RawJSON,
		}
		// Propagate cached pricing if available.
		if e.CachedReadPricePer1M > 0 || e.CachedWritePricePer1M > 0 {
			entry.CachedReadPricePer1M = e.CachedReadPricePer1M
			entry.CachedWritePricePer1M = e.CachedWritePricePer1M
		}
		// Propagate tiered pricing if available.
		if e.TierThreshold > 0 {
			entry.TierThreshold = e.TierThreshold
			entry.TieredInputPricePer1M = e.TieredInputPricePer1M
			entry.TieredOutputPricePer1M = e.TieredOutputPricePer1M
			entry.TieredCachedReadPer1M = e.TieredCachedReadPer1M
			entry.TieredCachedWritePer1M = e.TieredCachedWritePer1M
		}
		out[i] = entry
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
