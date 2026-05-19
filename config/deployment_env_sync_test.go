package config

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
)

func TestBuildRoutingPolicyFromDeployments_OpenAI(t *testing.T) {
	deployments := map[string]DeploymentConfig{
		"openai-direct": {APIKey: "sk-test"},
		"openai-azure":  {APIKey: "az-test"},
		"openrouter":    {APIKey: "or-test"},
	}
	policy := BuildRoutingPolicyFromDeployments(deployments)
	if len(policy.Providers["openai"]) < 2 {
		t.Fatalf("expected openai stages with openrouter fallback, got %+v", policy.Providers["openai"])
	}
	primary := policy.Providers["openai"][0]
	if len(primary.Deployments) != 2 {
		t.Fatalf("expected weighted openai stage, got %+v", primary.Deployments)
	}
}

func TestSyncProviderConfigFromCatalog(t *testing.T) {
	bootstrap := catalog.BootstrapCatalogV1()
	compiled, err := catalog.CompileCatalogV1(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"ANTHROPIC_API_KEY": "sk-ant-test-key-12345"}
	cfg := SyncProviderConfigFromCatalog(compiled, env)
	if _, ok := cfg.Deployments["anthropic-direct"]; !ok {
		t.Fatal("expected anthropic-direct deployment from env")
	}
	if cfg.Deployments["anthropic-direct"].APIKey != "" {
		t.Fatal("expected sanitized deployment without api_key on disk")
	}
	if cfg.Routing == nil || len(cfg.Routing.Providers["anthropic"]) == 0 {
		t.Fatal("expected anthropic routing")
	}
}
