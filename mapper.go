package juice

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Mapper defines a set of statements.
type Mapper struct {
	namespace  string
	mappers    *Mappers
	statements map[string]*Statement
	sqlNodes   map[string]*SQLNode
	attrs      map[string]string
	resultMaps map[string]*resultMapNode
}

// Namespace returns the namespace of the mapper.
func (m *Mapper) Namespace() string {
	return m.namespace
}

// Mappers is an getter of mappers.
func (m *Mapper) Mappers() *Mappers {
	return m.mappers
}

func (m *Mapper) setAttribute(key, value string) {
	if m.attrs == nil {
		m.attrs = make(map[string]string)
	}
	m.attrs[key] = value
}

func (m *Mapper) setSqlNode(node *SQLNode) error {
	if m.sqlNodes == nil {
		m.sqlNodes = make(map[string]*SQLNode)
	}
	if _, exists := m.sqlNodes[node.ID()]; exists {
		return fmt.Errorf("sql node %s already exists", node.ID())
	}
	m.sqlNodes[node.ID()] = node
	return nil
}

func (m *Mapper) setResultMap(node *resultMapNode) error {
	if m.resultMaps == nil {
		m.resultMaps = make(map[string]*resultMapNode)
	}
	if _, exists := m.resultMaps[node.ID()]; exists {
		return fmt.Errorf("result map %s already exists", node.ID())
	}
	m.resultMaps[node.ID()] = node
	return nil
}

// Attribute returns the attribute value by key.
func (m *Mapper) Attribute(key string) string {
	return m.attrs[key]
}

// Prefix returns the prefix of the mapper.
func (m *Mapper) Prefix() string {
	return m.Attribute("prefix")
}

// name is the name of the mapper. which is the unique key of the mapper.
func (m *Mapper) name() string {
	var builder strings.Builder
	if prefix := m.mappers.Prefix(); prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString(".")
	}
	if prefix := m.Prefix(); prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString(".")
	}
	builder.WriteString(m.Namespace())
	return builder.String()
}

func (m *Mapper) GetSQLNodeByID(id string) (Node, error) {
	node, exists := m.sqlNodes[id]
	if !exists {
		return nil, errors.New("sql node not found")
	}
	return node, nil
}

func (m *Mapper) GetResultMapByID(id string) (ResultMap, error) {
	retMap, exists := m.resultMaps[id]
	if !exists {
		return nil, fmt.Errorf("result map %s not found", id)
	}
	return retMap, nil
}

func (m *Mapper) Configuration() *Configuration {
	return m.mappers.Configuration()
}

// Mappers is a map of mappers.
type Mappers struct {
	statements map[string]*Statement
	cfg        *Configuration
	attrs      map[string]string
}

// GetStatementByID returns a statement by id.
// If the statement is not found, an error is returned.
func (m *Mappers) GetStatementByID(id string) (*Statement, error) {
	stmt, exists := m.statements[id]
	if !exists {
		return nil, fmt.Errorf("statement %s not found", id)
	}
	return stmt, nil
}

// GetStatement try to one the statement from the Mappers with the given interface
func (m *Mappers) GetStatement(v any) (*Statement, error) {
	var id string
	// if the interface is StatementIDGetter, use the StatementID() method to get the id
	// or if the interface is a string type, use the string as the id
	// otherwise, use the reflection to get the id
	switch t := v.(type) {
	case StatementIDGetter:
		id = t.StatementID()
	case string:
		id = t
	default:
		// else try to one the id from the interface
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch rv.Kind() {
		case reflect.Func:
			id = runtimeFuncName(rv)
		case reflect.Struct:
			id = rv.Type().Name()
		default:
			return nil, errors.New("invalid type of statement id")
		}
	}
	if len(id) == 0 {
		return nil, errors.New("can not get the statement id from the given interface")
	}
	return m.GetStatementByID(id)
}

// Configuration represents a configuration of juice.
func (m *Mappers) Configuration() *Configuration {
	return m.cfg
}

// setStatementByID sets a statement by id.
func (m *Mappers) setStatementByID(id string, stmt *Statement) error {
	if m.statements == nil {
		m.statements = make(map[string]*Statement)
	}
	if _, exists := m.statements[id]; exists {
		return fmt.Errorf("statement %s already exists", id)
	}
	m.statements[id] = stmt
	// set the unique name into this statement
	stmt.name = id
	return nil
}

// setAttribute sets an attribute.
// same as setAttribute, but it is used for Mappers.
func (m *Mappers) setAttribute(key, value string) {
	if m.attrs == nil {
		m.attrs = make(map[string]string)
	}
	m.attrs[key] = value
}

// Attribute returns an attribute from the Mappers attributes.
func (m *Mappers) Attribute(key string) string {
	return m.attrs[key]
}

// Prefix returns the prefix of the Mappers.
func (m *Mappers) Prefix() string {
	return m.Attribute("prefix")
}

// StatementIDGetter is an interface for getting statement id.
type StatementIDGetter interface {
	// StatementID returns a statement id.
	StatementID() string
}
