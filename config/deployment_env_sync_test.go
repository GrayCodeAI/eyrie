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

func TestBuildRoutingPolicyFromDeployments_ZAI(t *testing.T) {
	deployments := map[string]DeploymentConfig{
		"z-ai-direct": {APIKey: "zai-test"},
	}
	policy := BuildRoutingPolicyFromDeployments(deployments)
	if len(policy.Providers["z-ai"]) == 0 {
		t.Fatalf("expected z-ai routing, got %+v", policy.Providers)
	}
	if policy.Providers["z-ai"][0].Deployments[0].DeploymentID != "z-ai-direct" {
		t.Fatalf("expected z-ai-direct deployment, got %+v", policy.Providers["z-ai"])
	}
}

func TestBuildRoutingPolicyFromDeployments_CanopyWave(t *testing.T) {
	deployments := map[string]DeploymentConfig{
		"canopywave": {APIKey: "cw-test"},
		"openrouter": {APIKey: "or-test"},
	}
	policy := BuildRoutingPolicyFromDeployments(deployments)
	if len(policy.Providers["canopywave"]) == 0 {
		t.Fatalf("expected canopywave routing, got %+v", policy.Providers)
	}
	if policy.Providers["canopywave"][0].Deployments[0].DeploymentID != "canopywave" {
		t.Fatalf("expected canopywave deployment, got %+v", policy.Providers["canopywave"])
	}
	if len(policy.Providers["z-ai"]) != 0 {
		t.Fatalf("z-ai should not own canopywave routing, got %+v", policy.Providers["z-ai"])
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
