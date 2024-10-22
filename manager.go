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
	"github.com/eatmoreapple/juice/session"
)

// Manager is an interface for managing database operations.
type Manager interface {
	Object(v any) SQLRowsExecutor
}

// GenericManager is an interface for managing database operations.
type GenericManager[T any] interface {
	Object(v any) Executor[T]
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
func (s *genericManager[T]) Object(v any) Executor[T] {
	exe := &GenericExecutor[T]{
		SQLRowsExecutor: s.Manager.Object(v),
		cache:           s.cache,
	}
	return exe
}

// TxManager is a transactional mapper sqlRowsExecutor
type TxManager interface {
	Manager

	// Begin begins the transaction.
	Begin() error

	// Commit commits the transaction.
	Commit() error

	// Rollback rollbacks the transaction.
	Rollback() error
}

// txManager is a transaction xmlSQLStatement
type txManager struct {
	// engine is the engine of the transaction.
	engine *Engine

	// txOptions is the transaction options.
	// If nil, the default options will be used.
	txOptions *sql.TxOptions

	// tx is the transaction session if the transaction is begun.
	tx  session.TransactionSession
	ctx context.Context
}

// Object implements the Manager interface
func (t *txManager) Object(v any) SQLRowsExecutor {
	if t.tx == nil {
		return inValidExecutor(session.ErrTransactionNotBegun)
	}
	stat, err := t.engine.GetConfiguration().GetStatement(v)
	if err != nil {
		return inValidExecutor(err)
	}
	drv := t.engine.driver
	handler := NewSQLRowsStatementHandler(drv, t.tx, t.engine.middlewares...)
	return &sqlRowsExecutor{
		statement:        stat,
		statementHandler: handler,
		driver:           drv,
	}
}

// Begin begins the transaction
func (t *txManager) Begin() error {
	// If the transaction is already begun, return an error directly.
	if t.tx != nil {
		return session.ErrTransactionAlreadyBegun
	}
	tx, err := t.engine.DB().BeginTx(t.ctx, t.txOptions)
	if err != nil {
		return err
	}
	t.tx = tx
	return nil
}

// Commit commits the transaction
func (t *txManager) Commit() error {
	// If the transaction is not begun, return an error directly.
	if t.tx == nil {
		return session.ErrTransactionNotBegun
	}
	return t.tx.Commit()
}

// Rollback rollbacks the transaction
func (t *txManager) Rollback() error {
	// If the transaction is not begun, return an error directly.
	if t.tx == nil {
		return session.ErrTransactionNotBegun
	}
	return t.tx.Rollback()
}

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
func (t *txCacheManager) Object(v any) SQLRowsExecutor {
	return t.manager.Object(v)
}

func (t *txCacheManager) Begin() error {
	return t.manager.Begin()
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

// managerFromContext returns the Manager from the context.
func managerFromContext(ctx context.Context) (Manager, bool) {
	manager, ok := ctx.Value(managerKey{}).(Manager)
	return manager, ok
}

// ManagerFromContext returns the Manager from the context.
func ManagerFromContext(ctx context.Context) Manager {
	manager, _ := managerFromContext(ctx)
	return manager
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
