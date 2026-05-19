package config

import (
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
)

// DeploymentConfigFromEnv builds deployment credentials from catalog env_fallbacks and env values.
func DeploymentConfigFromEnv(dep catalog.DeploymentV1, env map[string]string) DeploymentConfig {
	var dc DeploymentConfig
	for _, fb := range dep.EnvFallbacks {
		val := firstEnvFromMap(env, fb.Env)
		switch fb.Field {
		case "api_key":
			dc.APIKey = firstNonEmpty(dc.APIKey, val)
		case "base_url":
			dc.BaseURL = firstNonEmpty(dc.BaseURL, val)
		case "endpoint":
			dc.Endpoint = firstNonEmpty(dc.Endpoint, val)
		case "api_version":
			dc.APIVersion = firstNonEmpty(dc.APIVersion, val)
		case "project_id":
			dc.ProjectID = firstNonEmpty(dc.ProjectID, val)
		case "region":
			dc.Region = firstNonEmpty(dc.Region, val)
		case "token":
			dc.Token = firstNonEmpty(dc.Token, val)
		case "access_key_id":
			dc.AccessKeyID = firstNonEmpty(dc.AccessKeyID, val)
		case "secret_access_key":
			dc.SecretAccessKey = firstNonEmpty(dc.SecretAccessKey, val)
		case "session_token":
			dc.SessionToken = firstNonEmpty(dc.SessionToken, val)
		}
	}
	return dc
}

// DeploymentConfigured reports whether env supplies enough credentials for this deployment.
func DeploymentConfigured(deploymentID string, dep catalog.DeploymentV1, dc DeploymentConfig) bool {
	switch deploymentID {
	case "ollama-local":
		return strings.TrimSpace(dc.BaseURL) != ""
	default:
		return deploymentHasLiveCredentials(deploymentID, dc)
	}
}

func deploymentHasLiveCredentials(deploymentID string, dc DeploymentConfig) bool {
	switch deploymentID {
	case "anthropic-bedrock":
		return strings.TrimSpace(dc.AccessKeyID) != "" && strings.TrimSpace(dc.SecretAccessKey) != ""
	case "anthropic-vertex", "gemini-vertex":
		return strings.TrimSpace(dc.ProjectID) != "" && strings.TrimSpace(dc.Region) != "" &&
			(strings.TrimSpace(dc.Token) != "" || strings.TrimSpace(dc.APIKey) != "")
	default:
		return strings.TrimSpace(dc.APIKey) != "" || strings.TrimSpace(dc.Token) != "" ||
			strings.TrimSpace(dc.AccessKeyID) != ""
	}
}

func firstEnvFromMap(env map[string]string, keys []string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(env[k]); v != "" {
			return v
		}
	}
	return ""
}

// SyncProviderConfigFromCatalog merges catalog + env into provider.json deployments and routing.
func SyncProviderConfigFromCatalog(compiled *catalog.CompiledCatalogV1, env map[string]string) *ProviderConfig {
	cfg := LoadProviderConfig("")
	if cfg == nil {
		cfg = &ProviderConfig{}
	}
	if env == nil {
		env = map[string]string{}
	}
	deployments := map[string]DeploymentConfig{}
	if compiled != nil && compiled.Catalog != nil {
		ids := make([]string, 0, len(compiled.Catalog.Deployments))
		for id := range compiled.Catalog.Deployments {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			dep := compiled.Catalog.Deployments[id]
			dc := DeploymentConfigFromEnv(dep, env)
			if DeploymentConfigured(id, dep, dc) {
				deployments[id] = SanitizeDeploymentConfigForDisk(dc)
			}
		}
	}
	cfg.Deployments = deployments
	cfg.ConfigVersion = 2
	cfg.Routing = BuildRoutingPolicyFromDeployments(deployments)
	return cfg
}
