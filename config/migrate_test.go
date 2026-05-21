package config

import "testing"

func TestEnsureDeploymentConfigV2FromLegacyAnthropic(t *testing.T) {
	cfg := &ProviderConfig{AnthropicAPIKey: "sk-test-1234567890"}
	out := EnsureDeploymentConfigV2(cfg)
	if out.ConfigVersion != 2 {
		t.Fatalf("config_version = %d, want 2", out.ConfigVersion)
	}
	if _, ok := out.Deployments["anthropic-direct"]; !ok {
		t.Fatal("expected anthropic-direct deployment")
	}
}
