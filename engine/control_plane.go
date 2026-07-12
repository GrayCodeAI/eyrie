package engine

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// ResolveCredential validates credential input and returns safe provider
// choices. Eyrie does not retain or return the supplied secret.
func (e *Engine) ResolveCredential(ctx context.Context, secret string) CredentialResolution {
	resolved := config.ResolveCredential(nonNilContext(ctx), secret)
	out := CredentialResolution{
		FormatOK: resolved.FormatOK, FormatError: resolved.FormatError,
		ProbeDisambiguationUsed: resolved.ProbeDisambiguationUsed,
		Providers:               make([]CredentialProvider, len(resolved.Providers)),
	}
	for i, provider := range resolved.Providers {
		out.Providers[i] = CredentialProvider{
			ProviderID: provider.ProviderID, DeploymentID: provider.DeploymentID,
			EnvVar: provider.EnvVar, DisplayName: provider.DisplayName,
			RequiresKey: provider.RequiresKey, Rank: provider.Rank,
		}
	}
	return out
}

// CredentialProviders lists safe provider choices for setup UIs.
func (e *Engine) CredentialProviders(context.Context) []CredentialProvider {
	providers := config.ListCredentialProviders()
	out := make([]CredentialProvider, len(providers))
	for i, provider := range providers {
		out[i] = CredentialProvider{
			ProviderID: provider.ProviderID, DeploymentID: provider.DeploymentID,
			EnvVar: provider.EnvVar, DisplayName: provider.DisplayName,
			RequiresKey: provider.RequiresKey, Rank: provider.Rank,
		}
	}
	return out
}

// Gateways returns safe configuration status using this Engine's injected
// credential store, catalog path, and provider state path.
func (e *Engine) Gateways(ctx context.Context) []Gateway {
	ctx = nonNilContext(ctx)
	selection := e.ActiveSelection(ctx)
	providerConfig := config.LoadProviderConfig(e.providerConfigPath)
	compiled, _ := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath})

	specs := registry.CredentialRegistry()
	out := make([]Gateway, 0, len(specs))
	for _, spec := range specs {
		providerSpec, _ := registry.SpecByProviderID(spec.ProviderID)
		envVars := append([]string{spec.EnvVar}, providerSpec.CredentialEnvFallbacks...)
		envVars = append(envVars, providerSpec.CredentialAliases...)
		configured := e.hasCredential(ctx, envVars)
		deploymentConfigured := providerConfig != nil && spec.DeploymentID != ""
		if deploymentConfigured {
			_, deploymentConfigured = providerConfig.Deployments[spec.DeploymentID]
		}
		modelCount := 0
		if compiled != nil {
			modelCount = len(catalog.ModelEntriesForProvider(compiled, spec.ProviderID))
		}
		out = append(out, Gateway{
			ID: spec.ProviderID, DisplayName: spec.DisplayName,
			DeploymentID: spec.DeploymentID, CredentialEnv: spec.EnvVar,
			RequiresKey: spec.RequiresKey, CredentialConfigured: configured,
			DeploymentConfigured: deploymentConfigured, ModelCount: modelCount,
			Active: NormalizeProviderID(selection.Provider) == NormalizeProviderID(spec.ProviderID),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (e *Engine) hasCredential(ctx context.Context, envVars []string) bool {
	for _, envVar := range envVars {
		envVar = strings.TrimSpace(envVar)
		if envVar == "" {
			continue
		}
		secret, err := e.secretStore.Get(ctx, credentials.AccountForEnv(envVar))
		if err == nil && strings.TrimSpace(secret) != "" {
			return true
		}
	}
	return false
}

func maskedCredential(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return strings.Repeat("•", len(secret))
	}
	return strings.Repeat("•", 8) + secret[len(secret)-4:]
}

func (e *Engine) credentialValue(ctx context.Context, envVar string) (string, error) {
	secret, err := e.secretStore.Get(nonNilContext(ctx), credentials.AccountForEnv(envVar))
	if errors.Is(err, credentials.ErrNotFound) {
		return "", nil
	}
	return secret, err
}
