package catalog

import (
	"strings"
	"testing"
)

func TestGetModelDeprecationWarning_KnownDeprecated(t *testing.T) {
	tests := []struct {
		modelID  string
		provider string
		wantSub  string
	}{
		{"claude-3-7-sonnet-20250219", "anthropic", "Claude 3.7 Sonnet"},
		{"claude-3-5-haiku-20241022", "anthropic", "Claude 3.5 Haiku"},
		{"claude-3-opus-20240229", "anthropic", "Claude 3 Opus"},
	}

	for _, tt := range tests {
		warning := GetModelDeprecationWarning(tt.modelID, tt.provider)
		if warning == "" {
			t.Errorf("GetModelDeprecationWarning(%q, %q) = empty, want warning containing %q", tt.modelID, tt.provider, tt.wantSub)
			continue
		}
		if !strings.Contains(warning, tt.wantSub) {
			t.Errorf("GetModelDeprecationWarning(%q, %q) = %q, want substring %q", tt.modelID, tt.provider, warning, tt.wantSub)
		}
		if !strings.Contains(warning, "retired") {
			t.Errorf("warning should mention 'retired': %q", warning)
		}
	}
}

func TestGetModelDeprecationWarning_NonDeprecated(t *testing.T) {
	tests := []struct {
		modelID  string
		provider string
	}{
		{"claude-sonnet-4-6", "anthropic"},
		{"claude-opus-4-6", "anthropic"},
		{"claude-haiku-4-5-20251001", "anthropic"},
		{"gpt-4o", "openai"},
		{"gemini-2.0-flash", "gemini"},
	}

	for _, tt := range tests {
		warning := GetModelDeprecationWarning(tt.modelID, tt.provider)
		if warning != "" {
			t.Errorf("GetModelDeprecationWarning(%q, %q) = %q, want empty", tt.modelID, tt.provider, warning)
		}
	}
}

func TestGetModelDeprecationWarning_WrongProvider(t *testing.T) {
	// claude-3-7-sonnet is deprecated on anthropic, but not necessarily on other providers
	warning := GetModelDeprecationWarning("claude-3-7-sonnet-20250219", "openai")
	if warning != "" {
		t.Errorf("expected no warning for wrong provider, got %q", warning)
	}
}

func TestGetModelDeprecationWarning_CaseInsensitive(t *testing.T) {
	// Should handle case insensitivity
	warning := GetModelDeprecationWarning("Claude-3-7-Sonnet-20250219", "anthropic")
	if warning == "" {
		t.Error("expected warning with mixed-case input")
	}
}

func TestDeprecatedModelsRegistry(t *testing.T) {
	// Verify the registry has expected entries
	if len(DeprecatedModels) == 0 {
		t.Fatal("DeprecatedModels should not be empty")
	}

	for key, entry := range DeprecatedModels {
		if entry.ModelName == "" {
			t.Errorf("DeprecatedModels[%q] has empty ModelName", key)
		}
		if len(entry.RetirementDates) == 0 {
			t.Errorf("DeprecatedModels[%q] has no retirement dates", key)
		}
		for provider, date := range entry.RetirementDates {
			if provider == "" {
				t.Errorf("DeprecatedModels[%q] has empty provider key", key)
			}
			if date == "" {
				t.Errorf("DeprecatedModels[%q][%q] has empty date", key, provider)
			}
		}
	}
}
