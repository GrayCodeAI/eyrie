// Package opencodego — static model metadata from https://opencode.ai/docs/go/
//
// The /v1/models API returns only {id, object, created, owned_by}.
// Pricing, protocol, context windows, and tiered pricing come from the docs.
// Update this file when OpenCode Go changes pricing or adds models.
//
// Last updated: 2026-06-14

package opencodego

// ModelMetadata holds static per-model data from the docs.
type ModelMetadata struct {
	Protocol    string  // "openai" or "anthropic"
	InputPer1M  float64 // USD per 1M input tokens
	OutputPer1M float64 // USD per 1M output tokens
	CachedRead  float64 // USD per 1M cached read tokens (0 if none)
	CachedWrite float64 // USD per 1M cached write tokens (0 if none)
	Context     int     // context window tokens
	MaxOutput   int     // max output tokens
	// Tiered pricing (Qwen models: different rate above threshold)
	TierThreshold     int     // 0 = no tiering (e.g., 256000 for Qwen)
	TieredInputPer1M  float64 // rate above threshold
	TieredOutputPer1M float64
	TieredCachedRead  float64
	TieredCachedWrite float64
}

// KnownModels is the static metadata map keyed by model ID.
// Merge with live /v1/models response — new models not listed here
// still work, just without pricing/protocol metadata.
var KnownModels = map[string]ModelMetadata{
	// --- Anthropic protocol (uses /v1/messages) ---
	"minimax-m3": {
		Protocol: "anthropic", InputPer1M: 0.30, OutputPer1M: 1.20,
		CachedRead: 0.06, Context: 128000, MaxOutput: 8000,
	},
	"minimax-m2.7": {
		Protocol: "anthropic", InputPer1M: 0.30, OutputPer1M: 1.20,
		CachedRead: 0.06, CachedWrite: 0.375, Context: 128000, MaxOutput: 8000,
	},
	"minimax-m2.5": {
		Protocol: "anthropic", InputPer1M: 0.30, OutputPer1M: 1.20,
		CachedRead: 0.06, CachedWrite: 0.375, Context: 128000, MaxOutput: 8000,
	},
	"qwen3.7-max": {
		Protocol: "anthropic", InputPer1M: 2.50, OutputPer1M: 7.50,
		CachedRead: 0.50, CachedWrite: 3.125, Context: 128000, MaxOutput: 8000,
	},
	"qwen3.7-plus": {
		Protocol: "anthropic", InputPer1M: 0.40, OutputPer1M: 1.60,
		CachedRead: 0.04, CachedWrite: 0.50, Context: 128000, MaxOutput: 8000,
		TierThreshold: 256000, TieredInputPer1M: 1.20, TieredOutputPer1M: 4.80,
		TieredCachedRead: 0.12, TieredCachedWrite: 1.50,
	},
	"qwen3.6-plus": {
		Protocol: "anthropic", InputPer1M: 0.50, OutputPer1M: 3.00,
		CachedRead: 0.05, CachedWrite: 0.625, Context: 128000, MaxOutput: 8000,
		TierThreshold: 256000, TieredInputPer1M: 2.00, TieredOutputPer1M: 6.00,
		TieredCachedRead: 0.20, TieredCachedWrite: 2.50,
	},
	"qwen3.5-plus": {
		Protocol: "anthropic", InputPer1M: 0.50, OutputPer1M: 3.00,
		CachedRead: 0.05, Context: 128000, MaxOutput: 8000,
	},

	// --- OpenAI protocol (uses /v1/chat/completions) ---
	"glm-5.1": {
		Protocol: "openai", InputPer1M: 1.40, OutputPer1M: 4.40,
		CachedRead: 0.26, Context: 128000, MaxOutput: 8000,
	},
	"glm-5": {
		Protocol: "openai", InputPer1M: 1.00, OutputPer1M: 3.20,
		CachedRead: 0.20, Context: 128000, MaxOutput: 8000,
	},
	"kimi-k2.7-code": {
		Protocol: "openai", InputPer1M: 0.95, OutputPer1M: 4.00,
		CachedRead: 0.19, Context: 128000, MaxOutput: 8000,
	},
	"kimi-k2.6": {
		Protocol: "openai", InputPer1M: 0.95, OutputPer1M: 4.00,
		CachedRead: 0.16, Context: 128000, MaxOutput: 8000,
	},
	"kimi-k2.5": {
		Protocol: "openai", InputPer1M: 0.95, OutputPer1M: 4.00,
		Context: 128000, MaxOutput: 8000,
	},
	"deepseek-v4-pro": {
		Protocol: "openai", InputPer1M: 1.74, OutputPer1M: 3.48,
		CachedRead: 0.0145, Context: 128000, MaxOutput: 8000,
	},
	"deepseek-v4-flash": {
		Protocol: "openai", InputPer1M: 0.14, OutputPer1M: 0.28,
		CachedRead: 0.0028, Context: 128000, MaxOutput: 8000,
	},
	"mimo-v2.5": {
		Protocol: "openai", InputPer1M: 0.14, OutputPer1M: 0.28,
		CachedRead: 0.0028, Context: 128000, MaxOutput: 8000,
	},
	"mimo-v2.5-pro": {
		Protocol: "openai", InputPer1M: 1.74, OutputPer1M: 3.48,
		CachedRead: 0.0145, Context: 128000, MaxOutput: 8000,
	},
	"mimo-v2-pro": {
		Protocol: "openai", InputPer1M: 1.74, OutputPer1M: 3.48,
		Context: 128000, MaxOutput: 8000,
	},
	"mimo-v2-omni": {
		Protocol: "openai", InputPer1M: 0.14, OutputPer1M: 0.28,
		Context: 128000, MaxOutput: 8000,
	},
	"hy3-preview": {
		Protocol: "openai", InputPer1M: 0.50, OutputPer1M: 2.00,
		Context: 128000, MaxOutput: 8000,
	},
}

// MetadataForModel returns static metadata for a model, or zero value if unknown.
func MetadataForModel(modelID string) (ModelMetadata, bool) {
	m, ok := KnownModels[modelID]
	return m, ok
}

// AllKnownModelIDs returns all model IDs from the static metadata.
func AllKnownModelIDs() []string {
	ids := make([]string, 0, len(KnownModels))
	for id := range KnownModels {
		ids = append(ids, id)
	}
	return ids
}
