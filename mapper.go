package juice

import (
	"errors"
	"fmt"
	"reflect"
)

// Mapper defines a set of statements.
type Mapper struct {
	mappers    *Mappers
	namespace  string
	resource   string
	url        string
	statements map[string]*Statement
	sqlNodes   map[string]*SQLNode
	attrs      map[string]string
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

func (m *Mapper) Attribute(key string) string {
	return m.attrs[key]
}

func (m *Mapper) GetSQLNodeByID(id string) (Node, error) {
	node, exists := m.sqlNodes[id]
	if !exists {
		return nil, errors.New("sql node not found")
	}
	return node, nil
}

func (m *Mapper) Configuration() *Configuration {
	return m.mappers.Configuration()
}

// Mappers is a map of mappers.
type Mappers struct {
	statements map[string]*Statement
	cfg        *Configuration
}

// GetStatementByID returns a statement by id.
// If the statement is not found, an error is returned.
func (m *Mappers) GetStatementByID(id string) (*Statement, error) {
	stmt, exists := m.statements[id]
	if !exists {
		return nil, errors.New("statement not found")
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
	return nil
}

// StatementIDGetter is an interface for getting statement id.
type StatementIDGetter interface {
	// StatementID returns a statement id.
	StatementID() string
}
