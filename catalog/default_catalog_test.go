package catalog

import (
	"testing"
)

func TestDefaultCatalogV1_ReturnsBootstrap(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	if c.SchemaVersion != CatalogV1SchemaVersion {
		t.Fatalf("schema_version = %q", c.SchemaVersion)
	}
	if c.Provenance == nil || c.Provenance.Source != bootstrapSource {
		t.Fatalf("provenance source = %v, want %q", c.Provenance, bootstrapSource)
	}
}

func TestDefaultCatalogV1_HasProviders(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	expected := []string{"anthropic", "openai", "google", "xai", "openrouter", "ollama"}
	for _, id := range expected {
		if c.Providers[id].ID == "" {
			t.Errorf("missing provider %q in default catalog", id)
		}
	}
}

func TestDefaultCatalogV1_HasDeployments(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	expected := []string{
		"anthropic-direct", "openai-direct", "gemini-direct",
		"grok-direct", "openrouter", "ollama-local",
	}
	for _, id := range expected {
		if c.Deployments[id].ID == "" {
			t.Errorf("missing deployment %q in default catalog", id)
		}
	}
}

func TestDefaultCatalogV1_HasAPIProtocols(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	expected := []string{"anthropic-messages", "openai-chat-completions", "gemini-generate-content"}
	for _, id := range expected {
		if c.APIProtocols[id].ID == "" {
			t.Errorf("missing api_protocol %q in default catalog", id)
		}
	}
}

func TestDefaultCatalogV1_NoModels(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	if len(c.Models) != 0 {
		t.Fatalf("bootstrap catalog should have no models, got %d", len(c.Models))
	}
	if len(c.Offerings) != 0 {
		t.Fatalf("bootstrap catalog should have no offerings, got %d", len(c.Offerings))
	}
}

func TestDefaultCatalogV1_DeploymentsReferenceProviders(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	for id, dep := range c.Deployments {
		if c.Providers[dep.ProviderID].ID == "" {
			t.Errorf("deployment %q references unknown provider %q", id, dep.ProviderID)
		}
		if c.APIProtocols[dep.APIProtocolID].ID == "" {
			t.Errorf("deployment %q references unknown api_protocol %q", id, dep.APIProtocolID)
		}
	}
}

func TestDefaultCatalogV1_Validates(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	if err := ValidateCatalogV1(&c); err != nil {
		t.Fatalf("default catalog should validate: %v", err)
	}
}

func TestDefaultCatalogV1_Compiles(t *testing.T) {
	t.Parallel()
	c := DefaultCatalogV1()
	compiled, err := CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("default catalog should compile: %v", err)
	}
	if compiled == nil {
		t.Fatal("compiled should not be nil")
	}
}

func TestIsBootstrapCatalog(t *testing.T) {
	t.Parallel()
	bootstrap := BootstrapCatalogV1()
	if !IsBootstrapCatalog(&bootstrap) {
		t.Error("BootstrapCatalogV1 should be identified as bootstrap")
	}

	legacy := testLegacyCatalogV1()
	if IsBootstrapCatalog(&legacy) {
		t.Error("legacy-sourced catalog should not be bootstrap")
	}

	if IsBootstrapCatalog(nil) {
		t.Error("nil should not be bootstrap")
	}
}

func TestBootstrapCatalogV1_HasEnvFallbacks(t *testing.T) {
	t.Parallel()
	c := BootstrapCatalogV1()
	anthDep, ok := c.Deployments["anthropic-direct"]
	if !ok {
		t.Fatal("missing anthropic-direct")
	}
	if len(anthDep.EnvFallbacks) == 0 {
		t.Error("bootstrap anthropic-direct should have env fallbacks")
	}
}

func TestBootstrapCatalogV1_HasCredentialProviders(t *testing.T) {
	t.Parallel()
	c := BootstrapCatalogV1()
	// Ollama is local, should not require key
	ollamaDep, ok := c.Deployments["ollama-local"]
	if !ok {
		t.Fatal("missing ollama-local")
	}
	if ollamaDep.Local != true {
		t.Error("ollama-local should be marked local")
	}

	// Anthropic requires key
	anthDep, ok := c.Deployments["anthropic-direct"]
	if !ok {
		t.Fatal("missing anthropic-direct")
	}
	hasAPIKey := false
	for _, fb := range anthDep.EnvFallbacks {
		if fb.Field == "api_key" {
			hasAPIKey = true
		}
	}
	if !hasAPIKey {
		t.Error("anthropic-direct should have api_key in env fallbacks")
	}
}
