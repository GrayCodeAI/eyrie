package setup

import (
	"context"
	"testing"

	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

func TestProviderForDeploymentAnthropicBedrockFromConfig(t *testing.T) {
	p, ok := ProviderForDeployment("anthropic-bedrock", config.DeploymentConfig{
		Region:          "us-east-1",
		AccessKeyID:     "AKIATEST",
		SecretAccessKey: "secret",
	})
	if !ok {
		t.Fatal("expected bedrock deployment to be configured")
	}
	if p.Name() != "anthropic-bedrock" {
		t.Fatalf("provider name = %q, want anthropic-bedrock", p.Name())
	}
}

func TestProviderForDeploymentAnthropicBedrockFromStore(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })
	ctx := context.Background()
	_ = store.Set(ctx, credentials.AccountForEnv("AWS_ACCESS_KEY_ID"), "AKIATEST")
	_ = store.Set(ctx, credentials.AccountForEnv("AWS_SECRET_ACCESS_KEY"), "secret")
	t.Setenv("AWS_REGION", "us-west-2")

	p, ok := ProviderForDeployment("anthropic-bedrock", config.DeploymentConfig{})
	if !ok {
		t.Fatal("expected bedrock deployment to be configured from credential store")
	}
	if p.Name() != "anthropic-bedrock" {
		t.Fatalf("provider name = %q, want anthropic-bedrock", p.Name())
	}
}

func TestProviderForDeploymentAnthropicBedrockRequiresCredentials(t *testing.T) {
	store := &credentials.MapStore{}
	credentials.SetDefaultStore(store)
	t.Cleanup(func() { credentials.SetDefaultStore(nil) })
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")

	if _, ok := ProviderForDeployment("anthropic-bedrock", config.DeploymentConfig{}); ok {
		t.Fatal("expected bedrock deployment to be unavailable without credentials")
	}
}
