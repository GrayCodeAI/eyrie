package config

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/eyrie/credentials"
)

func TestDiscoveryCredentials_UsesStoreNotProcessEnv(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })

	t.Setenv("OPENROUTER_API_KEY", "sk-or-from-shell-should-not-appear")
	_ = store.Set(context.Background(), credentials.AccountForEnv("ANTHROPIC_API_KEY"), "sk-ant-store-only-key-1234567890")

	creds := DiscoveryCredentials(context.Background())
	if creds.APIKeys["OPENROUTER_API_KEY"] != "" {
		t.Fatal("DiscoveryCredentials must not read API keys from process env")
	}
	if creds.APIKeys["ANTHROPIC_API_KEY"] == "" {
		t.Fatal("expected ANTHROPIC_API_KEY from store")
	}
}
