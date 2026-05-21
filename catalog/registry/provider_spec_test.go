package registry_test

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestAllProviders_Count(t *testing.T) {
	if n := len(registry.All()); n != 11 {
		t.Fatalf("expected 11 providers, got %d", n)
	}
}

func TestCredentialRegistry_MatchesAll(t *testing.T) {
	if len(registry.CredentialRegistry()) != len(registry.All()) {
		t.Fatal("credential registry should cover all provider specs")
	}
}

func TestLiveFetcherKeys_AllProviders(t *testing.T) {
	keys := registry.LiveFetcherKeys()
	if len(keys) != 11 {
		t.Fatalf("expected 11 live fetcher keys, got %d", len(keys))
	}
}

func TestOllamaStrategy_LiveOnly(t *testing.T) {
	spec, ok := registry.SpecByProviderID("ollama")
	if !ok {
		t.Fatal("missing ollama spec")
	}
	if spec.ModelStrategy != registry.StrategyLiveOnly {
		t.Fatalf("ollama strategy = %q", spec.ModelStrategy)
	}
}
