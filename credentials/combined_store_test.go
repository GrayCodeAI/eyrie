package credentials

import (
	"context"
	"errors"
	"testing"
)

// --- CombinedStore edge-case tests (nil keychain, empty secrets) ---

func TestCombinedStore_NilKeychain(t *testing.T) {
	cs := &CombinedStore{Keychain: nil}
	ctx := context.Background()

	t.Run("Set_returns_unavailable", func(t *testing.T) {
		err := cs.Set(ctx, "account", "secret")
		if !errors.Is(err, ErrKeychainUnavailable) {
			t.Errorf("Set(nil keychain) = %v, want ErrKeychainUnavailable", err)
		}
	})

	t.Run("Get_returns_not_found", func(t *testing.T) {
		_, err := cs.Get(ctx, "account")
		if err != ErrNotFound {
			t.Errorf("Get(nil keychain) = %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete_returns_nil", func(t *testing.T) {
		if err := cs.Delete(ctx, "account"); err != nil {
			t.Errorf("Delete(nil keychain) = %v, want nil", err)
		}
	})
}

func TestCombinedStore_EmptySecretIsNoop(t *testing.T) {
	ms := &MapStore{}
	cs := &CombinedStore{Keychain: ms}
	ctx := context.Background()

	tests := []struct {
		name   string
		secret string
	}{
		{name: "empty_string", secret: ""},
		{name: "whitespace_only", secret: "   "},
		{name: "tab_only", secret: "\t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cs.Set(ctx, "account", tt.secret)
			if err != nil {
				t.Fatalf("Set(%q) error: %v", tt.secret, err)
			}
			// MapStore should not have the key since CombinedStore skips empty secrets.
			if len(ms.Data) != 0 {
				t.Errorf("MapStore should be empty after Set with %q, got %v", tt.secret, ms.Data)
			}
		})
	}
}

func TestCombinedStore_FullCycle(t *testing.T) {
	ms := &MapStore{}
	cs := &CombinedStore{Keychain: ms}
	ctx := context.Background()

	account := "test_account"
	secret := "test_secret_value"

	// Get before set should return not found.
	_, err := cs.Get(ctx, account)
	if err != ErrNotFound {
		t.Fatalf("Get before Set: err = %v, want ErrNotFound", err)
	}

	// Set.
	if err := cs.Set(ctx, account, secret); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	// Get should return the value.
	got, err := cs.Get(ctx, account)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != secret {
		t.Errorf("Get = %q, want %q", got, secret)
	}

	// Delete.
	if err := cs.Delete(ctx, account); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	// Get after delete should return not found.
	_, err = cs.Get(ctx, account)
	if err != ErrNotFound {
		t.Fatalf("Get after Delete: err = %v, want ErrNotFound", err)
	}
}

// --- DefaultStore / SetDefaultStore tests ---

func TestSetDefaultStore_Override(t *testing.T) {
	_ = DefaultStore() // ensure initialized
	t.Cleanup(func() { SetDefaultStore(nil) })

	ms := &MapStore{}
	SetDefaultStore(ms)

	got := DefaultStore()
	if got != ms {
		t.Fatal("DefaultStore did not return the injected MapStore")
	}

	// Set to nil to force re-creation.
	SetDefaultStore(nil)
	got2 := DefaultStore()
	if got2 == nil {
		t.Fatal("DefaultStore() should auto-create a CombinedStore when nil")
	}
}

// --- discoveryEnvKeys tests ---

func TestDiscoveryEnvKeys_ReturnsNonEmpty(t *testing.T) {
	ctx := context.Background()
	keys := discoveryEnvKeys(ctx)

	if len(keys) == 0 {
		t.Fatal("discoveryEnvKeys should return at least the fallback keys")
	}

	// All keys should be non-empty uppercase strings.
	for _, k := range keys {
		if k == "" {
			t.Error("discoveryEnvKeys returned an empty key")
		}
	}
	// These should always be present in either the catalog or fallback.
	mustHave := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY"}
	keySet := map[string]bool{}
	for _, k := range keys {
		keySet[k] = true
	}
	for _, want := range mustHave {
		if !keySet[want] {
			t.Errorf("discoveryEnvKeys missing expected key %q", want)
		}
	}
}

func TestDiscoveryEnvKeys_NilContext(t *testing.T) {
	keys := discoveryEnvKeys(context.TODO())
	if len(keys) == 0 {
		t.Fatal("discoveryEnvKeys should return fallback keys")
	}
}

// --- APIKeysMap tests ---

func TestAPIKeysMap_NilStoreUsesDefault(t *testing.T) {
	ms := &MapStore{}
	SetDefaultStore(ms)
	t.Cleanup(func() { SetDefaultStore(nil) })

	ctx := context.Background()
	_ = ms.Set(ctx, AccountForEnv("ANTHROPIC_API_KEY"), "sk-test-123")

	m := APIKeysMap(ctx, nil)
	if m["ANTHROPIC_API_KEY"] != "sk-test-123" {
		t.Errorf("APIKeysMap with nil store: got %v, want ANTHROPIC_API_KEY=sk-test-123", m)
	}
}

func TestAPIKeysMap_WithStore(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()
	_ = ms.Set(ctx, AccountForEnv("OPENAI_API_KEY"), "sk-oai")

	m := APIKeysMap(ctx, ms)
	if m["OPENAI_API_KEY"] != "sk-oai" {
		t.Errorf("APIKeysMap: got %v, want OPENAI_API_KEY=sk-oai", m)
	}
	// Keys not stored should be absent.
	if _, ok := m["GEMINI_API_KEY"]; ok {
		t.Error("APIKeysMap should not include keys that are not stored")
	}
}
