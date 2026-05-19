package credentials

import (
	"context"
	"os"
	"strings"
)

// CombinedStore writes to keychain and optional env file; reads keychain first then env.
type CombinedStore struct {
	Keychain Store
	Env      Store
}

func NewCombinedStore() *CombinedStore {
	return &CombinedStore{
		Keychain: newPlatformKeyringStore(),
		Env:      &EnvFileStore{},
	}
}

func (c *CombinedStore) Set(ctx context.Context, account, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	if c.Keychain != nil {
		if err := c.Keychain.Set(ctx, account, secret); err == nil {
			if envFileSyncEnabled() {
				_ = c.Env.Set(ctx, account, secret)
			}
			return nil
		}
	}
	return c.Env.Set(ctx, account, secret)
}

// envFileSyncEnabled is true when hawk (or host) opts into ~/.hawk/env mirroring (HAWK_SECURE_CREDENTIALS=0).
func envFileSyncEnabled() bool {
	v := strings.TrimSpace(os.Getenv("HAWK_SECURE_CREDENTIALS"))
	return v == "0" || strings.EqualFold(v, "false")
}

func (c *CombinedStore) Get(ctx context.Context, account string) (string, error) {
	if c.Keychain != nil {
		if v, err := c.Keychain.Get(ctx, account); err == nil && strings.TrimSpace(v) != "" {
			return v, nil
		}
	}
	if c.Env != nil {
		return c.Env.Get(ctx, account)
	}
	return "", ErrNotFound
}

func (c *CombinedStore) Delete(ctx context.Context, account string) error {
	if c.Keychain != nil {
		_ = c.Keychain.Delete(ctx, account)
	}
	if c.Env != nil {
		_ = c.Env.Delete(ctx, account)
	}
	return nil
}
