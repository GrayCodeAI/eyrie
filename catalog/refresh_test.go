package catalog

import (
	"strings"
	"testing"
	"time"
)

func TestRefreshResult_DiscoverReport(t *testing.T) {
	t.Parallel()
	def := testLegacyCatalogV1()
	compiled, err := CompileCatalogV1(&def)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	r := &RefreshResult{
		Compiled:        compiled,
		CachePath:       "/tmp/model_catalog.json",
		Source:          "remote+providers",
		RemoteURL:       "https://example.com/catalog.json",
		RemoteRefreshed: true,
		LiveProviders: []LiveProviderEnrichment{
			{Provider: "openrouter", ModelCount: 42},
			{Provider: "canopywave", Error: "401 unauthorized"},
		},
		StaleAfter: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	out := r.DiscoverReport()
	for _, want := range []string{
		"Remote catalog: refreshed",
		"https://example.com/catalog.json",
		"openrouter: 42 models merged",
		"canopywave: failed",
		"stale_after: 2026-06-01T00:00:00Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("DiscoverReport missing %q:\n%s", want, out)
		}
	}
}

func TestRefreshResult_DiscoverReport_NoLiveAPIs(t *testing.T) {
	t.Parallel()
	def := testLegacyCatalogV1()
	compiled, err := CompileCatalogV1(&def)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	out := (&RefreshResult{Compiled: compiled, CachePath: "/tmp/c.json"}).DiscoverReport()
	if !strings.Contains(out, "Live APIs: none") {
		t.Fatalf("expected no-live-apis message:\n%s", out)
	}
}
