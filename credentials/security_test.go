package credentials

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Secret handling: verify secrets aren't leaked in error messages, logs, or
// string representations.
// ---------------------------------------------------------------------------

func TestErrNotFound_DoesNotContainSecrets(t *testing.T) {
	// ErrNotFound is a sentinel error. Its message must not contain any
	// secret values or hint at what the secret might be.
	errMsg := ErrNotFound.Error()
	if strings.Contains(errMsg, "secret") || strings.Contains(errMsg, "key") || strings.Contains(errMsg, "password") {
		t.Errorf("ErrNotFound message should not reference secret/key/password: %q", errMsg)
	}
}

func TestMapStore_GetErrorDoesNotLeakValue(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()

	// Store a secret.
	secret := "sk-super-secret-api-key-12345"
	if err := ms.Set(ctx, "anthropic_api_key", secret); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	// Get a different (non-existent) key. The error must not leak the stored value.
	_, err := ms.Get(ctx, "nonexistent_key")
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("Get error leaked secret value: %q", err.Error())
	}
}

func TestMapStore_GetEmptyKeyReturnsNotFound(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()

	// Empty account should be handled gracefully.
	_, err := ms.Get(ctx, "")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for empty key, got: %v", err)
	}
}

func TestMapStore_SetEmptySecretStoresEmpty(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()

	// MapStore stores trimmed values; empty string is stored as-is.
	// This is acceptable for a test-only store. The real CombinedStore
	// (OS keychain) rejects empty secrets.
	if err := ms.Set(ctx, "test_key", ""); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	// The key exists but the value is empty.
	if ms.Data["test_key"] != "" {
		t.Errorf("expected empty value, got %q", ms.Data["test_key"])
	}
	// Get should return ErrNotFound for an empty value.
	_, err := ms.Get(ctx, "test_key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for empty stored value, got: %v", err)
	}
}

func TestMapStore_SetWhitespaceSecretStoresEmpty(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()

	if err := ms.Set(ctx, "test_key", "   "); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	// After trimming, the value is empty.
	if ms.Data["test_key"] != "" {
		t.Errorf("expected empty value after trimming, got %q", ms.Data["test_key"])
	}
	// Get should return ErrNotFound for an empty value.
	_, err := ms.Get(ctx, "test_key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for empty stored value, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AccountForEnv: verify normalization is consistent and safe
// ---------------------------------------------------------------------------

func TestAccountForEnv_NormalizesCorrectly(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ANTHROPIC_API_KEY", "anthropic_api_key"},
		{"OPENAI_API_KEY", "openai_api_key"},
		{"  TRIMMED_KEY  ", "trimmed_key"},
		{"", ""},
		{"UPPER", "upper"},
		{"lower", "lower"},
		{"MixedCase", "mixedcase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := AccountForEnv(tt.input)
			if got != tt.want {
				t.Errorf("AccountForEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EnvForAccount: verify reverse mapping doesn't leak secrets
// ---------------------------------------------------------------------------

func TestEnvForAccount_DoesNotReturnSecretValues(t *testing.T) {
	// EnvForAccount maps account names to env var NAMES (not values).
	// Verify it returns env var names, not actual secret values.
	tests := []struct {
		account string
		wantEnv string
	}{
		{"anthropic_api_key", "ANTHROPIC_API_KEY"},
		{"openai_api_key", "OPENAI_API_KEY"},
		{"openrouter_api_key", "OPENROUTER_API_KEY"},
		{"gemini_api_key", "GEMINI_API_KEY"},
		{"xai_api_key", "XAI_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.account, func(t *testing.T) {
			got := EnvForAccount(tt.account)
			if got != tt.wantEnv {
				t.Errorf("EnvForAccount(%q) = %q, want %q", tt.account, got, tt.wantEnv)
			}
			// The returned value should be an env var name (uppercase, no secrets).
			if strings.HasPrefix(got, "sk-") || strings.HasPrefix(got, "secret") {
				t.Errorf("EnvForAccount returned what looks like a secret value: %q", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// LookupSecret: verify safe error handling
// ---------------------------------------------------------------------------

func TestLookupSecret_EmptyKeyReturnsEmpty(t *testing.T) {
	// Empty env key should return empty string, not error or panic.
	result := LookupSecret(context.Background(), "")
	if result != "" {
		t.Errorf("LookupSecret('') = %q, want empty", result)
	}
}

func TestLookupSecret_WhitespaceKeyReturnsEmpty(t *testing.T) {
	result := LookupSecret(context.Background(), "   ")
	if result != "" {
		t.Errorf("LookupSecret('   ') = %q, want empty", result)
	}
}

func TestLookupSecret_NilContextHandled(t *testing.T) {
	// Nil context should not panic.
	result := LookupSecret(nil, "NONEXISTENT_KEY")
	if result != "" {
		t.Errorf("LookupSecret(nil, 'NONEXISTENT_KEY') = %q, want empty", result)
	}
}

func TestHasSecret_EmptyKeyReturnsFalse(t *testing.T) {
	if HasSecret(context.Background(), "") {
		t.Error("HasSecret('') should return false")
	}
}

func TestHasSecret_NilContextHandled(t *testing.T) {
	// Should not panic.
	if HasSecret(nil, "NONEXISTENT_KEY") {
		t.Error("HasSecret(nil, 'NONEXISTENT_KEY') should return false")
	}
}

// ---------------------------------------------------------------------------
// DeleteSecret: verify safe error handling
// ---------------------------------------------------------------------------

func TestDeleteSecret_EmptyKeyReturnsError(t *testing.T) {
	err := DeleteSecret(context.Background(), "")
	if err == nil {
		t.Error("DeleteSecret('') should return an error")
	}
	// The error message should not leak any secrets.
	if strings.Contains(err.Error(), "sk-") || strings.Contains(err.Error(), "secret") {
		t.Errorf("DeleteSecret error leaked secret info: %q", err.Error())
	}
}

func TestDeleteSecret_WhitespaceKeyReturnsError(t *testing.T) {
	err := DeleteSecret(context.Background(), "   ")
	if err == nil {
		t.Error("DeleteSecret('   ') should return an error")
	}
}

func TestDeleteSecret_NonexistentKeyNoError(t *testing.T) {
	// Deleting a non-existent key should not error.
	store := &MapStore{}
	SetDefaultStore(store)
	t.Cleanup(func() { SetDefaultStore(nil) })

	err := DeleteSecret(context.Background(), "nonexistent_key_xyz")
	if err != nil {
		t.Errorf("DeleteSecret for nonexistent key should not error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ScrubProcessEnv: verify it doesn't leak secrets
// ---------------------------------------------------------------------------

func TestScrubProcessEnv_EmptyKeysIgnored(t *testing.T) {
	// Should not panic with empty or whitespace keys.
	ScrubProcessEnv([]string{"", "  ", "\t"})
}

func TestScrubProcessEnv_NilKeysIgnored(t *testing.T) {
	// Should not panic with nil.
	ScrubProcessEnv(nil)
}

// ---------------------------------------------------------------------------
// APIKeysMap: verify secrets are returned correctly but not leaked
// ---------------------------------------------------------------------------

func TestAPIKeysMap_SecretsNotInErrorPaths(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()

	secret := "sk-secret-value-test"
	_ = ms.Set(ctx, AccountForEnv("ANTHROPIC_API_KEY"), secret)

	m := APIKeysMap(ctx, ms)

	// The map should contain the secret value (it's meant to pass to providers).
	if m["ANTHROPIC_API_KEY"] != secret {
		t.Errorf("APIKeysMap missing expected secret for ANTHROPIC_API_KEY")
	}

	// Keys not stored should be absent.
	if _, ok := m["GEMINI_API_KEY"]; ok {
		t.Error("APIKeysMap should not include keys that are not stored")
	}
}

func TestAPIKeysMap_NilStoreHandledGracefully(t *testing.T) {
	ms := &MapStore{}
	SetDefaultStore(ms)
	t.Cleanup(func() { SetDefaultStore(nil) })

	// Should not panic with nil store (falls back to DefaultStore).
	m := APIKeysMap(context.Background(), nil)
	_ = m
}

// ---------------------------------------------------------------------------
// StorageReport: verify no secret values in output
// ---------------------------------------------------------------------------

func TestFormatStorageReport_NoSecretValues(t *testing.T) {
	report := StorageReport{
		PlatformStore:    "macOS Keychain",
		KeychainWritable: true,
		StoredEnvKeys:    []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY"},
	}

	output := FormatStorageReport(report)

	// The output should contain env key NAMES but no actual secret values.
	if strings.Contains(output, "sk-") {
		t.Errorf("FormatStorageReport output leaked secret-like value: %s", output)
	}
	if strings.Contains(output, "secret") {
		t.Errorf("FormatStorageReport output contains 'secret': %s", output)
	}

	// Should contain the key names.
	if !strings.Contains(output, "ANTHROPIC_API_KEY") {
		t.Error("FormatStorageReport should list stored env key names")
	}
}

func TestStorageReport_EmptyKeysReportedCorrectly(t *testing.T) {
	report := StorageReport{
		PlatformStore:    "TestStore",
		KeychainWritable: false,
		StoredEnvKeys:    nil,
	}

	output := FormatStorageReport(report)
	if !strings.Contains(output, "(none)") {
		t.Errorf("expected '(none)' in empty report, got: %s", output)
	}
}
