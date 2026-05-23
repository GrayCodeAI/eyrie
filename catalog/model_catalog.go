package catalog

// LiveProviderEnrichment records a live provider API fetch during catalog discovery.
type LiveProviderEnrichment struct {
	Provider   string `json:"provider"`
	ModelCount int    `json:"model_count"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}
