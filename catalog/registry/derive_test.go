package registry

import "testing"

func TestSpecByProviderID_AcceptsCatalogAliases(t *testing.T) {
	t.Parallel()
	if _, ok := SpecByProviderID("google"); !ok {
		t.Fatal("expected google to resolve to gemini spec")
	}
	if _, ok := SpecByProviderID("xai"); !ok {
		t.Fatal("expected xai to resolve to grok spec")
	}
	if spec, ok := SpecByProviderID("gemini"); !ok || spec.ProviderID != "gemini" {
		t.Fatalf("gemini spec = %+v ok=%v", spec, ok)
	}
}

func TestDisplayName_CatalogAlias(t *testing.T) {
	t.Parallel()
	if got := DisplayName("google"); got != "Gemini API" {
		t.Fatalf("DisplayName(google) = %q", got)
	}
}

func TestScopedProviderEnvExcludesUnrelatedCredentialsAndCanonicalizesAlias(t *testing.T) {
	t.Parallel()
	spec := ProviderSpec{
		ProviderID: "example", CredentialEnv: "EXAMPLE_API_KEY",
		CredentialEnvFallbacks: []string{"EXAMPLE_FALLBACK_KEY"},
		CredentialAliases:      []string{"EXAMPLE_ALIAS_KEY"},
		BaseURLEnv:             []string{"EXAMPLE_BASE_URL"},
	}
	got := ScopedProviderEnv(spec, map[string]string{
		"EXAMPLE_ALIAS_KEY": "alias-secret",
		"EXAMPLE_BASE_URL":  "https://example.test/v1",
		"OPENAI_API_KEY":    "must-not-leak",
		"AWS_ACCESS_KEY_ID": "must-not-leak",
	})
	if got["EXAMPLE_API_KEY"] != "alias-secret" || got["EXAMPLE_ALIAS_KEY"] != "alias-secret" {
		t.Fatalf("provider alias was not scoped and canonicalized: %#v", got)
	}
	if got["EXAMPLE_BASE_URL"] != "https://example.test/v1" {
		t.Fatalf("provider routing metadata missing: %#v", got)
	}
	for _, key := range []string{"OPENAI_API_KEY", "AWS_ACCESS_KEY_ID"} {
		if got[key] != "" {
			t.Fatalf("unrelated credential %s leaked into provider scope", key)
		}
	}
}
