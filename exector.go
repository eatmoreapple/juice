package juice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// Executor is an executor of SQL.
type Executor interface {
	Query(param Param) (*sql.Rows, error)
	QueryContext(ctx context.Context, param Param) (*sql.Rows, error)
	Exec(param Param) (sql.Result, error)
	ExecContext(ctx context.Context, param Param) (sql.Result, error)
	Statement() *Statement
}

// inValidExecutor is an invalid executor.
func inValidExecutor(err error) Executor {
	return &executor{err: err}
}

// executor is an executor of SQL.
type executor struct {
	session   Session
	statement *Statement
	err       error
}

// build builds the query and args.
func (e *executor) build(param Param) (query string, args []any, err error) {
	if e.err != nil {
		return "", nil, e.err
	}
	return e.Statement().Build(param)
}

// queryHandler returns the query handler.
func (e *executor) queryHandler() QueryHandler {
	next := sessionQueryHandler(e.session)
	return e.Statement().Engine().middlewares.QueryContext(e.Statement(), next)
}

// execHandler returns the exec handler.
func (e *executor) execHandler() ExecHandler {
	next := sessionExecHandler(e.session)
	return e.Statement().Engine().middlewares.ExecContext(e.Statement(), next)
}

// Query executes the query and returns the result.
func (e *executor) Query(param Param) (*sql.Rows, error) {
	return e.QueryContext(context.Background(), param)
}

// QueryContext executes the query and returns the result.
func (e *executor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	query, args, err := e.build(param)
	if err != nil {
		return nil, err
	}
	return e.queryHandler()(ctx, query, args...)
}

// Exec executes the query and returns the result.
func (e *executor) Exec(param Param) (sql.Result, error) {
	return e.ExecContext(context.Background(), param)
}

// ExecContext executes the query and returns the result.
func (e *executor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	query, args, err := e.build(param)
	if err != nil {
		return nil, err
	}
	ret, err := e.execHandler()(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	stmt := e.Statement()
	// If the statement is not an insert statement, return the result directly.
	if !stmt.IsInsert() {
		return ret, nil
	}

	// If the useGeneratedKeys is not set or false, return the result directly.
	if stmt.Attribute("useGeneratedKeys") != "true" {
		return ret, nil
	}

	// checkout the input param
	rv := reflect.ValueOf(param)

	// If the useGeneratedKeys is set and true but the param is not a pointer.
	if rv.Kind() != reflect.Ptr {
		return nil, errors.New("useGeneratedKeys is true, but the param is not a pointer")
	}

	rv = reflect.Indirect(rv)

	// If the useGeneratedKeys is set and true but the param is not a struct pointer.
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("useGeneratedKeys is true, but the param is not a struct pointer")
	}

	var field reflect.Value

	// keyProperty is the name of the field that will be set the generated key.
	keyProperty := stmt.Attribute("keyProperty")
	// The keyProperty is empty, return the result directly.
	if len(keyProperty) == 0 {
		ty := rv.Type()
		// If the keyProperty is empty, try to find from the tag.
		for i := 0; i < ty.NumField(); i++ {
			if autoIncr := ty.Field(i).Tag.Get("autoincr"); autoIncr == "true" {
				field = rv.Field(i)
				keyProperty = ty.Field(i).Name
				break
			}
		}
		if !field.IsValid() {
			return nil, errors.New("keyProperty not set or not tag named `autoincr`")
		}
	} else {
		// try to find the field from the given struct.
		field = rv.FieldByName(keyProperty)
		if !field.IsValid() {
			return nil, fmt.Errorf("the keyProperty %s is not found", keyProperty)
		}
	}

	// If the field is not an int, return the result directly.
	if !field.CanInt() {
		return nil, fmt.Errorf("the keyProperty %s is not a int", keyProperty)
	}

	// get the last insert id
	id, err := ret.LastInsertId()
	if err != nil {
		return nil, err
	}
	// set the id to the field
	field.SetInt(id)
	return ret, nil
}

// Statement returns the statement.
func (e *executor) Statement() *Statement {
	return e.statement
}

// GenericExecutor is a generic executor.
type GenericExecutor[T any] interface {
	Query(param Param) (T, error)
	QueryContext(ctx context.Context, param Param) (T, error)
	Exec(param Param) (sql.Result, error)
	ExecContext(ctx context.Context, param Param) (sql.Result, error)
}

// genericExecutor is a generic executor.
type genericExecutor[T any] struct {
	Executor
}

// Query executes the query and returns the scanner.
func (e *genericExecutor[T]) Query(p Param) (T, error) {
	return e.QueryContext(context.Background(), p)
}

// QueryContext executes the query and returns the scanner.
func (e *genericExecutor[T]) QueryContext(ctx context.Context, p Param) (result T, err error) {
	rows, err := e.Executor.QueryContext(ctx, p)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	retMap, err := e.Executor.Statement().ResultMap()

	// set but not found
	if err != nil {
		if !errors.Is(err, ErrResultMapNotSet) {
			return result, err
		}
	}

	rv := reflect.ValueOf(result)

	switch rv.Kind() {
	case reflect.Ptr:
		// if T is a pointer, then set prt to T
		value := reflect.New(rv.Type().Elem()).Interface().(T)
		// NOTE: create an object using with the reflection may be slow, but it is not a big problem.
		// You should better use the direct type instead of the pointer type.
		if err = BindWithResultMap(rows, value, retMap); err != nil {
			// if bind failed, then return the original value
			// result is a zero value
			return result, err
		}
		// if bind success, then return the new value
		result = value
	default:
		// bind the result to the pointer
		err = BindWithResultMap(rows, &result, retMap)
	}
	return
}

// Exec executes the query and returns the result.
func (e *genericExecutor[_]) Exec(p Param) (sql.Result, error) {
	return e.ExecContext(context.Background(), p)
}

// ExecContext executes the query and returns the result.
func (e *genericExecutor[_]) ExecContext(ctx context.Context, p Param) (sql.Result, error) {
	return e.Executor.ExecContext(ctx, p)
}

var _ GenericExecutor[any] = (*genericExecutor[any])(nil)

// BinderExecutor is a binder executor.
// It is used to bind the result to the given value.
type BinderExecutor interface {
	Query(param Param) (Binder, error)
	QueryContext(ctx context.Context, param Param) (Binder, error)
	Exec(param Param) (sql.Result, error)
	ExecContext(ctx context.Context, param Param) (sql.Result, error)
}

// binderExecutor is a binder executor.
// binderExecutor implements the BinderExecutor interface.
type binderExecutor struct {
	Executor
}

// Query executes the query and returns the scanner.
func (b *binderExecutor) Query(param Param) (Binder, error) {
	return b.QueryContext(context.Background(), param)
}

// QueryContext executes the query and returns the scanner.
func (b *binderExecutor) QueryContext(ctx context.Context, param Param) (Binder, error) {
	rows, err := b.Executor.QueryContext(ctx, param)
	if err != nil {
		return nil, err
	}
	retMap, err := b.Executor.Statement().ResultMap()
	if err != nil && !errors.Is(err, ErrResultMapNotSet) {
		return nil, err
	}
	return &rowsBinder{rows: rows, mapper: retMap}, nil
}

var _ BinderExecutor = (*binderExecutor)(nil)
