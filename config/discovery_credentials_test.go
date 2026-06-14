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

func TestDiscoveryCredentials_IncludesTokenPlanRegionFromProviderConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HAWK_CONFIG_DIR", dir)

	cfg := &ProviderConfig{Version: "2", XiaomiMimoTokenPlanRegion: "sgp"}
	if err := SaveProviderConfig(cfg, ""); err != nil {
		t.Fatal(err)
	}

	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })
	_ = store.Set(context.Background(), credentials.AccountForEnv(EnvXiaomiMimoTokenPlanAPIKey), "tp-test-key-1234567890123456")

	creds := DiscoveryCredentials(context.Background())
	if creds.APIKeys[EnvXiaomiMimoTokenPlanRegion] != "sgp" {
		t.Fatalf("region = %q, want sgp", creds.APIKeys[EnvXiaomiMimoTokenPlanRegion])
	}
	wantBase := "https://token-plan-sgp.xiaomimimo.com/v1"
	if creds.APIKeys[EnvXiaomiMimoTokenPlanBaseURL] != wantBase {
		t.Fatalf("base = %q, want %s", creds.APIKeys[EnvXiaomiMimoTokenPlanBaseURL], wantBase)
	}
	if creds.APIKeys[EnvXiaomiMimoTokenPlanAPIKey] == "" {
		t.Fatal("expected token plan API key from store")
	}
}
