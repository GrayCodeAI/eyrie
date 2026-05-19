package catalog

import "testing"

func TestPrimaryAPIKeyEnvForProvider(t *testing.T) {
	bootstrap := BootstrapCatalogV1()
	compiled, err := CompileCatalogV1(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "anthropic"); got != "ANTHROPIC_API_KEY" {
		t.Fatalf("anthropic env = %q", got)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "openrouter"); got != "OPENROUTER_API_KEY" {
		t.Fatalf("openrouter env = %q", got)
	}
}

func TestCredentialStatusForProvider_OllamaLocal(t *testing.T) {
	bootstrap := BootstrapCatalogV1()
	compiled, err := CompileCatalogV1(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	// ollama-local has base_url env; api_key is optional — still has api_key fallbacks in seed
	status := CredentialStatusForProvider(compiled, "ollama")
	if status != "empty" && status != "set" && status != "local" {
		t.Fatalf("unexpected status %q", status)
	}
}

func TestProviderIDsFromCompiled_Bootstrap(t *testing.T) {
	bootstrap := BootstrapCatalogV1()
	compiled, err := CompileCatalogV1(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	ids := ProviderIDsFromCompiled(compiled)
	if len(ids) < 5 {
		t.Fatalf("expected several providers, got %v", ids)
	}
}
