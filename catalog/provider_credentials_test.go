package catalog

import "testing"

func TestPrimaryAPIKeyEnvForProvider(t *testing.T) {
	t.Parallel()
	bootstrap := BootstrapCatalog()
	compiled, err := CompileCatalog(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "anthropic"); got != "ANTHROPIC_API_KEY" {
		t.Fatalf("anthropic env = %q", got)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "openrouter"); got != "OPENROUTER_API_KEY" {
		t.Fatalf("openrouter env = %q", got)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "canopywave"); got != "CANOPYWAVE_API_KEY" {
		t.Fatalf("canopywave env = %q", got)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "zai_payg"); got != "ZAI_API_KEY" {
		t.Fatalf("zai_payg env = %q", got)
	}
	if got := PrimaryAPIKeyEnvForProvider(compiled, "zai_coding"); got != "ZAI_CODING_API_KEY" {
		t.Fatalf("zai_coding env = %q", got)
	}
}

func TestCredentialStatusForProvider_OllamaLocal(t *testing.T) {
	t.Parallel()
	bootstrap := BootstrapCatalog()
	compiled, err := CompileCatalog(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	// ollama-local has base_url env; api_key is optional — still has api_key fallbacks in seed
	status := CredentialStatusForProvider(compiled, "ollama")
	if status != "local" && status != "required" {
		t.Fatalf("unexpected status %q", status)
	}
}

func TestProviderIDsFromCompiled_Bootstrap(t *testing.T) {
	t.Parallel()
	bootstrap := BootstrapCatalog()
	compiled, err := CompileCatalog(&bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	ids := ProviderIDsFromCompiled(compiled)
	if len(ids) < 5 {
		t.Fatalf("expected several providers, got %v", ids)
	}
}
