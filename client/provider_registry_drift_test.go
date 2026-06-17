//nolint:errcheck
package client

import (
	"sort"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// TestProviderRegistry_NoDriftFromCatalog guards against future drift between
// the runtime provider maps (CoreProviders, OpenAICompatibleProviders) and the
// authoritative spec list in catalog/registry. If a new ProviderSpec is added
// to catalog/registry/providers.go without a corresponding runtime entry, or
// vice versa, the test fails and the PR is blocked.
//
// This regression test was introduced for H11 (provider-registry drift):
// minimax_token_plan and minimax_payg were present in catalog/registry but
// missing from OpenAICompatibleProviders, causing NewOpenAICompatibleClient
// to mis-route callers to NewOpenAIClient with BaseURL="" (4xx).
func TestProviderRegistry_NoDriftFromCatalog(t *testing.T) {
	specs := registry.All()
	specCount := len(specs)
	runtimeCount := len(CoreProviders) + len(OpenAICompatibleProviders)
	if runtimeCount != specCount {
		var missing []string
		seen := map[string]bool{}
		for k := range CoreProviders {
			seen[k] = true
		}
		for k := range OpenAICompatibleProviders {
			seen[k] = true
		}
		for _, s := range specs {
			if !seen[s.ProviderID] {
				missing = append(missing, s.ProviderID)
			}
		}
		sort.Strings(missing)
		t.Fatalf(
			"provider-registry drift: runtime maps have %d entries (CoreProviders=%d, OpenAICompatibleProviders=%d) "+
				"but catalog/registry has %d. Missing from runtime maps: %v",
			runtimeCount, len(CoreProviders), len(OpenAICompatibleProviders), specCount, missing,
		)
	}

	// Every spec ProviderID must appear in one of the runtime maps.
	for _, s := range specs {
		if _, ok := CoreProviders[s.ProviderID]; ok {
			continue
		}
		if _, ok := OpenAICompatibleProviders[s.ProviderID]; ok {
			continue
		}
		t.Fatalf("provider %q is registered in catalog/registry but missing from both CoreProviders and OpenAICompatibleProviders", s.ProviderID)
	}
}
