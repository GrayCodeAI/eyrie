package live

import "encoding/json"

// Entry is one model row from a live provider list API.
type Entry struct {
	ID               string
	DisplayName      string
	Description      string
	OwnedBy          string
	ContextWindow    int
	MaxOutput        int
	InputPricePer1M  float64
	OutputPricePer1M float64
	// CachedReadPricePer1M is the price per 1M tokens for cached reads (prompt caching).
	CachedReadPricePer1M float64
	// CachedWritePricePer1M is the price per 1M tokens for cached writes (cache creation).
	CachedWritePricePer1M float64
	// Tiered pricing: when input tokens exceed TierThreshold, TieredInputPricePer1M applies.
	// Same for output. 0 means no tiering.
	TierThreshold            int     `json:"tier_threshold,omitempty"`
	TieredInputPricePer1M    float64 `json:"tiered_input_price_per_1m,omitempty"`
	TieredOutputPricePer1M   float64 `json:"tiered_output_price_per_1m,omitempty"`
	TieredCachedReadPer1M    float64 `json:"tiered_cached_read_per_1m,omitempty"`
	TieredCachedWritePer1M   float64 `json:"tiered_cached_write_per_1m,omitempty"`
	Features                 []string
	// Protocol indicates which API protocol the model uses ("openai" or "anthropic").
	// Populated from live API metadata when available; empty means unknown/heuristic.
	Protocol string
	// RawJSON is the provider's full model object from the list API (preserved verbatim).
	RawJSON json.RawMessage

	// Capability fields (populated from live API responses that include capabilities).
	MaxInputTokens     int    `json:"max_input_tokens,omitempty"`
	ThinkingEnabled    bool   `json:"thinking_enabled,omitempty"`     // model supports thinking with explicit budget
	ThinkingAdaptive   bool   `json:"thinking_adaptive,omitempty"`   // model supports adaptive thinking
	EffortSupported    bool   `json:"effort_supported,omitempty"`    // model supports effort parameter
	EffortLevels       string `json:"effort_levels,omitempty"`       // comma-separated: "low,medium,high,xhigh,max"
	StructuredOutput   bool   `json:"structured_output,omitempty"`   // model supports native JSON schema output
	CodeExecution      bool   `json:"code_execution,omitempty"`     // model supports server-side code execution
	CitationsSupported bool   `json:"citations_supported,omitempty"` // model supports citations
	PDFInput           bool   `json:"pdf_input,omitempty"`          // model supports PDF input
	ImageInput         bool   `json:"image_input,omitempty"`        // model supports image input
}
