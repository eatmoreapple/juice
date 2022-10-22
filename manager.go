package juice

import (
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
	return &genericManager[T]{manager}
}

// genericManager implements the GenericManager interface.
type genericManager[T any] struct {
	manager Manager
}

// Object implements the GenericManager interface.
func (s *genericManager[T]) Object(v any) GenericExecutor[T] {
	exe := s.manager.Object(v)
	return &genericExecutor[T]{Executor: exe}
}

// TxManager is a transactional mapper executor
type TxManager interface {
	Manager
	Commit() error
	Rollback() error
}

// txManager is a transaction statement
type txManager struct {
	manager Manager
	tx      *sql.Tx
	err     error
}

// Object implements the Manager interface
func (t *txManager) Object(v any) Executor {
	if t.err != nil {
		return inValidExecutor(t.err)
	}
	return t.manager.Object(v)
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