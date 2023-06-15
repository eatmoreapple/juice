package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

// ErrCacheNotFound is the error that cache not found.
var ErrCacheNotFound = errors.New("juice: cache not found")

type Cache interface {
	// Set sets the value for the key.
	Set(ctx context.Context, key string, value any) error

	// Get gets the value for the key.
	// If the value does not exist, it returns ErrCacheNotFound.
	Get(ctx context.Context, key string, dst any) error

	// Flush flushes all the cache.
	Flush(ctx context.Context) error
}

type memeryCache struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func (m *memeryCache) Set(_ context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = data
	return nil
}

func (m *memeryCache) Get(_ context.Context, key string, dst any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[key]
	if !ok {
		return ErrCacheNotFound
	}
	return json.Unmarshal(data, dst)
}

func (m *memeryCache) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.data {
		delete(m.data, k)
	}
	return nil
}

// New returns a memery cache.
func New() Cache {
	return new(memeryCache)
}
