package juice

import (
	"context"
	"database/sql"
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
	return &genericManager[T]{Manager: manager}
}

// genericManager implements the GenericManager interface.
type genericManager[T any] struct {
	Manager
}

// Object implements the GenericManager interface.
func (s *genericManager[T]) Object(v any) GenericExecutor[T] {
	exe := s.Manager.Object(v)
	return &genericExecutor[T]{Executor: exe}
}

type BinderManager interface {
	Object(v any) BinderExecutor
}

// NewBinderManager returns a new BinderManager.
func NewBinderManager(manager Manager) BinderManager {
	return &binderManager{manager}
}

// binderManager implements the BinderManager interface.
type binderManager struct {
	Manager
}

// Object implements the BinderManager interface.
func (b *binderManager) Object(v any) BinderExecutor {
	exe := b.Manager.Object(v)
	return &binderExecutor{Executor: exe}
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

type managerKey struct{}

// ManagerFromContext returns the Manager from the context.
func ManagerFromContext(ctx context.Context) Manager {
	return ctx.Value(managerKey{}).(Manager)
}

// ContextWithManager returns a new context with the given Manager.
func ContextWithManager(ctx context.Context, manager Manager) context.Context {
	return context.WithValue(ctx, managerKey{}, manager)
}
