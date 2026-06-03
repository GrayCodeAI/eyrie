package credential

import "context"

// CredentialInference is save metadata for a gateway chosen in setup UI (no secret).
type CredentialInference struct {
	ProviderID   string `json:"provider_id"`
	DeploymentID string `json:"deployment_id"`
	EnvVar       string `json:"env_var"`
	DisplayName  string `json:"display_name"`
}

// InferCredentialsFromAPIKey is deprecated: setup is gateway-first (select provider, then paste key).
func InferCredentialsFromAPIKey(ctx context.Context, secret string) []CredentialInference {
	_ = ctx
	_ = secret
	return nil
}