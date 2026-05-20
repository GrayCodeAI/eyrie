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
	Features         []string
	// RawJSON is the provider's full model object from the list API (preserved verbatim).
	RawJSON json.RawMessage
}
