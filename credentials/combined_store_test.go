package credentials

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type countingStore struct {
	mu      sync.Mutex
	data    map[string]string
	gets    int
	deletes int
}

func (s *countingStore) Set(ctx context.Context, account, secret string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = map[string]string{}
	}
	s.data[account] = secret
	return nil
}

func (s *countingStore) Get(ctx context.Context, account string) (string, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gets++
	if s.data == nil {
		return "", ErrNotFound
	}
	v, ok := s.data[account]
	if !ok || v == "" {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *countingStore) Delete(ctx context.Context, account string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes++
	if s.data != nil {
		delete(s.data, account)
	}
	return nil
}

// --- CombinedStore edge-case tests (nil keychain, empty secrets) ---

func TestCombinedStore_NilKeychain(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestCombinedStore_GetUsesShortLivedCache(t *testing.T) {
	t.Parallel()
	backend := &countingStore{data: map[string]string{"account": "secret"}}
	cs := &CombinedStore{Keychain: backend}
	ctx := context.Background()

	got1, err := cs.Get(ctx, "account")
	if err != nil || got1 != "secret" {
		t.Fatalf("first Get = (%q, %v), want (secret, nil)", got1, err)
	}
	got2, err := cs.Get(ctx, "account")
	if err != nil || got2 != "secret" {
		t.Fatalf("second Get = (%q, %v), want (secret, nil)", got2, err)
	}
	if backend.gets != 1 {
		t.Fatalf("backend Get count = %d, want 1", backend.gets)
	}
}

func TestCombinedStore_SetInvalidatesCache(t *testing.T) {
	t.Parallel()
	backend := &countingStore{data: map[string]string{"account": "old"}}
	cs := &CombinedStore{Keychain: backend}
	ctx := context.Background()

	got, err := cs.Get(ctx, "account")
	if err != nil || got != "old" {
		t.Fatalf("initial Get = (%q, %v), want (old, nil)", got, err)
	}
	if err := cs.Set(ctx, "account", "new"); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	got, err = cs.Get(ctx, "account")
	if err != nil || got != "new" {
		t.Fatalf("post-Set Get = (%q, %v), want (new, nil)", got, err)
	}
	if backend.gets != 2 {
		t.Fatalf("backend Get count after invalidation = %d, want 2", backend.gets)
	}
}

func TestCombinedStore_DeleteInvalidatesCache(t *testing.T) {
	t.Parallel()
	backend := &countingStore{data: map[string]string{"account": "secret"}}
	cs := &CombinedStore{Keychain: backend}
	ctx := context.Background()

	got, err := cs.Get(ctx, "account")
	if err != nil || got != "secret" {
		t.Fatalf("initial Get = (%q, %v), want (secret, nil)", got, err)
	}
	if err := cs.Delete(ctx, "account"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	got, err = cs.Get(ctx, "account")
	if err != ErrNotFound || got != "" {
		t.Fatalf("post-Delete Get = (%q, %v), want (\"\", ErrNotFound)", got, err)
	}
	if backend.gets != 2 {
		t.Fatalf("backend Get count after delete invalidation = %d, want 2", backend.gets)
	}
}

// --- DefaultStore / SetDefaultStore tests ---

func TestSetDefaultStore_Override(t *testing.T) {
	// Not parallel: mutates global DefaultStore.
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
	t.Parallel()
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
	t.Parallel()
	keys := discoveryEnvKeys(context.Background())
	if len(keys) == 0 {
		t.Fatal("discoveryEnvKeys should return fallback keys")
	}
}

// --- APIKeysMap tests ---

func TestAPIKeysMap_NilStoreUsesDefault(t *testing.T) {
	// Not parallel: mutates global DefaultStore.
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
	t.Parallel()
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
