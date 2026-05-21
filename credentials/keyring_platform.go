//go:build darwin || linux || windows

package credentials

import (
	"context"
	"strings"

	"github.com/zalando/go-keyring"
)

type keyringStore struct{}

func newPlatformKeyringStore() Store {
	return &keyringStore{}
}

func (k *keyringStore) Set(ctx context.Context, account, secret string) error {
	_ = ctx
	return keyring.Set(ServiceName, account, secret)
}

func (k *keyringStore) Get(ctx context.Context, account string) (string, error) {
	_ = ctx
	v, err := keyring.Get(ServiceName, account)
	if err != nil {
		return "", ErrNotFound
	}
	if strings.TrimSpace(v) == "" {
		return "", ErrNotFound
	}
	return v, nil
}

func (k *keyringStore) Delete(ctx context.Context, account string) error {
	_ = ctx
	return keyring.Delete(ServiceName, account)
}
