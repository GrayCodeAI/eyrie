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
	if len(enrichment) != len(registry.All()) {
		t.Fatalf("expected skipped status for all %d providers, got %d", len(registry.All()), len(enrichment))
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
	if len(enrichment) != len(registry.All()) {
		t.Fatalf("expected enrichment for all %d providers, got %d", len(registry.All()), len(enrichment))
	}
	attempted := 0
	for _, item := range enrichment {
		if strings.HasPrefix(item.Error, "skipped") {
			continue
		}
		attempted++
	}
	expectedAttempts := 0
	for _, spec := range registry.All() {
		if spec.RequiresKey {
			expectedAttempts++
		}
	}
	if attempted != expectedAttempts {
		t.Fatalf("expected %d live fetch attempts, got %d", expectedAttempts, attempted)
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
