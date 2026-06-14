package catalog

import "encoding/json"

// ModelCatalogEntry represents a single model in the catalog.
type ModelCatalogEntry struct {
	ID               string  `json:"id"`
	InputPricePer1M  float64 `json:"input_price_per_1m"`
	OutputPricePer1M float64 `json:"output_price_per_1m"`
	// CachedReadPricePer1M is the price per 1M tokens for cache hits (prompt caching).
	CachedReadPricePer1M float64 `json:"cached_read_price_per_1m,omitempty"`
	// CachedWritePricePer1M is the price per 1M tokens for cache creation/writes.
	CachedWritePricePer1M float64 `json:"cached_write_price_per_1m,omitempty"`
	// Tiered pricing: when input tokens exceed TierThreshold, higher rates apply.
	// 0 means no tiering. Used by Qwen3.7 Plus / Qwen3.6 Plus (256K threshold).
	TierThreshold          int             `json:"tier_threshold,omitempty"`
	TieredInputPricePer1M  float64         `json:"tiered_input_price_per_1m,omitempty"`
	TieredOutputPricePer1M float64         `json:"tiered_output_price_per_1m,omitempty"`
	TieredCachedReadPer1M  float64         `json:"tiered_cached_read_per_1m,omitempty"`
	TieredCachedWritePer1M float64         `json:"tiered_cached_write_per_1m,omitempty"`
	ContextWindow          int             `json:"context_window"`
	MaxOutput              int             `json:"max_output"`
	ServerTools            []string        `json:"server_tools,omitempty"`
	DisplayName            string          `json:"display_name,omitempty"`
	Description            string          `json:"description,omitempty"`
	Owner                  string          `json:"owner,omitempty"` // upstream vendor (API owned_by)
	LiveMetadata           json.RawMessage `json:"live_metadata,omitempty"`
}

// ModelCatalog holds the full model catalog with per-provider entries.
type ModelCatalog struct {
	UpdatedAt string                         `json:"updated_at"`
	Source    string                         `json:"source"`
	Providers map[string][]ModelCatalogEntry `json:"providers"`
}
