package juice

import (
	"errors"
	"fmt"
)

// Mapper defines a set of statements.
type Mapper struct {
	mappers    *Mappers
	namespace  string
	resource   string
	url        string
	statements map[string]*Statement
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

func (m *Mapper) Attribute(key string) string {
	return m.attrs[key]
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
