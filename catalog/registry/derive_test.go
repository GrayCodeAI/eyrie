package registry

import "testing"

func TestSpecByProviderID_AcceptsCatalogAliases(t *testing.T) {
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
	if got := DisplayName("google"); got != "Google Gemini" {
		t.Fatalf("DisplayName(google) = %q", got)
	}
}
