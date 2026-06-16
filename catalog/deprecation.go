package catalog

import (
	"fmt"
	"strings"
)

// DeprecationEntry holds deprecation metadata for a model.
type DeprecationEntry struct {
	ModelName       string
	RetirementDates map[string]string // provider → date string, empty = not deprecated
}

// DeprecatedModels lists deprecated models and their per-provider retirement dates.
var DeprecatedModels = map[string]DeprecationEntry{
	"claude-3-opus": {
		ModelName:       "Claude 3 Opus",
		RetirementDates: map[string]string{"anthropic": "January 5, 2026"},
	},
	"claude-3-7-sonnet": {
		ModelName:       "Claude 3.7 Sonnet",
		RetirementDates: map[string]string{"anthropic": "February 19, 2026"},
	},
	"claude-3-5-haiku": {
		ModelName:       "Claude 3.5 Haiku",
		RetirementDates: map[string]string{"anthropic": "February 19, 2026"},
	},
}

// GetModelDeprecationWarning returns a deprecation warning or empty string.
func GetModelDeprecationWarning(modelID, provider string) string {
	canonical := AnthropicNameToCanonical(strings.ToLower(modelID))
	for key, entry := range DeprecatedModels {
		date, ok := entry.RetirementDates[provider]
		if !strings.Contains(canonical, key) || !ok || date == "" {
			continue
		}
		return fmt.Sprintf("[warn] %s will be retired on %s. Consider switching to a newer model.", entry.ModelName, date)
	}
	return ""
}
