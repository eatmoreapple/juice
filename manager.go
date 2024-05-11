/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	cache cache.ScopeCache
}

// Object implements the GenericManager interface.
func (s *genericManager[T]) Object(v any) GenericExecutor[T] {
	exe := &genericExecutor[T]{Executor: s.Manager.Object(v)}
	// add the scopeCache middleware if the scopeCache is not nil
	if s.cache != nil {
		exe.Use(&CacheMiddleware[T]{scopeCache: s.cache})
	}
	return exe
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

// invalidTxManager is an invalid transaction xmlSQLStatement which implements the TxManager interface.
type invalidTxManager struct {
	error
}

// Object implements the Manager interface
func (i invalidTxManager) Object(_ any) Executor { return inValidExecutor(i) }

// Commit commits the transaction, but it will return an error directly.
func (i invalidTxManager) Commit() error { return i }

// Rollback rollbacks the transaction, but it will return an error directly.
func (i invalidTxManager) Rollback() error { return i }

// txManager is a transaction xmlSQLStatement
type txManager struct {
	engine *Engine
	tx     *sql.Tx
}

// Object implements the Manager interface
func (t *txManager) Object(v any) Executor {
	exe, err := t.engine.executor(v)
	if err != nil {
		return inValidExecutor(err)
	}
	exe.session = t.tx
	return t.engine.warpExecutor(exe)
}

// Commit commits the transaction
func (t *txManager) Commit() error { return t.tx.Commit() }

// Rollback rollbacks the transaction
func (t *txManager) Rollback() error { return t.tx.Rollback() }

// TxCacheManager defines a transactional scopeCache manager whose scopeCache can be accessed.
// All queries in the transaction will be cached.
// scopeCache.Flush() will be called after Commit() or Rollback().
type TxCacheManager interface {
	TxManager
	Cache() cache.ScopeCache
}

// txCacheManager implements the TxCacheManager interface.
type txCacheManager struct {
	manager TxManager
	cache   cache.ScopeCache
}

// Object implements the Manager interface.
func (t *txCacheManager) Object(v any) Executor {
	return t.manager.Object(v)
}

// Commit commits the transaction and flushes the scopeCache.
func (t *txCacheManager) Commit() error {
	defer func() { _ = t.cache.Flush(context.Background()) }()
	return t.manager.Commit()
}

// Rollback rollbacks the transaction and flushes the scopeCache.
func (t *txCacheManager) Rollback() error {
	defer func() { _ = t.cache.Flush(context.Background()) }()
	return t.manager.Rollback()
}

// Cache returns the scopeCache of the TxCacheManager.
func (t *txCacheManager) Cache() cache.ScopeCache {
	return t.cache
}

// NewTxCacheManager returns a new TxCacheManager.
func NewTxCacheManager(manager TxManager, cache cache.ScopeCache) TxCacheManager {
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

// IsTxManager returns true if the manager is a TxManager.
func IsTxManager(manager Manager) bool {
	_, ok := manager.(TxManager)
	return ok
}

// HasTxManager returns true if the context has a TxManager.
func HasTxManager(ctx context.Context) bool {
	return IsTxManager(ManagerFromContext(ctx))
}
