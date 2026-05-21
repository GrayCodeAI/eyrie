package catalog

import "testing"

func TestDiscoveryEnvKeysFromCatalog(t *testing.T) {
	embedded := DefaultCatalogV1()
	compiled, err := CompileCatalogV1(&embedded)
	if err != nil {
		t.Fatalf("CompileCatalogV1: %v", err)
	}
	keys := DiscoveryEnvKeysFromCatalog(compiled)
	has := func(want string) bool {
		for _, k := range keys {
			if k == want {
				return true
			}
		}
		return false
	}
	if !has("OPENROUTER_API_KEY") || !has("ANTHROPIC_API_KEY") {
		t.Fatalf("expected catalog env keys from deployments, got %v", keys)
	}
}

func TestEnvVarsForDeployment_OpenRouter(t *testing.T) {
	envs := EnvVarsForDeployment("openrouter")
	if len(envs) == 0 {
		t.Fatal("expected env vars for openrouter deployment")
	}
}

func TestEnsureDeploymentEnvFallbacks_FillsMissing(t *testing.T) {
	c := &CatalogV1{
		Deployments: map[string]DeploymentV1{
			"openrouter": {ID: "openrouter"},
		},
	}
	EnsureDeploymentEnvFallbacks(c)
	if len(c.Deployments["openrouter"].EnvFallbacks) == 0 {
		t.Fatal("expected seeded env_fallbacks for openrouter")
	}
}

func TestEnsureDeploymentEnvFallbacks_PreservesPublishedEnvFallbacks(t *testing.T) {
	c := &CatalogV1{
		Deployments: map[string]DeploymentV1{
			"custom": {
				ID: "custom",
				EnvFallbacks: []EnvFallbackV1{
					{Field: "api_key", Env: []string{"CUSTOM_API_KEY"}},
				},
			},
		},
	}
	EnsureDeploymentEnvFallbacks(c)
	if c.Deployments["custom"].EnvFallbacks[0].Env[0] != "CUSTOM_API_KEY" {
		t.Fatal("published env_fallbacks should not be overwritten")
	}
}
