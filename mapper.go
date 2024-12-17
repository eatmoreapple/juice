package juice

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/eatmoreapple/juice/internal/container"
)

// Mapper defines a set of statements.
type Mapper struct {
	namespace  string
	mappers    *Mappers
	statements map[string]*xmlSQLStatement
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

// Attribute returns the attribute value by key.
func (m *Mapper) Attribute(key string) string {
	return m.attrs[key]
}

// Prefix returns the prefix of the mapper.
func (m *Mapper) Prefix() string {
	return m.Attribute("prefix")
}

func (m *Mapper) GetSQLNodeByID(id string) (Node, error) {
	// first, try to get sql node from current namespace.
	node, exists := m.sqlNodes[id]
	if !exists {
		// if not exists, try to get sql node from other namespace.
		return m.mappers.GetSQLNodeByID(id)
	}
	return node, nil
}

// ErrSQLNodeNotFound indicates that the SQL node was not found in the mapper
type ErrSQLNodeNotFound struct {
	NodeName   string
	MapperName string
}

func (e ErrSQLNodeNotFound) Error() string {
	return fmt.Sprintf("SQL node %q not found in mapper %q", e.NodeName, e.MapperName)
}

// Mappers is a container for all mappers.
type Mappers struct {
	attrs map[string]string
	cfg   IConfiguration
	// mappers uses Trie instead of map because mapper namespaces often share common prefixes
	// (e.g., "com.example.user", "com.example.order"). Trie provides both memory efficiency
	// by storing shared prefixes only once and fast prefix-based lookups
	mappers *container.Trie[*Mapper]
}

func (m *Mappers) setMapper(key string, mapper *Mapper) error {
	if prefix := m.Prefix(); prefix != "" {
		key = fmt.Sprintf("%s.%s", prefix, key)
	}
	if m.mappers == nil {
		m.mappers = container.NewTrie[*Mapper]()
	}
	if _, exists := m.mappers.Get(key); exists {
		return fmt.Errorf("mapper %s already exists", key)
	}
	mapper.mappers = m
	m.mappers.Insert(key, mapper)
	return nil
}

func (m *Mappers) GetMapperByNamespace(namespace string) (*Mapper, bool) {
	if m.mappers == nil {
		return nil, false
	}
	return m.mappers.Get(namespace)
}

func (m *Mappers) getMapperAndKey(id string) (mapper *Mapper, key string, err error) {
	lastDotIndex := strings.LastIndex(id, ".")
	if lastDotIndex <= 0 {
		return nil, "", ErrInvalidStatementID
	}

	namespace, key := id[:lastDotIndex], id[lastDotIndex+1:]
	mapper, exists := m.GetMapperByNamespace(namespace)
	if !exists {
		return nil, "", ErrMapperNotFound(namespace)
	}
	return mapper, key, nil
}

// GetStatementByID returns a Statement by id.
// The id should be in the format of "namespace.statementName"
// For example: "main.UserMapper.SelectUser"
func (m *Mappers) GetStatementByID(id string) (Statement, error) {
	mapper, key, err := m.getMapperAndKey(id)
	if err != nil {
		return nil, err
	}

	stmt, exists := mapper.statements[key]
	if !exists {
		return nil, &ErrStatementNotFound{StatementName: key, MapperName: mapper.namespace}
	}
	return stmt, nil
}

func (m *Mappers) GetSQLNodeByID(id string) (Node, error) {
	mapper, key, err := m.getMapperAndKey(id)
	if err != nil {
		return nil, err
	}

	node, exists := mapper.sqlNodes[key]
	if !exists {
		return nil, &ErrSQLNodeNotFound{NodeName: key, MapperName: mapper.namespace}
	}
	return node, nil
}

// GetStatement try to one the xmlSQLStatement from the Mappers with the given interface
func (m *Mappers) GetStatement(v any) (Statement, error) {
	var id string
	// if the interface is StatementIDGetter, use the StatementID() method to get the id
	// or if the interface is a string type, use the string as the id
	// otherwise, use the reflection to get the id
	switch t := v.(type) {
	case interface{ StatementID() string }:
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
			id = rv.Type().PkgPath() + "." + rv.Type().Name()
		default:
			return nil, errors.New("invalid type of xmlSQLStatement id")
		}
	}
	if len(id) == 0 {
		return nil, errors.New("can not get the xmlSQLStatement id from the given interface")
	}
	return m.GetStatementByID(id)
}

// Configuration represents a configuration of juice.
func (m *Mappers) Configuration() IConfiguration {
	return m.cfg
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
