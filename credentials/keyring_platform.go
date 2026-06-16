//go:build darwin || linux || windows

package credentials

import (
	"context"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

type keyringStore struct{}

func newPlatformKeyringStore() Store {
	return &keyringStore{}
}

// keyringWait is the hard upper bound on a single keyring operation.
// The zalando/go-keyring library does not accept context, so cancellation
// can only release the caller — the underlying call continues until the
// keyring daemon responds or hits its own timeout. This bound keeps the
// caller from waiting indefinitely on a stuck keyring.
const keyringWait = 30 * time.Second

// keyringDo runs fn with context timeout/cancellation. The underlying
// go-keyring library does not accept context.Context, so we wrap it in a
// goroutine and select to support cancellation.
//
// Caveat: cancelling the context does NOT interrupt fn — the spawned
// goroutine runs fn to completion (bounded by the keyring daemon's own
// timeout, typically a few seconds) and the result is discarded. The
// outer select returns ctx.Err() to the caller immediately so the caller
// can move on. A hard upper bound (keyringWait) caps the worst case.
func keyringDo(ctx context.Context, fn func() error) error {
	return keyringDoWithTimeout(ctx, fn, keyringWait)
}

func keyringDoWithTimeout(ctx context.Context, fn func() error, timeout time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ch := make(chan error, 1)
	go func() {
		ch <- fn()
	}()
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return context.DeadlineExceeded
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
	var r result
	err := keyringDo(ctx, func() error {
		v, e := keyring.Get(ServiceName, account)
		r.val, r.err = v, e
		return nil
	})
	if err != nil {
		return "", err
	}
	if r.err != nil {
		return "", ErrNotFound
	}
	if strings.TrimSpace(r.val) == "" {
		return "", ErrNotFound
	}
	return r.val, nil
}

func (k *keyringStore) Delete(ctx context.Context, account string) error {
	return keyringDo(ctx, func() error {
		return keyring.Delete(ServiceName, account)
	})
}
