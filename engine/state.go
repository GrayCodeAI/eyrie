package engine

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
)

func (e *Engine) loadRuntimeState(ctx context.Context) (*catalog.CompiledCatalog, *config.ProviderConfig, error) {
	compiled, err := catalog.LoadCatalog(ctx, catalog.LoadCatalogOptions{CachePath: e.catalogPath, RequireCache: true})
	if err != nil {
		return nil, nil, &Error{Code: ErrorCatalogUnavailable, Operation: "load_state", Message: err.Error(), Cause: err}
	}
	persisted := config.LoadProviderConfig(e.providerConfigPath)
	if persisted == nil {
		persisted = &config.ProviderConfig{}
	}
	cfg := *persisted
	cfg.Deployments = buildDeployments(compiled, persisted.Deployments, e.credentialEnv(ctx, compiled))
	if cfg.Routing == nil {
		cfg.Routing = config.BuildRoutingPolicyFromDeployments(cfg.Deployments)
	}
	if len(cfg.Deployments) > 0 && cfg.ConfigVersion < 2 {
		cfg.ConfigVersion = 2
	}
	return compiled, &cfg, nil
}

func (e *Engine) credentialEnv(ctx context.Context, compiled *catalog.CompiledCatalog) map[string]string {
	out := make(map[string]string)
	if compiled == nil {
		return out
	}
	for _, envKey := range catalog.DiscoveryEnvKeysFromCatalog(compiled) {
		secret, err := e.secretStore.Get(ctx, credentials.AccountForEnv(envKey))
		if err == nil && strings.TrimSpace(secret) != "" && !config.LooksLikePlaceholderSecret(secret) {
			out[envKey] = secret
		}
	}
	return out
}

func buildDeployments(compiled *catalog.CompiledCatalog, persisted map[string]config.DeploymentConfig, env map[string]string) map[string]config.DeploymentConfig {
	out := make(map[string]config.DeploymentConfig)
	if compiled == nil || compiled.Catalog == nil {
		return out
	}
	for id, deployment := range compiled.Catalog.Deployments {
		derived := config.DeploymentConfigFromEnv(deployment, env)
		if existing, ok := persisted[id]; ok {
			derived = mergeDeployment(existing, derived)
		}
		if config.DeploymentConfigured(id, deployment, derived) {
			out[id] = derived
		}
	}
	return out
}

// mergeDeployment keeps only non-secret routing fields from disk while filling
// credential fields from the injected store. Legacy secret-bearing provider
// state is never accepted as a runtime credential source.
func mergeDeployment(persisted, derived config.DeploymentConfig) config.DeploymentConfig {
	out := config.SanitizeDeploymentConfigForDisk(persisted)
	if derived.APIKey != "" {
		out.APIKey = derived.APIKey
	}
	if derived.BaseURL != "" {
		out.BaseURL = derived.BaseURL
	}
	if derived.Endpoint != "" {
		out.Endpoint = derived.Endpoint
	}
	if derived.APIVersion != "" {
		out.APIVersion = derived.APIVersion
	}
	if derived.ProjectID != "" {
		out.ProjectID = derived.ProjectID
	}
	if derived.Region != "" {
		out.Region = derived.Region
	}
	if derived.Token != "" {
		out.Token = derived.Token
	}
	if derived.AccessKeyID != "" {
		out.AccessKeyID = derived.AccessKeyID
	}
	if derived.SecretAccessKey != "" {
		out.SecretAccessKey = derived.SecretAccessKey
	}
	if derived.SessionToken != "" {
		out.SessionToken = derived.SessionToken
	}
	if len(derived.ModelMappings) > 0 {
		out.ModelMappings = derived.ModelMappings
	}
	return out
}
