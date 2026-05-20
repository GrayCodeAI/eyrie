package catalog

import (
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// DefaultDeploymentEnvFallbacks seeds env_fallbacks per deployment until the published catalog includes them.
var DefaultDeploymentEnvFallbacks = func() map[string][]EnvFallbackV1 {
	base := map[string][]EnvFallbackV1{
	"anthropic-direct": {
		{Field: "api_key", Env: []string{"ANTHROPIC_API_KEY"}},
		{Field: "base_url", Env: []string{"ANTHROPIC_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"}},
	},
	"anthropic-bedrock": {
		{Field: "access_key_id", Env: []string{"AWS_ACCESS_KEY_ID"}},
		{Field: "secret_access_key", Env: []string{"AWS_SECRET_ACCESS_KEY"}},
		{Field: "session_token", Env: []string{"AWS_SESSION_TOKEN"}},
		{Field: "region", Env: []string{"AWS_REGION", "AWS_DEFAULT_REGION"}},
	},
	"anthropic-vertex": {
		{Field: "project_id", Env: []string{"VERTEX_PROJECT_ID"}},
		{Field: "region", Env: []string{"VERTEX_REGION"}},
		{Field: "token", Env: []string{"VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"}},
	},
	"openai-direct": {
		{Field: "api_key", Env: []string{"OPENAI_API_KEY"}},
		{Field: "base_url", Env: []string{"OPENAI_BASE_URL", "OPENAI_API_BASE"}},
	},
	"openai-azure": {
		{Field: "api_key", Env: []string{"AZURE_OPENAI_API_KEY", "OPENAI_API_KEY"}},
		{Field: "endpoint", Env: []string{"AZURE_OPENAI_ENDPOINT"}},
		{Field: "api_version", Env: []string{"AZURE_OPENAI_API_VERSION"}},
	},
	"gemini-direct": {
		{Field: "api_key", Env: []string{"GEMINI_API_KEY"}},
		{Field: "base_url", Env: []string{"GEMINI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"}},
	},
	"gemini-vertex": {
		{Field: "project_id", Env: []string{"VERTEX_PROJECT_ID"}},
		{Field: "region", Env: []string{"VERTEX_REGION"}},
		{Field: "token", Env: []string{"VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"}},
	},
	"grok-direct": {
		{Field: "api_key", Env: []string{"XAI_API_KEY", "GROK_API_KEY"}},
		{Field: "base_url", Env: []string{"GROK_BASE_URL", "XAI_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"}},
	},
	"openrouter": {
		{Field: "api_key", Env: []string{"OPENROUTER_API_KEY"}},
		{Field: "base_url", Env: []string{"OPENROUTER_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"}},
	},
	"canopywave": {
		{Field: "api_key", Env: []string{"CANOPYWAVE_API_KEY"}},
		{Field: "base_url", Env: []string{"CANOPYWAVE_BASE_URL", "OPENAI_BASE_URL", "OPENAI_API_BASE"}},
	},
	"ollama-local": {
		{Field: "base_url", Env: []string{"OLLAMA_BASE_URL"}},
		{Field: "api_key", Env: []string{"OLLAMA_API_KEY", "OPENAI_API_KEY"}},
	},
	"opencodego": {
		{Field: "api_key", Env: []string{"OPENCODEGO_API_KEY"}},
		{Field: "base_url", Env: []string{"OPENCODEGO_BASE_URL"}},
	},
	}
	for id, fbs := range registry.DeploymentEnvFallbacks() {
		if _, ok := base[id]; ok {
			continue
		}
		var converted []EnvFallbackV1
		for _, fb := range fbs {
			converted = append(converted, EnvFallbackV1{Field: fb.Field, Env: fb.Env})
		}
		base[id] = converted
	}
	return base
}()

// EnsureDeploymentEnvFallbacks fills missing env_fallbacks from the embedded seed.
// Published catalogs with env_fallbacks set are left unchanged.
func EnsureDeploymentEnvFallbacks(c *CatalogV1) {
	if c == nil || c.Deployments == nil {
		return
	}
	for id, dep := range c.Deployments {
		if len(dep.EnvFallbacks) > 0 {
			continue
		}
		if fb, ok := DefaultDeploymentEnvFallbacks[id]; ok {
			dep.EnvFallbacks = fb
			c.Deployments[id] = dep
		}
	}
}

// DiscoveryEnvKeysFromCatalog returns env var names needed for catalog discovery
// (API keys, base URLs) from deployment env_fallbacks in the compiled catalog.
func DiscoveryEnvKeysFromCatalog(compiled *CompiledCatalogV1) []string {
	if compiled == nil || compiled.Catalog == nil {
		return nil
	}
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	for _, dep := range compiled.Catalog.Deployments {
		for _, fb := range dep.EnvFallbacks {
			for _, env := range fb.Env {
				add(env)
			}
		}
	}
	return keys
}

// EnvVarsForDeployment returns env var names for a deployment ID from the seed catalog.
func EnvVarsForDeployment(deploymentID string) []string {
	fb, ok := DefaultDeploymentEnvFallbacks[deploymentID]
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, f := range fb {
		for _, env := range f.Env {
			if env != "" && !seen[env] {
				seen[env] = true
				out = append(out, env)
			}
		}
	}
	return out
}
