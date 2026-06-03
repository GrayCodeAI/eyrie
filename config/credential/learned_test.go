package credential

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractLearningPrefix(t *testing.T) {
	got := extractLearningPrefix("AQ.abc123-secret")
	if len(got) < 4 || got[:3] != "AQ." {
		t.Fatalf("prefix = %q", got)
	}
	if extractLearningPrefix("ab") != "" {
		t.Fatal("expected empty for short secret")
	}
}

func TestRecordLearnedCredential_BoostsInference(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.MkdirAll(filepath.Join(home, ".hawk"), 0o700)
	InvalidateLearnedPrefixCache()

	RecordLearnedCredential("gemini", "AQ.test-key-1234567890")
	InvalidateLearnedPrefixCache()

	boost := learnedPrefixBoost("AQ.test-key-9999999999")
	if boost["gemini"] == 0 {
		t.Fatal("expected learned boost for gemini after record")
	}

	ctx := ContextWithoutProbeDisambiguation(t.Context())
	res := ResolveCredential(ctx, "AQ.test-key-9999999999")
	if !res.FormatOK {
		t.Fatalf("resolve failed: %s", res.FormatError)
	}
	if res.Providers[0].ProviderID != "gemini" || !res.Providers[0].Inferred {
		t.Fatalf("expected gemini inferred first, got %#v", res.Providers[0])
	}
}

func TestResolveCredential_GeminiAQPrefix(t *testing.T) {
	ctx := ContextWithoutProbeDisambiguation(t.Context())
	res := ResolveCredential(ctx, "AQ.test-gemini-key-1234567890")
	if !res.FormatOK {
		t.Fatalf("format: %s", res.FormatError)
	}
	if res.Providers[0].ProviderID != "gemini" || !res.Providers[0].Inferred {
		t.Fatalf("top = %#v, want inferred gemini", res.Providers[0])
	}
}