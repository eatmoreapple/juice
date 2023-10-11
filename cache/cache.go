package cache

import (
	"context"
	"errors"
	"sync"
)

// ErrCacheNotFound is the error that cache not found.
var ErrCacheNotFound = errors.New("juice: cache not found")

// ScopeCache is an interface for transactional cache.
type ScopeCache interface {
	// Set sets the value for the key.
	Set(ctx context.Context, key string, value any) error

	// Get gets the value for the key.
	// If the value does not exist, it should return ErrCacheNotFound.
	Get(ctx context.Context, key string) (any, error)

	// Flush flushes all the cache.
	// It will be called after Commit() or Rollback().
	Flush(ctx context.Context) error
}

type inMemoryScopeCache struct {
	data map[string]any
	mu   sync.RWMutex
}

func (m *inMemoryScopeCache) Set(_ context.Context, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]any)
	}
	m.data[key] = value
	return nil
}

func (m *inMemoryScopeCache) Get(_ context.Context, key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[key]
	if !ok {
		return nil, ErrCacheNotFound
	}
	return data, nil
}

func (m *inMemoryScopeCache) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.data)
	return nil
}

// InMemoryScopeCache returns an ScopeCache instance.
func InMemoryScopeCache() ScopeCache {
	return new(inMemoryScopeCache)
}
