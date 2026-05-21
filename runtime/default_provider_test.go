package runtime

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
)

func TestDefaultModelProviderFilter_FromProviderConfig(t *testing.T) {
	cfg := &config.ProviderConfig{
		ActiveProvider:  "anthropic",
		AnthropicAPIKey: "sk-test",
	}
	p := config.DefaultProviderFromConfig(cfg)
	if catalog.CanonicalProviderID(p) != "anthropic" {
		t.Fatalf("expected anthropic, got %q", p)
	}
}

func TestDefaultModelProviderFilter_LoadDoesNotPanic(t *testing.T) {
	ctx := context.Background()
	_ = DefaultModelProviderFilter(ctx)
}
