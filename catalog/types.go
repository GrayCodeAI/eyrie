package catalog

// ModelCatalogEntry represents a single model in the catalog.
type ModelCatalogEntry struct {
	ID              string   `json:"id"`
	InputPricePer1M float64  `json:"input_price_per_1m"`
	OutputPricePer1M float64 `json:"output_price_per_1m"`
	ContextWindow   int      `json:"context_window"`
	MaxOutput       int      `json:"max_output"`
	ServerTools     []string `json:"server_tools,omitempty"`
	DisplayName     string   `json:"display_name,omitempty"`
	Description     string   `json:"description,omitempty"`
}

// ModelCatalog holds the full model catalog with per-provider entries.
type ModelCatalog struct {
	UpdatedAt string                      `json:"updated_at"`
	Source    string                      `json:"source"`
	Providers map[string][]ModelCatalogEntry `json:"providers"`
}
