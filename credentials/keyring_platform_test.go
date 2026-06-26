//go:build darwin || linux || windows

package credentials

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestKeyringDo_NormalReturn verifies keyringDo returns fn's result when fn
// completes before context cancellation.
func TestKeyringDo_NormalReturn(t *testing.T) {
	t.Parallel()
	err := keyringDoWithTimeout(context.Background(), func() error { return nil }, time.Second)
	if err != nil {
		t.Errorf("keyringDo(nil) = %v, want nil", err)
	}

	want := errors.New("sentinel")
	err = keyringDoWithTimeout(context.Background(), func() error { return want }, time.Second)
	if !errors.Is(err, want) {
		t.Errorf("keyringDo(sentinel) = %v, want %v", err, want)
	}
}

// TestKeyringDo_AlreadyCancelledContext verifies keyringDo returns ctx.Err()
// immediately (and does NOT spawn the inner goroutine) when the context is
// already done.
func TestKeyringDo_AlreadyCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := keyringDoWithTimeout(ctx, func() error {
		t.Error("fn should not be invoked when ctx is already cancelled")
		return nil
	}, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// TestKeyringDo_ContextCancel_ReturnsPromptly is the caller-side leak
// regression test: after cancelling the context, keyringDo returns
// ctx.Err() within a short window even though the inner goroutine may
// still be running. The caller must not be blocked by the underlying
// non-cancellable keyring call.
//
// Note: the inner goroutine continues to run fn to completion (bounded
// by the keyring daemon's own timeout). The spawned goroutine is the
// unavoidable cost of using a ctx-unaware keyring library; keyringDo
// guarantees only that the caller is released promptly.
func TestKeyringDo_ContextCancel_ReturnsPromptly(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)

	fn := func() error {
		close(started)
		<-release
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	keyringDone := make(chan error, 1)
	go func() { keyringDone <- keyringDoWithTimeout(ctx, fn, 5*time.Second) }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("fn never started; keyringDo may be deadlocked")
	}

	cancelStart := time.Now()
	cancel()

	select {
	case err := <-keyringDone:
		elapsed := time.Since(cancelStart)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("keyringDo err = %v, want context.Canceled", err)
		}
		if elapsed > 200*time.Millisecond {
			t.Errorf("keyringDo took %v to return after cancel; should be < 200ms", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("keyringDo did not return after ctx cancel")
	}
}

// TestKeyringDo_ContextTimeout verifies keyringDo respects context deadlines.
func TestKeyringDo_ContextTimeout(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	defer close(release)

	fn := func() error {
		<-release
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- keyringDoWithTimeout(ctx, fn, 5*time.Second) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("err = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("keyringDo did not return after timeout")
	}
}

// TestKeyringDo_HardTimeout verifies the bounded-wait safety net: when
// the keyring daemon is permanently stuck, the caller is released by
// the timeout rather than waiting forever. The inner goroutine continues
// to run, but the caller gets a deterministic deadline.
func TestKeyringDo_HardTimeout(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	// intentionally not closing release — simulates a stuck keyring

	fn := func() error {
		<-release
		return nil
	}

	start := time.Now()
	err := keyringDoWithTimeout(context.Background(), fn, 50*time.Millisecond)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("hard timeout took %v; expected ~50ms", elapsed)
	}
	close(release)
}

// TestKeyringStore_Get_CancelledContext exercises the Get path that was
// independently leaking (its own inline select+goroutine before the
// refactor moved it onto keyringDo).
func TestKeyringStore_Get_CancelledContext(t *testing.T) {
	t.Parallel()
	k := &keyringStore{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := k.Get(ctx, "any-account")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Get err = %v, want context.Canceled", err)
	}
}
