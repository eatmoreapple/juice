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

func (m *Mapper) GetSQLNodeByID(id string) (Node, error) {
	// first, try to get sql node from current namespace.
	node, exists := m.sqlNodes[id]
	if !exists {
		// if not exists, try to get sql node from other namespace.
		return m.getSQLNodeFromNamespace(id)
	}
	return node, nil
}

// getSQLNodeFromNamespace gets sql node from other namespace.
func (m *Mapper) getSQLNodeFromNamespace(id string) (Node, error) {
	if m.mappers == nil {
		return nil, errors.New("mappers is nil")
	}
	items := strings.Split(id, ".")
	if len(items) == 1 {
		return nil, &sqlNodeNotFoundError{id}
	}
	namespace, pk := strings.Join(items[:len(items)-1], "."), items[len(items)-1]
	mapper, exists := m.mappers.GetMapperByNamespace(namespace)
	if !exists {
		return nil, &sqlNodeNotFoundError{id}
	}
	node, exists := mapper.sqlNodes[pk]
	if !exists {
		return nil, &sqlNodeNotFoundError{id}
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

func (m *Mapper) Engine() *Engine {
	return m.Configuration().engine
}

// checkResultMap checks statement's resultMap attribute is valid or not.
func (m *Mapper) checkResultMap() error {
	for _, stmt := range m.statements {
		resultMapName := stmt.attrs["resultMap"]
		if resultMapName == "" {
			continue
		}
		if _, err := m.GetResultMapByID(resultMapName); err != nil {
			return err
		}
	}
	return nil
}

// Mappers is a map of mappers.
type Mappers struct {
	mappers map[string]*Mapper
	attrs   map[string]string
	cfg     *Configuration
}

func (m *Mappers) setMapper(key string, mapper *Mapper) error {
	if _, exists := m.mappers[key]; exists {
		return fmt.Errorf("mapper %s already exists", key)
	}
	if m.mappers == nil {
		m.mappers = make(map[string]*Mapper)
	}
	m.mappers[key] = mapper
	mapper.mappers = m
	return nil
}

func (m *Mappers) GetMapperByNamespace(namespace string) (*Mapper, bool) {
	mapper, exists := m.mappers[namespace]
	return mapper, exists
}

// GetStatementByID returns a statement by id.
// If the statement is not found, an error is returned.
func (m *Mappers) GetStatementByID(id string) (*Statement, error) {
	items := strings.Split(id, ".")
	if len(items) == 1 {
		return nil, fmt.Errorf("invalid statement id: %s", id)
	}
	// get the namespace and pk
	// main.UserMapper.SelectUser => main.UserMapper, SelectUser
	namespace, pk := strings.Join(items[:len(items)-1], "."), items[len(items)-1]
	mapper, exists := m.mappers[namespace]
	if !exists {
		return nil, fmt.Errorf("mapper `%s` not found", namespace)
	}
	stmt, exists := mapper.statements[pk]
	if !exists {
		return nil, fmt.Errorf("statement `%s` not found", id)
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
