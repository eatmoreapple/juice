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
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/eatmoreapple/juice/internal/reflectlite"

	"github.com/eatmoreapple/juice/ctxreducer"
	"github.com/eatmoreapple/juice/driver"
	"github.com/eatmoreapple/juice/session"
)

type Statement interface {
	ID() string
	Name() string
	Attribute(key string) string
	Action() Action
	Configuration() IConfiguration
	ResultMap() (ResultMap, error)
	Build(translator driver.Translator, param Param) (query string, args []any, err error)
}

type StatementHandler interface {
	ExecContext(ctx context.Context, statement Statement, param Param) (sql.Result, error)
	QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error)
}

var formatRegexp = regexp.MustCompile(`\$\{ *?([a-zA-Z0-9_\.]+) *?\}`)

// xmlSQLStatement defines a sql xmlSQLStatement.
type xmlSQLStatement struct {
	mapper *Mapper
	action Action
	Nodes  NodeGroup
	attrs  map[string]string
	name   string
	id     string
}

// Attribute returns the value of the attribute with the given key.
func (s *xmlSQLStatement) Attribute(key string) string {
	value := s.attrs[key]
	if value == "" {
		value = s.mapper.Attribute(key)
	}
	return value
}

// setAttribute sets the attribute with the given key and value.
func (s *xmlSQLStatement) setAttribute(key, value string) {
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[key] = value
}

// ID returns the unique key of the namespace.
func (s *xmlSQLStatement) ID() string {
	return s.id
}

func (s *xmlSQLStatement) lazyName() string {
	var builder = getStringBuilder()
	defer putStringBuilder(builder)
	if prefix := s.mapper.mappers.Prefix(); prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString(".")
	}
	builder.WriteString(s.mapper.namespace)
	builder.WriteString(".")
	builder.WriteString(s.id)
	return builder.String()
}

// Name is a unique key of the whole xmlSQLStatement.
func (s *xmlSQLStatement) Name() string {
	if s.name == "" {
		s.name = s.lazyName()
	}
	return s.name
}

// Action returns the action of the xmlSQLStatement.
func (s *xmlSQLStatement) Action() Action {
	return s.action
}

// Configuration returns the configuration of the xmlSQLStatement.
func (s *xmlSQLStatement) Configuration() IConfiguration {
	return s.mapper.Configuration()
}

// ResultMap returns the ResultMap of the xmlSQLStatement.
func (s *xmlSQLStatement) ResultMap() (ResultMap, error) {
	key := s.Attribute("resultMap")
	if key == "" {
		return nil, ErrResultMapNotSet
	}
	return s.mapper.GetResultMapByID(key)
}

// Build builds the xmlSQLStatement with the given parameter.
func (s *xmlSQLStatement) Build(translator driver.Translator, param Param) (query string, args []any, err error) {
	value := newGenericParam(param, s.Attribute("paramName"))
	query, args, err = s.Nodes.Accept(translator, value)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, ErrEmptyQuery
	}
	return query, args, nil
}

type BatchSQLRowsStatementHandler struct {
	driver      driver.Driver
	middlewares MiddlewareGroup
	session     session.Session
}

func (b *BatchSQLRowsStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	statementHandler := NewSQLRowsStatementHandler(b.driver, b.session, b.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

// ExecContext executes a batch of SQL statements within a context. It handles
// the execution of SQL statements in batches if the action is an Insert and a
// batch size is specified. If the action is not an Insert or no batch size is
// specified, it delegates to the execContext method.
func (b *BatchSQLRowsStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (result sql.Result, err error) {
	if statement.Action() != Insert {
		return b.execContext(ctx, statement, param)
	}
	batchSizeValue := statement.Attribute("batchSize")
	if len(batchSizeValue) == 0 {
		return b.execContext(ctx, statement, param)
	}
	batchSize, err := strconv.ParseInt(batchSizeValue, 10, 64)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("failed to parse batch size: %s", batchSizeValue))
	}
	if batchSize <= 0 {
		return nil, errors.New("batch size must be greater than 0")
	}
	// ensure the param is a slice or array
	value := reflectlite.ValueOf(param)

	switch value.IndirectType().Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return nil, errSliceOrArrayRequired
	}

	unwrapValue := value.Unwrap()
	length := unwrapValue.Len()
	if length == 0 {
		return nil, errors.New("invalid param length")
	}
	times := (length + int(batchSize) - 1) / int(batchSize)

	// execute the statement in batches.
	for i := 0; i < times; i++ {
		start := i * int(batchSize)
		end := (i + 1) * int(batchSize)
		if end > length {
			end = length
		}
		batchParam := unwrapValue.Slice(start, end).Interface()
		result, err = b.execContext(ctx, statement, batchParam)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (b *BatchSQLRowsStatementHandler) execContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	statementHandler := NewSQLRowsStatementHandler(b.driver, b.session, b.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

// NewBatchSQLRowsStatementHandler creates a new instance of BatchSQLRowsStatementHandler
func NewBatchSQLRowsStatementHandler(driver driver.Driver, session session.Session, middlewares ...Middleware) StatementHandler {
	return &BatchSQLRowsStatementHandler{
		driver:      driver,
		middlewares: middlewares,
		session:     session,
	}
}

// DefaultStatementHandler returns a new instance of StatementHandler with the default behavior.
func DefaultStatementHandler(driver driver.Driver, session session.Session, middlewares ...Middleware) StatementHandler {
	return NewBatchSQLRowsStatementHandler(driver, session, middlewares...)
}

// SQLRowsStatementHandler handles the execution of SQL statements and returns
// the results in a sql.Rows structure. It integrates a driver, middlewares, and
// a session to manage the execution flow.
type SQLRowsStatementHandler struct {
	driver      driver.Driver
	middlewares MiddlewareGroup
	session     session.Session
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *SQLRowsStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	query, args, err := statement.Build(s.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	contextReducer := ctxreducer.G{
		ctxreducer.NewSessionContextReducer(s.session),
		ctxreducer.NewParamContextReducer(param),
	}
	ctx = contextReducer.Reduce(ctx)
	queryHandler := CombineQueryHandler(statement, s.middlewares...)
	return queryHandler(ctx, query, args...)
}

// ExecContext executes a non-query SQL statement (such as INSERT, UPDATE, DELETE)
// within a context, and returns the result. Similar to QueryContext, it constructs
// the SQL command, applies middlewares, and executes the command using the driver.
func (s *SQLRowsStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	query, args, err := statement.Build(s.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	contextReducer := ctxreducer.G{
		ctxreducer.NewSessionContextReducer(s.session),
		ctxreducer.NewParamContextReducer(param),
	}
	ctx = contextReducer.Reduce(ctx)
	execHandler := CombineExecHandler(statement, s.middlewares...)
	return execHandler(ctx, query, args...)
}

var _ StatementHandler = (*SQLRowsStatementHandler)(nil)

// NewSQLRowsStatementHandler creates a new instance of SQLRowsStatementHandler
// with the provided driver, session, and an optional list of middlewares. This
// function is typically used to initialize the handler before executing SQL statements.
func NewSQLRowsStatementHandler(driver driver.Driver, session session.Session, middlewares ...Middleware) StatementHandler {
	return &SQLRowsStatementHandler{
		driver:      driver,
		middlewares: middlewares,
		session:     session,
	}
}
