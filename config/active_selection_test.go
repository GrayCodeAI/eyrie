package config

import "testing"

func TestSetProviderModel_WritesScopedAndActiveFields(t *testing.T) {
	t.Parallel()
	cfg := &ProviderConfig{}
	SetProviderModel(cfg, ProviderAnthropic, "claude-sonnet-4-6")
	if cfg.ActiveModel != "claude-sonnet-4-6" {
		t.Fatalf("active_model: %q", cfg.ActiveModel)
	}
	if cfg.AnthropicModel != "claude-sonnet-4-6" {
		t.Fatalf("anthropic_model: %q", cfg.AnthropicModel)
	}
	if cfg.ActiveProvider != ProviderAnthropic {
		t.Fatalf("active_provider: %q", cfg.ActiveProvider)
	}
}

func TestActiveModel_PrefersActiveModelField(t *testing.T) {
	t.Parallel()
	cfg := &ProviderConfig{
		ActiveProvider: ProviderOpenAI,
		OpenAIModel:    "gpt-4o",
		ActiveModel:    "anthropic/claude-sonnet-4-6",
	}
	if got := ActiveModel(cfg); got != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("got %q", got)
	}
}
