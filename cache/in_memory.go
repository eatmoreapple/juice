package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"sync"
)

// bufPool is a pool of bytes.Buffer pointers, used to reduce memory allocations.
var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// getBufferFromPool retrieves a bytes.Buffer pointer from the pool.
func getBufferFromPool() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

// putBufferToPool resets the buffer and puts it back into the pool.
func putBufferToPool(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

// inMemoryScopeCache is a thread-safe in-memory cache.
type inMemoryScopeCache struct {
	data map[string][]byte // The actual cache data, stored as byte slices.
	mu   sync.RWMutex      // Mutex for thread-safe operations.
}

// Set encodes the value and stores it in the cache under the specified key.
func (m *inMemoryScopeCache) Set(_ context.Context, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	var buf = getBufferFromPool()
	defer putBufferToPool(buf)
	if err := gob.NewEncoder(buf).Encode(value); err != nil {
		return err
	}
	m.data[key] = buf.Bytes()
	return nil
}

// Get retrieves the value associated with the key from the cache and decodes it.
func (m *inMemoryScopeCache) Get(_ context.Context, key string, prt any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[key]
	if !ok {
		return ErrCacheNotFound
	}
	buf := bytes.NewReader(data)
	return gob.NewDecoder(buf).Decode(prt)
}

// Flush clears all data from the cache.
func (m *inMemoryScopeCache) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.data)
	return nil
}

// InMemoryScopeCache returns a new instance of inMemoryScopeCache.
func InMemoryScopeCache() ScopeCache {
	return new(inMemoryScopeCache)
}
