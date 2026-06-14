package config

import (
	"context"
	"os"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// DiscoveryCredentials loads API keys from the OS secret store (not process env or .env files),
// merged with non-secret routing from ~/.hawk/provider.json (e.g. MiMo Token Plan region/base URL).
func DiscoveryCredentials(ctx context.Context) catalog.Credentials {
	if ctx == nil {
		ctx = context.Background()
	}
	env := filterPlaceholderEnv(credentials.APIKeysMap(ctx, credentials.DefaultStore()))
	mergeDiscoveryEnvFromProviderConfig(env)
	return catalog.Credentials{APIKeys: env}
}

// mergeDiscoveryEnvFromProviderConfig adds provider.json fields required for live catalog fetch.
// API keys always come from the secret store; values here must not override stored keys.
func mergeDiscoveryEnvFromProviderConfig(env map[string]string) {
	if env == nil {
		return
	}
	cfg := LoadProviderConfig("")
	if cfg == nil {
		return
	}
	r := strings.TrimSpace(cfg.XiaomiMimoTokenPlanRegion)
	if r == "" {
		r = strings.TrimSpace(os.Getenv(EnvXiaomiMimoTokenPlanRegion))
	}
	if norm, err := xiaomi.NormalizeRegion(r); err == nil {
		env[EnvXiaomiMimoTokenPlanRegion] = string(norm)
	}
	if base, err := ResolveXiaomiMimoOpenAIBase(ProviderXiaomiMimoTokenPlan, cfg); err == nil && base != "" {
		env[EnvXiaomiMimoTokenPlanBaseURL] = base
	}
	if paygBase := strings.TrimSpace(cfg.XiaomiMimoPaygBaseURL); paygBase != "" {
		env[EnvXiaomiMimoPaygBaseURL] = paygBase
	}
	if ollamaBase := NormalizeOllamaOpenAIBaseURL(AsNonEmptyString(cfg.OllamaBaseURL)); ollamaBase != "" {
		env["OLLAMA_BASE_URL"] = ollamaBase
	}
	mergeDeploymentDiscoveryEnv(env, cfg.Deployments)
}

func mergeDeploymentDiscoveryEnv(env map[string]string, deployments map[string]DeploymentConfig) {
	for id, dep := range deployments {
		switch id {
		case "openai-azure":
			setDiscoveryEnv(env, "AZURE_OPENAI_ENDPOINT", dep.Endpoint)
			setDiscoveryEnv(env, "AZURE_OPENAI_API_VERSION", dep.APIVersion)
			setDiscoveryEnv(env, "AZURE_OPENAI_DEPLOYMENT", firstNonEmptyDeploymentMapping(dep.ModelMappings))
		case "anthropic-bedrock":
			setDiscoveryEnv(env, "AWS_ACCESS_KEY_ID", dep.AccessKeyID)
			setDiscoveryEnv(env, "AWS_SESSION_TOKEN", dep.SessionToken)
			setDiscoveryEnv(env, "AWS_REGION", dep.Region)
		case "anthropic-vertex", "gemini-vertex":
			setDiscoveryEnv(env, "VERTEX_PROJECT_ID", dep.ProjectID)
			setDiscoveryEnv(env, "VERTEX_REGION", dep.Region)
		}
	}
}

func setDiscoveryEnv(env map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.TrimSpace(env[key]) != "" {
		return
	}
	env[key] = value
}

func firstNonEmptyDeploymentMapping(m map[string]string) string {
	for _, value := range m {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}

func filterPlaceholderEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return env
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		if strings.TrimSpace(v) != "" && !LooksLikePlaceholderSecret(v) {
			out[k] = v
		}
	}
	return out
}
