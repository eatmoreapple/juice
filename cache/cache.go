package cache

import (
	"context"
	"errors"
)

// ErrCacheNotFound is the error that cache not found.
var ErrCacheNotFound = errors.New("juice: cache not found")

// ScopeCache is an interface for transactional cache.
type ScopeCache interface {
	// Set sets the value for the key.
	Set(ctx context.Context, key string, value any) error

	// Get gets the value for the key.
	// If the value does not exist, it should return ErrCacheNotFound.
	Get(ctx context.Context, key string, prt any) error

	// Flush flushes all the cache.
	// It will be called after Commit() or Rollback().
	Flush(ctx context.Context) error
}
