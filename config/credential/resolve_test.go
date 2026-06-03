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

func TestValidateKeyFormat_NoPrefixRequired(t *testing.T) {
	// Gateway is chosen in UI; secrets need not match vendor prefix patterns.
	keys := []string{
		"0123456789abcdef",                    // no sk- prefix
		"mimo-secret-token-abcdef01",          // xiaomi without tp-
		"random_openrouter_key_1234567890",    // no sk-or-
		"cw_custom_canopywave_key_1234567890", // no cw_ prefix required
	}
	for _, key := range keys {
		if err := ValidateKeyFormat(key); err != nil {
			t.Fatalf("ValidateKeyFormat(%q): %v", key, err)
		}
		if err := ValidateCredentialSecret("XIAOMI_MIMO_PAYG_API_KEY", key); err != nil {
			t.Fatalf("ValidateCredentialSecret xiaomi: %v", err)
		}
	}
}

func TestResolveCredential_ListsAllProviders(t *testing.T) {
	res := ResolveCredential(context.Background(), "sk-ant-api03-test-key-1234567890")
	if !res.FormatOK {
		t.Fatalf("format should be ok: %s", res.FormatError)
	}
	if len(res.Providers) != 11 {
		t.Fatalf("expected 11 key-required providers, got %d", len(res.Providers))
	}
	for _, p := range res.Providers {
		if p.Inferred {
			t.Fatalf("provider %q should not be inferred without gateway selection", p.ProviderID)
		}
	}
	if res.Providers[0].ProviderID != "anthropic" {
		t.Fatalf("first provider = %q, want anthropic (registry order)", res.Providers[0].ProviderID)
	}
}

func TestResolveCredential_InvalidFormat(t *testing.T) {
	res := ResolveCredential(context.Background(), "short")
	if res.FormatOK {
		t.Fatal("expected format error")
	}
}

func TestListCredentialProviders_Count(t *testing.T) {
	if n := len(ListCredentialProviders()); n != 12 {
		t.Fatalf("expected 12 providers, got %d", n)
	}
}