package adapters

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/credentials"
)

func TestResolveEnvSecret(t *testing.T) {
	store := &credentials.MapStore{
		Data: map[string]string{
			"openai_api_key": "sk-test-key",
		},
	}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	got := ResolveEnvSecret("OPENAI_API_KEY")
	if got != "sk-test-key" {
		t.Errorf("ResolveEnvSecret = %q, want sk-test-key", got)
	}
}

func TestResolveEnvSecret_NotFound(t *testing.T) {
	store := &credentials.MapStore{Data: map[string]string{}}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	got := ResolveEnvSecret("NONEXISTENT_KEY")
	if got != "" {
		t.Errorf("ResolveEnvSecret = %q, want empty", got)
	}
}

func TestResolveProviderModelEnvOverride(t *testing.T) {
	store := &credentials.MapStore{
		Data: map[string]string{
			"openai_model": "gpt-4o",
		},
	}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	got := ResolveProviderModelEnvOverride("openai")
	if got != "gpt-4o" {
		t.Errorf("ResolveProviderModelEnvOverride = %q, want gpt-4o", got)
	}
}

func TestResolveProviderModelEnvOverride_Empty(t *testing.T) {
	store := &credentials.MapStore{Data: map[string]string{}}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	got := ResolveProviderModelEnvOverride("openai")
	if got != "" {
		t.Errorf("ResolveProviderModelEnvOverride = %q, want empty", got)
	}
}
