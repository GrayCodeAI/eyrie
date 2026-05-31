package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func newTestBudgetStore(t *testing.T) *BudgetStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "budgets.db")
	s, err := OpenBudgetStore(path)
	if err != nil {
		t.Fatalf("open budget store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestBudgetStore_CRUDAndEnforcement(t *testing.T) {
	s := newTestBudgetStore(t)
	ctx := context.Background()

	if err := s.CreateVirtualKey(ctx, "team-a", "Team A", "openai", 1.00); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.CheckBudget(ctx, "team-a", 0.40); err != nil {
		t.Fatalf("under budget should pass: %v", err)
	}
	if err := s.RecordUsage(ctx, "team-a", 0.70, 100, 200); err != nil {
		t.Fatalf("record: %v", err)
	}

	// 0.70 used + 0.40 est > 1.00 → exceeded.
	if err := s.CheckBudget(ctx, "team-a", 0.40); !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("expected ErrBudgetExceeded, got %v", err)
	}

	vk, err := s.Get(ctx, "team-a")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if vk.UsedUSD != 0.70 || vk.TokensIn != 100 || vk.TokensOut != 200 {
		t.Errorf("unexpected state: %+v", vk)
	}
}

func TestBudgetStore_UnknownKey(t *testing.T) {
	s := newTestBudgetStore(t)
	ctx := context.Background()
	if err := s.CheckBudget(ctx, "ghost", 0.01); !errors.Is(err, ErrUnknownVirtualKey) {
		t.Errorf("expected ErrUnknownVirtualKey, got %v", err)
	}
	if err := s.RecordUsage(ctx, "ghost", 0.01, 1, 1); !errors.Is(err, ErrUnknownVirtualKey) {
		t.Errorf("expected ErrUnknownVirtualKey on record, got %v", err)
	}
}

func TestBudgetStore_UnlimitedAndSecrets(t *testing.T) {
	s := newTestBudgetStore(t)
	ctx := context.Background()
	if err := s.CreateVirtualKey(ctx, "free", "Free", "anthropic", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckBudget(ctx, "free", 9999); err != nil {
		t.Errorf("zero limit should be unlimited, got %v", err)
	}

	if err := s.SetProviderSecret(ctx, "free", "anthropic", "sk-real-123"); err != nil {
		t.Fatal(err)
	}
	got, err := s.ProviderSecret(ctx, "free", "anthropic")
	if err != nil || got != "sk-real-123" {
		t.Errorf("provider secret roundtrip failed: %q err=%v", got, err)
	}
}
