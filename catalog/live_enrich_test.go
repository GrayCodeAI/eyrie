package catalog_test

import (
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

func TestFetchLiveProviderCatalog_SkipsProvidersWithoutCredentials(t *testing.T) {
	cat, enrichment := catalog.FetchLiveProviderCatalog(map[string]string{})
	if len(cat.Providers) != 0 {
		t.Fatalf("expected no providers without creds, got %d", len(cat.Providers))
	}
	if len(enrichment) != 9 {
		t.Fatalf("expected skipped status for all 9 providers, got %d", len(enrichment))
	}
	for _, item := range enrichment {
		if !strings.HasPrefix(item.Error, "skipped") {
			t.Fatalf("expected skipped enrichment, got %+v", item)
		}
	}
}

func TestFetchLiveProviderCatalog_AttemptsAllRegisteredFetchers(t *testing.T) {
	env := map[string]string{}
	for _, spec := range registry.All() {
		if !spec.RequiresKey {
			continue
		}
		env[spec.CredentialEnv] = "test-key-should-fail-network-12345678"
	}
	_, enrichment := catalog.FetchLiveProviderCatalog(env)
	if len(enrichment) != 9 {
		t.Fatalf("expected enrichment for all 9 providers, got %d", len(enrichment))
	}
	attempted := 0
	for _, item := range enrichment {
		if strings.HasPrefix(item.Error, "skipped") {
			continue
		}
		attempted++
	}
	if attempted != 8 {
		t.Fatalf("expected 8 live fetch attempts, got %d", attempted)
	}
	seen := map[string]bool{}
	for _, item := range enrichment {
		seen[item.Provider] = true
	}
	for _, spec := range registry.All() {
		if !spec.RequiresKey {
			continue
		}
		if !seen[spec.LiveCatalogKey] {
			t.Errorf("missing live fetch attempt for %s (catalog key %q)", spec.ProviderID, spec.LiveCatalogKey)
		}
	}
}
