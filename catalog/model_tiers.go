package catalog

import "strings"

// ModelName is a model identifier string.
type ModelName = string

// ModelTier represents opus/sonnet/haiku tiers.
type ModelTier string

const (
	TierOpus   ModelTier = "opus"
	TierSonnet ModelTier = "sonnet"
	TierHaiku  ModelTier = "haiku"
)

// ModelTierAliases lists all valid tier names.
var ModelTierAliases = []ModelTier{TierSonnet, TierHaiku, TierOpus}

// ModelConfig maps each APIProvider to a model name.
type ModelConfig map[string]ModelName

// ModelKey identifies a specific model version config.
type ModelKey string

// GetProviderModelCandidates returns tier candidates from the live catalog.
// All model discovery is dynamic — returns nil (use live catalog).
func GetProviderModelCandidates(provider string, tier ModelTier) []ModelName {
	return nil
}

func catalogModelIDs(catalog *ModelCatalog, provider string) []string {
	if catalog == nil {
		return nil
	}
	models := catalog.Providers[provider]
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	return ids
}

// GetPreferredProviderModel returns the preferred model for a provider/tier from the catalog.
// Returns "" if catalog is nil or has no models for this provider.
func GetPreferredProviderModel(provider string, tier ModelTier, catalog *ModelCatalog) ModelName {
	if catalog == nil {
		return ""
	}
	ids := catalogModelIDs(catalog, provider)
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// GetProviderDefaultModel returns the first model from the catalog for a provider.
// Returns "" if no catalog data is available.
func GetProviderDefaultModel(provider string, catalog *ModelCatalog) ModelName {
	if catalog == nil {
		return ""
	}
	ids := catalogModelIDs(catalog, provider)
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// AnthropicNameToCanonical normalizes an Anthropic model ID to its canonical short name.
func AnthropicNameToCanonical(name string) string {
	name = strings.ToLower(name)
	checks := []struct{ sub, canon string }{
		{"claude-opus-4-6", "claude-opus-4-6"},
		{"claude-opus-4-5", "claude-opus-4-5"},
		{"claude-opus-4-1", "claude-opus-4-1"},
		{"claude-opus-4", "claude-opus-4"},
		{"claude-sonnet-4-6", "claude-sonnet-4-6"},
		{"claude-sonnet-4-5", "claude-sonnet-4-5"},
		{"claude-sonnet-4", "claude-sonnet-4"},
		{"claude-haiku-4-5", "claude-haiku-4-5"},
		{"claude-3-7-sonnet", "claude-3-7-sonnet"},
		{"claude-3-5-sonnet", "claude-3-5-sonnet"},
		{"claude-3-5-haiku", "claude-3-5-haiku"},
		{"claude-3-opus", "claude-3-opus"},
		{"claude-3-sonnet", "claude-3-sonnet"},
		{"claude-3-haiku", "claude-3-haiku"},
	}
	for _, c := range checks {
		if strings.Contains(name, c.sub) {
			return c.canon
		}
	}
	return name
}

// GetModelMarketingName returns the marketing display name for a model ID.
func GetModelMarketingName(modelID string) string {
	lower := strings.ToLower(modelID)
	has1m := strings.Contains(lower, "[1m]")
	base := strings.NewReplacer("[1m]", "", "[2m]", "", "[1M]", "", "[2M]", "").Replace(lower)
	base = strings.TrimSpace(base)
	canonical := AnthropicNameToCanonical(base)

	type entry struct{ sub, name, name1m string }
	names := []entry{
		{"claude-opus-4-6", "Opus 4.6", "Opus 4.6 (1M context)"},
		{"claude-opus-4-5", "Opus 4.5", ""},
		{"claude-opus-4-1", "Opus 4.1", ""},
		{"claude-opus-4", "Opus 4", ""},
		{"claude-sonnet-4-6", "Sonnet 4.6", "Sonnet 4.6 (1M context)"},
		{"claude-sonnet-4-5", "Sonnet 4.5", "Sonnet 4.5 (1M context)"},
		{"claude-sonnet-4", "Sonnet 4", "Sonnet 4 (1M context)"},
		{"claude-3-7-sonnet", "Sonnet 3.7", ""},
		{"claude-3-5-sonnet", "Sonnet 3.5", ""},
		{"claude-haiku-4-5", "Haiku 4.5", ""},
		{"claude-3-5-haiku", "Haiku 3.5", ""},
	}
	for _, e := range names {
		if strings.Contains(canonical, e.sub) {
			if has1m && e.name1m != "" {
				return e.name1m
			}
			return e.name
		}
	}
	return ""
}
