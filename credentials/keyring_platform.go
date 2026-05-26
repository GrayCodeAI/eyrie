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

// keyringDo runs fn with context timeout/cancellation. The underlying
// go-keyring library does not accept context.Context, so we wrap it with a
// goroutine and select to support cancellation.
func keyringDo(ctx context.Context, fn func() error) error {
	ch := make(chan error, 1)
	go func() {
		ch <- fn()
	}()
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (k *keyringStore) Set(ctx context.Context, account, secret string) error {
	return keyringDo(ctx, func() error {
		return keyring.Set(ServiceName, account, secret)
	})
}

func (k *keyringStore) Get(ctx context.Context, account string) (string, error) {
	type result struct {
		val string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		v, err := keyring.Get(ServiceName, account)
		ch <- result{v, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			return "", ErrNotFound
		}
		if strings.TrimSpace(r.val) == "" {
			return "", ErrNotFound
		}
		return r.val, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (k *keyringStore) Delete(ctx context.Context, account string) error {
	return keyringDo(ctx, func() error {
		return keyring.Delete(ServiceName, account)
	})
}
