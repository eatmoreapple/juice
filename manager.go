package juice

import (
	"context"
	"database/sql"
	"github.com/eatmoreapple/juice/cache"
)

// Manager is an interface for managing database operations.
type Manager interface {
	Object(v any) Executor
}

// GenericManager is an interface for managing database operations.
type GenericManager[T any] interface {
	Object(v any) GenericExecutor[T]
}

// NewGenericManager returns a new GenericManager.
func NewGenericManager[T any](manager Manager) GenericManager[T] {
	m := &genericManager[T]{Manager: manager}
	if tcm, ok := manager.(TxCacheManager); ok {
		m.cache = tcm.Cache()
	}
	return m
}

// genericManager implements the GenericManager interface.
type genericManager[T any] struct {
	Manager
	cache cache.Cache
}

// Object implements the GenericManager interface.
func (s *genericManager[T]) Object(v any) GenericExecutor[T] {
	exe := s.Manager.Object(v)
	return &genericExecutor[T]{Executor: exe, cache: s.cache}
}

// TxManager is a transactional mapper executor
type TxManager interface {
	Manager
	// Commit commits the transaction.
	Commit() error
	// Rollback rollbacks the transaction.
	// The rollback will be ignored if the tx has been committed.
	Rollback() error
}

// txManager is a transaction statement
type txManager struct {
	engine *Engine
	tx     *sql.Tx
	err    error
}

// Object implements the Manager interface
func (t *txManager) Object(v any) Executor {
	if t.err != nil {
		return inValidExecutor(t.err)
	}
	exe, err := t.engine.executor(v)
	if err != nil {
		return inValidExecutor(err)
	}
	exe.session = t.tx
	return exe
}

// Commit commits the transaction
func (t *txManager) Commit() error {
	if t.err != nil {
		return t.err
	}
	return t.tx.Commit()
}

// Rollback rollbacks the transaction
func (t *txManager) Rollback() error {
	if t.err != nil {
		return t.err
	}
	return t.tx.Rollback()
}

// TxCacheManager defines a transactional cache manager whose cache can be accessed.
// All queries in the transaction will be cached.
// cache.Flush() will be called after Commit() or Rollback().
type TxCacheManager interface {
	TxManager
	Cache() cache.Cache
}

// txCacheManager implements the TxCacheManager interface.
type txCacheManager struct {
	manager TxManager
	cache   cache.Cache
}

// Object implements the Manager interface.
func (t *txCacheManager) Object(v any) Executor {
	return t.manager.Object(v)
}

// Commit commits the transaction and flushes the cache.
func (t *txCacheManager) Commit() error {
	defer func() { _ = t.cache.Flush(context.Background()) }()
	return t.manager.Commit()
}

// Rollback rollbacks the transaction and flushes the cache.
func (t *txCacheManager) Rollback() error {
	defer func() { _ = t.cache.Flush(context.Background()) }()
	return t.manager.Rollback()
}

// Cache returns the cache of the TxCacheManager.
func (t *txCacheManager) Cache() cache.Cache {
	return t.cache
}

// NewTxCacheManager returns a new TxCacheManager.
func NewTxCacheManager(manager TxManager, cache cache.Cache) TxCacheManager {
	return &txCacheManager{manager: manager, cache: cache}
}

type managerKey struct{}

// ManagerFromContext returns the Manager from the context.
func ManagerFromContext(ctx context.Context) Manager {
	return ctx.Value(managerKey{}).(Manager)
}

// ContextWithManager returns a new context with the given Manager.
func ContextWithManager(ctx context.Context, manager Manager) context.Context {
	return context.WithValue(ctx, managerKey{}, manager)
}
