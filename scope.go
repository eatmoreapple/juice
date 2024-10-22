/*
Copyright 2024 eatmoreapple

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
	"errors"
)

// ErrInvalidManager is an error for invalid manager.
var ErrInvalidManager = errors.New("juice: invalid manager")

// Transaction executes a transaction with the given handler.
// If the manager is not an instance of Engine, it will return ErrInvalidManager.
// If the handler returns an error, the transaction will be rolled back.
// Otherwise, the transaction will be committed.
// The ctx must should be created by ContextWithManager.
// For example:
//
//		var engine *juice.Engine
//		// ... initialize engine
//		ctx := juice.ContextWithManager(context.Background(), engine)
//	    if err := juice.Transaction(ctx, func(ctx context.Context) error {
//			// ... do something
//			return nil
//		}); err != nil {
//			// handle error
//		}
func Transaction(ctx context.Context, handler func(ctx context.Context) error) error {
	manager := ManagerFromContext(ctx)
	engine, ok := manager.(*Engine)
	if !ok {
		return ErrInvalidManager
	}
	// create a new transaction
	tx := engine.ContextTx(ctx, nil)

	if err := tx.Begin(); err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// create a new context with the transaction
	txCtx := ContextWithManager(ctx, tx)

	// call the handler
	if err := handler(txCtx); err != nil {
		return err
	}
	return tx.Commit()
}
