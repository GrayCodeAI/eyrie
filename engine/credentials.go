package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// SaveCredential validates, stores, and probes a provider credential. The
// secret is persisted before the probe so a transient network failure does not
// discard user input. The returned error states that persistence occurred.
func (e *Engine) SaveCredential(ctx context.Context, providerID, secret string) (CredentialStatus, error) {
	ctx = nonNilContext(ctx)
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return CredentialStatus{}, invalid("save_credential", "eyrie engine: provider id is required")
	}
	inference, err := config.InferenceForProvider(providerID)
	if err != nil {
		return CredentialStatus{}, &Error{Code: ErrorInvalidRequest, Operation: "save_credential", Provider: providerID, Message: err.Error(), Cause: err}
	}
	prepared, err := config.PrepareCredentialForSave(inference, secret)
	if err != nil {
		return CredentialStatus{}, &Error{Code: ErrorInvalidRequest, Operation: "save_credential", Provider: providerID, Message: err.Error(), Cause: err}
	}
	envKey := strings.TrimSpace(inference.EnvVar)
	if envKey == "" {
		return CredentialStatus{}, invalid("save_credential", "eyrie engine: provider has no credential target")
	}
	if err := e.secretStore.Set(ctx, credentials.AccountForEnv(envKey), prepared); err != nil {
		return CredentialStatus{}, &Error{Code: ErrorInternal, Operation: "save_credential", Provider: providerID, Message: "eyrie engine: could not save credential", Cause: err}
	}
	status := CredentialStatus{ProviderID: providerID, EnvVar: envKey, Configured: true}
	if spec, ok := registry.DefaultRegistry.Get(providerID); ok && !spec.RequiresKey {
		if err := config.ProbeLocalCredential(ctx, envKey, prepared); err != nil {
			return status, &Error{Code: ErrorProviderUnavailable, Operation: "probe_credential", Provider: providerID, Message: fmt.Sprintf("%v (value saved in keychain)", err), Cause: err}
		}
		status.Verified = true
		return status, nil
	}
	if err := config.ProbeCredential(ctx, envKey, prepared); err != nil {
		return status, &Error{Code: ErrorAuthentication, Operation: "probe_credential", Provider: providerID, Message: fmt.Sprintf("%v (key saved in keychain)", err), Cause: err}
	}
	status.Verified = true
	return status, nil
}

// RemoveCredential deletes a provider credential from the configured store.
func (e *Engine) RemoveCredential(ctx context.Context, providerID string) error {
	ctx = nonNilContext(ctx)
	inference, err := config.InferenceForProvider(strings.TrimSpace(providerID))
	if err != nil {
		return &Error{Code: ErrorInvalidRequest, Operation: "remove_credential", Provider: providerID, Message: err.Error(), Cause: err}
	}
	if err := e.secretStore.Delete(ctx, credentials.AccountForEnv(inference.EnvVar)); err != nil && !errors.Is(err, credentials.ErrNotFound) {
		return &Error{Code: ErrorInternal, Operation: "remove_credential", Provider: providerID, Message: "eyrie engine: could not remove credential", Cause: err}
	}
	return nil
}

// CredentialStatus reports whether a provider credential is configured.
func (e *Engine) CredentialStatus(ctx context.Context, providerID string) (CredentialStatus, error) {
	ctx = nonNilContext(ctx)
	providerID = strings.TrimSpace(providerID)
	inference, err := config.InferenceForProvider(providerID)
	if err != nil {
		return CredentialStatus{}, &Error{Code: ErrorInvalidRequest, Operation: "credential_status", Provider: providerID, Message: err.Error(), Cause: err}
	}
	secret, err := e.credentialValue(ctx, inference.EnvVar)
	if err == nil {
		configured := strings.TrimSpace(secret) != ""
		return CredentialStatus{ProviderID: providerID, EnvVar: inference.EnvVar, Configured: configured, Masked: maskedCredential(secret)}, nil
	}
	return CredentialStatus{}, &Error{Code: ErrorInternal, Operation: "credential_status", Provider: providerID, Message: "eyrie engine: could not read credential status", Cause: err}
}
