package config

import "testing"

func TestDiscoveryEnvFromOS_UsesCatalogEnvFallbacks(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("OPENROUTER_BASE_URL", "http://example.test")

	env := DiscoveryEnvFromOS()
	if env["OPENROUTER_API_KEY"] != "test-key" {
		t.Fatalf("expected OPENROUTER_API_KEY from catalog env_fallbacks, got %q", env["OPENROUTER_API_KEY"])
	}
	if env["OPENROUTER_BASE_URL"] != "http://example.test" {
		t.Fatalf("expected OPENROUTER_BASE_URL from catalog env_fallbacks")
	}
}

func TestDiscoveryEnvKeysFromProfilesLegacy(t *testing.T) {
	keys := discoveryEnvKeysFromProfiles()
	has := func(want string) bool {
		for _, k := range keys {
			if k == want {
				return true
			}
		}
		return false
	}
	if !has("ANTHROPIC_API_KEY") || !has("OPENAI_API_KEY") {
		t.Fatalf("legacy profile keys should include anthropic and openai, got %v", keys)
	}
}
