package runtime

import (
	"context"

	"github.com/GrayCodeAI/eyrie/config"
)

// Credential types re-exported for host apps.
type (
	CredentialInference      = config.CredentialInference
	CredentialProviderOption = config.CredentialProviderOption
	CredentialResolveResult  = config.CredentialResolveResult
)

// ValidateKeyFormat rejects empty/placeholder keys before provider selection.
func ValidateKeyFormat(secret string) error {
	return config.ValidateKeyFormat(secret)
}

// ResolveCredential validates format and lists registered providers (gateway must be chosen first).
func ResolveCredential(ctx context.Context, secret string) CredentialResolveResult {
	return config.ResolveCredential(ctx, secret)
}

// InferenceForProvider returns save metadata for a gateway selected in setup UI.
func InferenceForProvider(providerID string) (CredentialInference, error) {
	return config.InferenceForProvider(providerID)
}

// ListCredentialProviders returns all registry providers for setup UIs.
func ListCredentialProviders() []CredentialProviderOption {
	return config.ListCredentialProviders()
}

// InferCredentialsFromAPIKey is deprecated; use InferenceForProvider after gateway selection.
func InferCredentialsFromAPIKey(ctx context.Context, secret string) []CredentialInference {
	return config.InferCredentialsFromAPIKey(ctx, secret)
}

// ProbeCredential validates a key against the provider HTTP API.
func ProbeCredential(ctx context.Context, envKey, secret string) error {
	return config.ProbeCredential(ctx, envKey, secret)
}

// CommitCredential runs format + provider validation + probe (no persistence).
func CommitCredential(ctx context.Context, inference CredentialInference, secret string) error {
	return config.CommitCredential(ctx, inference, secret)
}

// LocalCredentialInference returns setup metadata for no-key providers.
func LocalCredentialInference(providerID string) (CredentialInference, error) {
	return config.LocalCredentialInference(providerID)
}

// SaveCredential validates, probes, and stores a credential in keychain.
func SaveCredential(ctx context.Context, inference CredentialInference, secret string) error {
	if err := config.CommitCredential(ctx, inference, secret); err != nil {
		return err
	}
	return SetCredential(ctx, inference.EnvVar, secret)
}
