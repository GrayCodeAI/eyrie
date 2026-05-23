package credentials

import (
	"context"
	"strings"
	"sync"
)

// MapStore is an in-memory credential store for tests.
type MapStore struct {
	mu   sync.RWMutex
	Data map[string]string
}

func (m *MapStore) accountKey(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}

func (m *MapStore) Set(ctx context.Context, account, secret string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Data == nil {
		m.Data = map[string]string{}
	}
	m.Data[m.accountKey(account)] = strings.TrimSpace(secret)
	return nil
}

func (m *MapStore) Get(ctx context.Context, account string) (string, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Data == nil {
		return "", ErrNotFound
	}
	v, ok := m.Data[m.accountKey(account)]
	if !ok || strings.TrimSpace(v) == "" {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *MapStore) Delete(ctx context.Context, account string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Data != nil {
		delete(m.Data, m.accountKey(account))
	}
	return nil
}
