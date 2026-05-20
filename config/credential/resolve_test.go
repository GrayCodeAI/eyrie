package credential

import (
	"context"
	"testing"
)

func TestValidateKeyFormat(t *testing.T) {
	if err := ValidateKeyFormat(""); err == nil {
		t.Fatal("expected error for empty")
	}
	if err := ValidateKeyFormat("your-api-key"); err == nil {
		t.Fatal("expected placeholder error")
	}
	if err := ValidateKeyFormat("sk-ant-api03-test-key-1234567890"); err != nil {
		t.Fatalf("valid key: %v", err)
	}
}

func TestResolveCredential_ListsAllProviders(t *testing.T) {
	res := ResolveCredential(context.Background(), "sk-ant-api03-test-key-1234567890")
	if !res.FormatOK {
		t.Fatalf("format should be ok: %s", res.FormatError)
	}
	if len(res.Providers) != 7 {
		t.Fatalf("expected 7 catalog providers, got %d", len(res.Providers))
	}
	inferred := 0
	for _, p := range res.Providers {
		if p.Inferred {
			inferred++
		}
	}
	if inferred == 0 {
		t.Fatal("expected at least one inferred provider")
	}
	if !res.Providers[0].Inferred {
		t.Fatal("inferred providers should be ranked first")
	}
	if res.Providers[0].ProviderID != "anthropic" {
		t.Fatalf("top inferred = %q, want anthropic", res.Providers[0].ProviderID)
	}
}

func TestResolveCredential_InvalidFormat(t *testing.T) {
	res := ResolveCredential(context.Background(), "short")
	if res.FormatOK {
		t.Fatal("expected format error")
	}
}

func TestListCredentialProviders_Count(t *testing.T) {
	if n := len(ListCredentialProviders()); n != 8 {
		t.Fatalf("expected 8 providers, got %d", n)
	}
}
