package setup

import (
	"testing"

	"github.com/GrayCodeAI/eyrie/config"
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

func TestProviderForDeploymentAnthropicBedrockFromEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

	p, ok := ProviderForDeployment("anthropic-bedrock", config.DeploymentConfig{})
	if !ok {
		t.Fatal("expected bedrock deployment to be configured from env")
	}
	if p.Name() != "anthropic-bedrock" {
		t.Fatalf("provider name = %q, want anthropic-bedrock", p.Name())
	}
}

func TestProviderForDeploymentAnthropicBedrockRequiresCredentials(t *testing.T) {
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	if _, ok := ProviderForDeployment("anthropic-bedrock", config.DeploymentConfig{}); ok {
		t.Fatal("expected bedrock deployment to be unavailable without credentials")
	}
}
