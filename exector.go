package juice

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eatmoreapple/juice/cache"
	"reflect"
)

// Executor is an executor of SQL.
type Executor interface {
	// Query executes a query that returns rows, typically a SELECT.
	// The param are the placeholder collection for this query.
	Query(param Param) (*sql.Rows, error)

	// QueryContext executes a query that returns rows, typically a SELECT.
	// The param are the placeholder collection for this query.
	QueryContext(ctx context.Context, param Param) (*sql.Rows, error)

	// Exec executes a query without returning any rows.
	// The param are the placeholder collection for this query.
	Exec(param Param) (sql.Result, error)

	// ExecContext executes a query without returning any rows.
	// The param are the placeholder collection for this query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the statement of the current executor.
	Statement() *Statement

	// Session returns the session of the current executor.
	Session() Session
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
	return e.Statement().QueryHandler(e.Session())(ctx, query, args...)
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
	ret, err := e.Statement().ExecHandler(e.Session())(ctx, query, args...)
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

func (e *executor) Session() Session {
	return e.session
}

// GenericExecutor is a generic executor.
type GenericExecutor[T any] interface {
	// Query executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	Query(param Param) (T, error)

	// QueryContext executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, param Param) (T, error)

	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(param Param) (sql.Result, error)

	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the statement of the current executor.
	Statement() *Statement

	// Session returns the session of the current executor.
	Session() Session
}

// genericExecutor is a generic executor.
type genericExecutor[T any] struct {
	Executor
	cache cache.Cache
}

// Query executes the query and returns the scanner.
func (e *genericExecutor[T]) Query(p Param) (T, error) {
	return e.QueryContext(context.Background(), p)
}

// QueryContext executes the query and returns the scanner.
func (e *genericExecutor[T]) QueryContext(ctx context.Context, p Param) (result T, err error) {
	query, args, err := e.Executor.Statement().Build(p)
	if err != nil {
		return
	}
	var ptr any = &result

	rv := reflect.ValueOf(result)

	if rv.Kind() == reflect.Ptr {
		result = reflect.New(rv.Type().Elem()).Interface().(T)
		ptr = result
	}

	// If the cache is enabled and cache is not disabled in this statement.
	if e.cache != nil && e.Statement().Attribute("cache") != "false" {
		// cacheKey is the key which is used to get the result and put the result to the cache.
		var cacheKey string

		// CacheKeyFunc is the function which is used to generate the cache key.
		// default is the md5 of the query and args.
		// reset the CacheKeyFunc variable to change the default behavior.
		cacheKey, err = CacheKeyFunc(query, args)
		if err != nil {
			return
		}

		// try to get the result from the cache
		// if the result is found, return it directly.
		if err = e.cache.Get(ctx, cacheKey, ptr); err == nil {
			return
		}

		// ErrCacheNotFound means the cache is not found,
		// we should continue to query the database.
		if !errors.Is(err, cache.ErrCacheNotFound) {
			return
		}
		// put the result to the cache
		defer func() {
			if err == nil {
				err = e.cache.Set(ctx, cacheKey, result)
			}
		}()
	}

	// try to query the database.
	rows, err := e.Statement().QueryHandler(e.Session())(ctx, query, args...)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	retMap, err := e.Executor.Statement().ResultMap()

	// ErrResultMapNotSet means the result map is not set, use the default result map.
	if err != nil {
		if !errors.Is(err, ErrResultMapNotSet) {
			return
		}
	}
	err = BindWithResultMap(rows, ptr, retMap)
	return
}

// Exec executes the query and returns the result.
func (e *genericExecutor[_]) Exec(p Param) (sql.Result, error) {
	return e.ExecContext(context.Background(), p)
}

// ExecContext executes the query and returns the result.
func (e *genericExecutor[_]) ExecContext(ctx context.Context, p Param) (ret sql.Result, err error) {
	defer func() {
		if err == nil && e.cache != nil {
			// clear the cache
			_ = e.cache.Flush(ctx)
		}
	}()
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

// cacheKeyFunc defines the function which is used to generate the cache key.
type cacheKeyFunc func(query string, args []any) (string, error)

// CacheKeyFunc is the function which is used to generate the cache key.
// default is the md5 of the query and args.
// reset the CacheKeyFunc variable to change the default behavior.
var CacheKeyFunc cacheKeyFunc = func(query string, args []any) (string, error) {
	writer := md5.New()
	writer.Write([]byte(query))
	if len(args) > 0 {
		item, err := json.Marshal(args)
		if err != nil {
			return "", err
		}
		writer.Write(item)
	}
	return hex.EncodeToString(writer.Sum(nil)), nil
}
