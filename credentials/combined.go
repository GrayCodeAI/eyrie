package credentials

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// CombinedStore persists secrets in the OS secret store (macOS Keychain / Linux Secret Service).
type CombinedStore struct {
	Keychain Store
	mu       sync.RWMutex
	cache    map[string]combinedCacheEntry
}

type combinedCacheEntry struct {
	secret   string
	found    bool
	cachedAt time.Time
}

const combinedGetCacheTTL = 2 * time.Second

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
		return ErrKeychainUnavailable
	}
	if err := c.Keychain.Set(ctx, account, secret); err != nil {
		return fmt.Errorf("%w: %s", err, KeyringUnavailableHelp())
	}
	c.invalidateAccount(account)
	return nil
}

func (c *CombinedStore) Get(ctx context.Context, account string) (string, error) {
	if c.Keychain == nil {
		return "", ErrNotFound
	}
	if secret, ok := c.cachedGet(account); ok {
		if secret == "" {
			return "", ErrNotFound
		}
		return secret, nil
	}
	secret, err := c.Keychain.Get(ctx, account)
	if err == nil {
		c.storeCachedGet(account, secret, true)
		return secret, nil
	}
	if err == ErrNotFound {
		c.storeCachedGet(account, "", false)
	}
	return secret, err
}

func (c *CombinedStore) Delete(ctx context.Context, account string) error {
	if c.Keychain == nil {
		return nil
	}
	err := c.Keychain.Delete(ctx, account)
	if err == nil {
		c.invalidateAccount(account)
	}
	return err
}

func (c *CombinedStore) cachedGet(account string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cache == nil {
		return "", false
	}
	entry, ok := c.cache[c.cacheKey(account)]
	if !ok || time.Since(entry.cachedAt) >= combinedGetCacheTTL {
		return "", false
	}
	if !entry.found {
		return "", true
	}
	return entry.secret, true
}

func (c *CombinedStore) storeCachedGet(account, secret string, found bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[string]combinedCacheEntry)
	}
	c.cache[c.cacheKey(account)] = combinedCacheEntry{
		secret:   strings.TrimSpace(secret),
		found:    found,
		cachedAt: time.Now(),
	}
}

func (c *CombinedStore) invalidateAccount(account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		return
	}
	delete(c.cache, c.cacheKey(account))
}

func (c *CombinedStore) cacheKey(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}
