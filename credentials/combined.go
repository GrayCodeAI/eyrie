package credentials

import (
	"context"
	"fmt"
	"strings"
)

// CombinedStore persists secrets in the OS secret store (macOS Keychain / Linux Secret Service).
type CombinedStore struct {
	Keychain Store
}

func NewCombinedStore() *CombinedStore {
	return &CombinedStore{
		Keychain: newPlatformKeyringStore(),
	}
}

func (c *CombinedStore) Set(ctx context.Context, account, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	if c.Keychain == nil {
		return ErrKeychainUnavailable()
	}
	if err := c.Keychain.Set(ctx, account, secret); err != nil {
		return fmt.Errorf("%w: %s", err, KeyringUnavailableHelp())
	}
	return nil
}

func (c *CombinedStore) Get(ctx context.Context, account string) (string, error) {
	if c.Keychain == nil {
		return "", ErrNotFound
	}
	return c.Keychain.Get(ctx, account)
}

func (c *CombinedStore) Delete(ctx context.Context, account string) error {
	if c.Keychain == nil {
		return nil
	}
	return c.Keychain.Delete(ctx, account)
}
