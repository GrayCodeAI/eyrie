package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogV1FromLegacyCompiles(t *testing.T) {
	c := testLegacyCatalogV1()
	compiled, err := CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("CompileCatalogV1 failed: %v", err)
	}
	if compiled.ModelsByID["anthropic/claude-sonnet-4-6"].ID == "" {
		t.Fatal("expected canonical anthropic model")
	}
	offering, ok := compiled.OfferingForDeployment("anthropic/claude-sonnet-4-6", "anthropic-direct")
	if !ok {
		t.Fatal("expected anthropic direct offering")
	}
	if offering.NativeModelID != "claude-sonnet-4-6" {
		t.Fatalf("native model = %q, want claude-sonnet-4-6", offering.NativeModelID)
	}
	if canonical, ok := compiled.CanonicalModelForAliasOrID("claude-sonnet-4-6"); !ok || canonical != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("alias resolved to %q, %v", canonical, ok)
	}
}

func TestCatalogV1FromLegacyZAIDirectModels(t *testing.T) {
	legacy := testLegacyModelCatalog()
	legacy.Providers["zai_payg"] = []ModelCatalogEntry{{ID: "glm-5.1", DisplayName: "GLM-5.1"}}
	c := CatalogV1FromLegacy(legacy)
	compiled, err := CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("CompileCatalogV1 failed: %v", err)
	}
	if _, ok := compiled.OfferingForDeployment("zai_payg/glm-5.1", "zai_payg-direct"); !ok {
		t.Fatal("expected zai_payg-direct offering on zai_payg/glm-5.1")
	}
}

func TestCatalogV1FromLegacyCanopyWaveNamespacedModels(t *testing.T) {
	legacy := testLegacyModelCatalog()
	legacy.Providers["canopywave"] = append(
		legacy.Providers["canopywave"],
		ModelCatalogEntry{ID: "moonshotai/kimi-k2.6", DisplayName: "Kimi K2.6"},
	)
	c := CatalogV1FromLegacy(legacy)
	compiled, err := CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("CompileCatalogV1 failed: %v", err)
	}
	if _, ok := compiled.OfferingForDeployment("moonshotai/kimi-k2.6", "canopywave"); !ok {
		t.Fatal("expected canopywave offering on moonshotai/kimi-k2.6")
	}
}

func TestValidateCatalogV1RejectsBadReferences(t *testing.T) {
	c := testLegacyCatalogV1()
	c.Offerings = append(c.Offerings, ModelOfferingV1{
		ID:               "missing:model",
		CanonicalModelID: "anthropic/claude-sonnet-4-6",
		DeploymentID:     "missing",
		NativeModelID:    "model",
		Pricing:          PricingV1{Status: PricingUnknown},
	})
	if err := ValidateCatalogV1(&c); err == nil {
		t.Fatal("expected validation failure")
	}
}

func TestLoadCatalogV1UsesValidCacheBeforeRemote(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "catalog.json")
	c := testLegacyCatalogV1()
	c.SourceForTest("cache")
	if err := WriteCatalogV1Cache(cachePath, &c); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	compiled, err := LoadCatalogV1(context.Background(), LoadCatalogV1Options{
		CachePath: cachePath,
		RemoteURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("LoadCatalogV1 failed: %v", err)
	}
	if compiled.Catalog.Provenance == nil || compiled.Catalog.Provenance.Source != "cache" {
		t.Fatalf("expected cached catalog, got %#v", compiled.Catalog.Provenance)
	}
	if calls != 0 {
		t.Fatalf("remote called despite valid cache")
	}
}

func TestLoadCatalogV1RefreshRemoteOverridesValidCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "catalog.json")
	cached := testLegacyCatalogV1()
	cached.SourceForTest("cache")
	if err := WriteCatalogV1Cache(cachePath, &cached); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	remote := testLegacyCatalogV1()
	remote.SourceForTest("remote")
	data, err := json.Marshal(remote)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	compiled, err := LoadCatalogV1(context.Background(), LoadCatalogV1Options{
		CachePath:     cachePath,
		RemoteURL:     srv.URL,
		RefreshRemote: true,
	})
	if err != nil {
		t.Fatalf("LoadCatalogV1 failed: %v", err)
	}
	if compiled.Catalog.Provenance == nil || compiled.Catalog.Provenance.Source != "remote" {
		t.Fatalf("expected remote catalog, got %#v", compiled.Catalog.Provenance)
	}
	if calls != 1 {
		t.Fatalf("remote calls = %d, want 1", calls)
	}
}

func TestLoadCatalogV1RejectsInvalidRemote(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "missing.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"schema_version":"model-catalog/v1"}`))
	}))
	defer srv.Close()
	_, err := LoadCatalogV1(context.Background(), LoadCatalogV1Options{
		CachePath:     cachePath,
		RemoteURL:     srv.URL,
		RefreshRemote: true,
	})
	if err == nil {
		t.Fatal("expected error for invalid remote catalog")
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("invalid remote should not write cache, stat err=%v", err)
	}
}

func TestFetchRemoteCatalogV1StrictValidation(t *testing.T) {
	c := testLegacyCatalogV1()
	c.GeneratedAt = time.Now().UTC()
	c.StaleAfter = c.GeneratedAt.Add(time.Hour)
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	fetched, err := FetchRemoteCatalogV1(context.Background(), LoadCatalogV1Options{RemoteURL: srv.URL})
	if err != nil {
		t.Fatalf("FetchRemoteCatalogV1 failed: %v", err)
	}
	if fetched.SchemaVersion != CatalogV1SchemaVersion {
		t.Fatalf("schema version = %q", fetched.SchemaVersion)
	}
}

func (c *CatalogV1) SourceForTest(source string) {
	if c.Provenance == nil {
		c.Provenance = &CatalogProvenanceV1{}
	}
	c.Provenance.Source = source
}

func TestCapabilitySetFromLegacy_AnthropicFeatures(t *testing.T) {
	entry := ModelCatalogEntry{
		ID:            "claude-sonnet-4-6",
		ContextWindow: 1000000,
		MaxOutput:     128000,
		ServerTools: []string{
			"thinking:enabled",
			"thinking:adaptive",
			"effort",
			"effort:low,medium,high,xhigh,max",
			"structured_output",
			"code_execution",
			"citations",
			"pdf_input",
			"image_input",
			"tools",
		},
	}
	set := capabilitySetFromLegacy(entry)

	if set.ExplicitThinkingBudget != CapabilitySupported {
		t.Errorf("ExplicitThinkingBudget = %q, want supported", set.ExplicitThinkingBudget)
	}
	if set.AdaptiveThinking != CapabilitySupported {
		t.Errorf("AdaptiveThinking = %q, want supported", set.AdaptiveThinking)
	}
	if set.Effort != CapabilitySupported {
		t.Errorf("Effort = %q, want supported", set.Effort)
	}
	if set.StructuredOutput != CapabilitySupported {
		t.Errorf("StructuredOutput = %q, want supported", set.StructuredOutput)
	}
	if set.CodeExecution != CapabilitySupported {
		t.Errorf("CodeExecution = %q, want supported", set.CodeExecution)
	}
	if set.Citations != CapabilitySupported {
		t.Errorf("Citations = %q, want supported", set.Citations)
	}
	if set.PDFInput != CapabilitySupported {
		t.Errorf("PDFInput = %q, want supported", set.PDFInput)
	}
	if set.ImageInput != CapabilitySupported {
		t.Errorf("ImageInput = %q, want supported", set.ImageInput)
	}
	if set.FunctionCalling != CapabilitySupported {
		t.Errorf("FunctionCalling = %q, want supported", set.FunctionCalling)
	}
	if set.MaxInputTokens != 1000000 {
		t.Errorf("MaxInputTokens = %d, want 1000000", set.MaxInputTokens)
	}
	if set.MaxOutputTokens != 128000 {
		t.Errorf("MaxOutputTokens = %d, want 128000", set.MaxOutputTokens)
	}
	if len(set.ThinkingTypes) != 2 {
		t.Errorf("ThinkingTypes len = %d, want 2", len(set.ThinkingTypes))
	}
	if len(set.EffortLevels) != 5 {
		t.Errorf("EffortLevels len = %d, want 5", len(set.EffortLevels))
	}
}

func TestCapabilitySetFromLegacy_EmptyFeatures(t *testing.T) {
	entry := ModelCatalogEntry{ID: "test-model"}
	set := capabilitySetFromLegacy(entry)
	if set.ServerTools != nil {
		t.Errorf("expected nil ServerTools, got %v", set.ServerTools)
	}
	if set.ExplicitThinkingBudget != "" {
		t.Errorf("expected empty ExplicitThinkingBudget, got %q", set.ExplicitThinkingBudget)
	}
}
