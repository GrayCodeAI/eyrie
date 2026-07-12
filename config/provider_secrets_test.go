package config

import "testing"

func TestSanitizeProviderConfigForDiskRemovesLegacyAndDeploymentSecrets(t *testing.T) {
	original := ProviderConfig{
		OpenAIAPIKey: "sk-legacy", AnthropicAPIKey: "sk-ant-legacy",
		OpenAIBaseURL: "https://example.test", ActiveModel: "custom/model",
		Deployments: map[string]DeploymentConfig{
			"openai-direct": {APIKey: "sk-deployment", BaseURL: "https://deployment.test"},
		},
	}
	if !ProviderConfigContainsSecrets(original) {
		t.Fatal("expected legacy provider state to contain secrets")
	}
	sanitized := SanitizeProviderConfigForDisk(original)
	if ProviderConfigContainsSecrets(sanitized) {
		t.Fatalf("sanitized provider state still contains secrets: %+v", sanitized)
	}
	if sanitized.OpenAIBaseURL != original.OpenAIBaseURL || sanitized.ActiveModel != original.ActiveModel ||
		sanitized.Deployments["openai-direct"].BaseURL != "https://deployment.test" {
		t.Fatalf("sanitization lost routing metadata: %+v", sanitized)
	}
	if original.OpenAIAPIKey == "" || original.Deployments["openai-direct"].APIKey == "" {
		t.Fatal("sanitization mutated its input")
	}
}
