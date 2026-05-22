package credential

import (
	"context"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// CommitCredential validates and probes a credential before persistence (host calls before SetCredential).
func CommitCredential(ctx context.Context, inference CredentialInference, secret string) error {
	secret = strings.TrimSpace(secret)
	envKey := strings.TrimSpace(inference.EnvVar)
	if secret == "" || envKey == "" {
		return fmt.Errorf("commit credential: key and env var required")
	}
	if spec, ok := registry.DefaultRegistry.Get(inference.ProviderID); ok && !spec.RequiresKey {
		return CommitLocalCredential(ctx, inference, secret)
	}
	if err := ValidateKeyFormat(secret); err != nil {
		return err
	}
	if err := ValidateCredentialSecret(envKey, secret); err != nil {
		return err
	}
	return ProbeCredential(ctx, envKey, secret)
}

// ProviderIDForEnv maps an env var to registry provider id.
func ProviderIDForEnv(envKey string) string {
	if spec, ok := registry.DefaultRegistry.GetByEnv(strings.TrimSpace(envKey)); ok {
		return spec.ProviderID
	}
	return ""
}
