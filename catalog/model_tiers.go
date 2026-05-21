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

// Per-provider model defaults for OpenAI-compatible tiers.
var OpenAIModelDefaults = map[ModelTier]string{
	TierOpus: "gpt-4o", TierSonnet: "gpt-4o-mini", TierHaiku: "gpt-4o-mini",
}

var GeminiModelDefaults = map[ModelTier]string{
	TierOpus: "gemini-2.5-pro-preview-03-25", TierSonnet: "gemini-2.0-flash", TierHaiku: "gemini-2.0-flash-lite",
}

// Individual model configs (tier × version → model ID per provider).
var (
	Hawk37SonnetConfig = ModelConfig{
		"anthropic": "claude-3-7-sonnet-20250219", "openai": "gpt-4o-mini", "z-ai": "glm-5.1", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o-mini", "grok": "grok-2", "gemini": "gemini-2.0-flash",
		"ollama": "llama3.1:8b", "opencodego": "kimi-k2.5",
	}
	Hawk35V2SonnetConfig = ModelConfig{
		"anthropic": "claude-3-5-sonnet-20241022", "openai": "gpt-4o-mini", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o-mini", "grok": "grok-2", "gemini": "gemini-2.0-flash",
		"ollama": "llama3.1:8b", "opencodego": "kimi-k2.5",
	}
	Hawk35HaikuConfig = ModelConfig{
		"anthropic": "claude-3-5-haiku-20241022", "openai": "gpt-4o-mini", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o-mini", "grok": "grok-2", "gemini": "gemini-2.0-flash-lite",
		"ollama": "llama3.2:3b", "opencodego": "kimi-k2.5",
	}
	HawkHaiku45Config = ModelConfig{
		"anthropic": "claude-haiku-4-5-20251001", "openai": "gpt-4o-mini", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o-mini", "grok": "grok-2", "gemini": "gemini-2.0-flash-lite",
		"ollama": "llama3.2:3b", "opencodego": "kimi-k2.5",
	}
	HawkSonnet4Config = ModelConfig{
		"anthropic": "claude-sonnet-4-20250514", "openai": "gpt-4o-mini", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o-mini", "grok": "grok-2", "gemini": "gemini-2.0-flash",
		"ollama": "llama3.1:8b", "opencodego": "kimi-k2.5",
	}
	HawkSonnet45Config = ModelConfig{
		"anthropic": "claude-sonnet-4-5-20250929", "openai": "gpt-4o", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o", "grok": "grok-2", "gemini": "gemini-2.0-flash",
		"ollama": "llama3.1:70b", "opencodego": "kimi-k2.5",
	}
	HawkSonnet46Config = ModelConfig{
		"anthropic": "claude-sonnet-4-6", "openai": "gpt-4o", "z-ai": "glm-5.1", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o", "grok": "grok-2", "gemini": "gemini-2.0-flash",
		"ollama": "llama3.1:70b", "opencodego": "kimi-k2.5",
	}
	HawkOpus4Config = ModelConfig{
		"anthropic": "claude-opus-4-20250514", "openai": "gpt-4o", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o", "grok": "grok-2", "gemini": "gemini-2.5-pro-preview-03-25",
		"ollama": "llama3.1:70b", "opencodego": "kimi-k2.5",
	}
	HawkOpus41Config = ModelConfig{
		"anthropic": "claude-opus-4-1-20250805", "openai": "gpt-4o", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o", "grok": "grok-2", "gemini": "gemini-2.5-pro-preview-03-25",
		"ollama": "llama3.1:70b", "opencodego": "kimi-k2.5",
	}
	HawkOpus45Config = ModelConfig{
		"anthropic": "claude-opus-4-5-20251101", "openai": "gpt-4o", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o", "grok": "grok-2", "gemini": "gemini-2.5-pro-preview-03-25",
		"ollama": "llama3.1:70b", "opencodego": "kimi-k2.5",
	}
	HawkOpus46Config = ModelConfig{
		"anthropic": "claude-opus-4-6", "openai": "gpt-4o", "canopywave": "zai/glm-4.6",
		"openrouter": "openai/gpt-4o", "grok": "grok-2", "gemini": "gemini-2.5-pro-preview-03-25",
		"ollama": "llama3.1:70b", "opencodego": "kimi-k2.5",
	}
)

// AllModelConfigs maps ModelKey to its config.
var AllModelConfigs = map[ModelKey]ModelConfig{
	"haiku35":  Hawk35HaikuConfig,
	"haiku45":  HawkHaiku45Config,
	"sonnet35": Hawk35V2SonnetConfig,
	"sonnet37": Hawk37SonnetConfig,
	"sonnet40": HawkSonnet4Config,
	"sonnet45": HawkSonnet45Config,
	"sonnet46": HawkSonnet46Config,
	"opus40":   HawkOpus4Config,
	"opus41":   HawkOpus41Config,
	"opus45":   HawkOpus45Config,
	"opus46":   HawkOpus46Config,
}

var modelKeys = []ModelKey{
	"haiku35", "haiku45", "sonnet35", "sonnet37", "sonnet40", "sonnet45", "sonnet46",
	"opus40", "opus41", "opus45", "opus46",
}

// CanonicalModelIDs returns all canonical Anthropic model IDs.
func CanonicalModelIDs() []string {
	var ids []string
	for _, key := range modelKeys {
		ids = append(ids, AllModelConfigs[key]["anthropic"])
	}
	return ids
}

// CanonicalIDToKey maps canonical Anthropic model ID → ModelKey.
func CanonicalIDToKey() map[string]ModelKey {
	m := make(map[string]ModelKey)
	for _, key := range modelKeys {
		m[AllModelConfigs[key]["anthropic"]] = key
	}
	return m
}

// Preferred model keys by provider and tier.
var preferredKeys = map[string]map[ModelTier]ModelKey{
	"anthropic":  {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"openai":     {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"canopywave": {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"z-ai":       {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"openrouter": {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"grok":       {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"gemini":     {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"ollama":     {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
	"opencodego": {TierOpus: "opus46", TierSonnet: "sonnet46", TierHaiku: "haiku45"},
}

var fallbackKeys = map[ModelTier][]ModelKey{
	TierOpus:   {"opus46", "opus45", "opus41", "opus40"},
	TierSonnet: {"sonnet46", "sonnet45", "sonnet40", "sonnet37", "sonnet35"},
	TierHaiku:  {"haiku45", "haiku35"},
}

// GetProviderModelCandidates returns ordered candidate model IDs for a provider/tier.
func GetProviderModelCandidates(provider string, tier ModelTier) []ModelName {
	if usesLiveCatalogOnly(provider) {
		return nil
	}
	seen := make(map[string]bool)
	var ordered []ModelName

	if pk, ok := preferredKeys[provider]; ok {
		if key, ok := pk[tier]; ok {
			if cfg, ok := AllModelConfigs[key]; ok {
				if m := cfg[provider]; m != "" && !seen[m] {
					seen[m] = true
					ordered = append(ordered, m)
				}
			}
		}
	}

	for _, key := range fallbackKeys[tier] {
		if cfg, ok := AllModelConfigs[key]; ok {
			if m := cfg[provider]; m != "" && !seen[m] {
				seen[m] = true
				ordered = append(ordered, m)
			}
		}
	}
	return ordered
}

func catalogModelIDs(catalog *ModelCatalog, provider string) []string {
	models := ModelsForProvider(catalog, provider)
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	return ids
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func providerModelPool(provider string) []ModelName {
	if usesLiveCatalogOnly(provider) {
		return nil
	}
	seen := make(map[string]bool)
	var ordered []ModelName
	for _, key := range modelKeys {
		m := AllModelConfigs[key][provider]
		if m != "" && !seen[m] {
			seen[m] = true
			ordered = append(ordered, m)
		}
	}
	return ordered
}

// GetPreferredProviderModel returns the preferred model for a provider/tier.
func GetPreferredProviderModel(provider string, tier ModelTier, catalog *ModelCatalog) ModelName {
	if catalog == nil {
		c := LoadModelCatalogSync("")
		catalog = &c
	}

	ids := catalogModelIDs(catalog, provider)
	if len(ids) > 0 {
		candidates := GetProviderModelCandidates(provider, tier)
		for _, c := range candidates {
			if contains(ids, c) {
				return c
			}
		}
		return ids[0]
	}

	candidates := GetProviderModelCandidates(provider, tier)
	if len(candidates) > 0 {
		return candidates[0]
	}

	if usesLiveCatalogOnly(provider) {
		return ""
	}

	pool := providerModelPool(provider)
	if len(pool) > 0 {
		return pool[0]
	}
	return HawkSonnet46Config[provider]
}

// GetProviderDefaultModel returns the default model for a provider from catalog when available.
func GetProviderDefaultModel(provider string, catalog *ModelCatalog) ModelName {
	if catalog == nil {
		c := LoadModelCatalogSync("")
		catalog = &c
	}
	ids := catalogModelIDs(catalog, provider)
	if len(ids) > 0 {
		return ids[0]
	}
	if IsLiveOnlyProvider(provider) {
		return ""
	}
	pool := providerModelPool(provider)
	if len(pool) > 0 {
		return pool[0]
	}
	return HawkSonnet46Config[provider]
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
