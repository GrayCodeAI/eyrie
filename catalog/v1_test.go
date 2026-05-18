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
	c := DefaultCatalogV1()
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

func TestValidateCatalogV1RejectsBadReferences(t *testing.T) {
	c := DefaultCatalogV1()
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
	c := DefaultCatalogV1()
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
	cached := DefaultCatalogV1()
	cached.SourceForTest("cache")
	if err := WriteCatalogV1Cache(cachePath, &cached); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	remote := DefaultCatalogV1()
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

func TestLoadCatalogV1RejectsInvalidRemoteAndKeepsEmbedded(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "missing.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"schema_version":"model-catalog/v1"}`))
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
	if compiled.Catalog.Provenance == nil || compiled.Catalog.Provenance.Source != "embedded" {
		t.Fatalf("expected embedded fallback, got %#v", compiled.Catalog.Provenance)
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("invalid remote should not write cache, stat err=%v", err)
	}
}

func TestFetchRemoteCatalogV1StrictValidation(t *testing.T) {
	c := DefaultCatalogV1()
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
