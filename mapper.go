package pillow

import (
	"errors"
)

// Mapper defines a set of statements.
type Mapper struct {
	namespace  string
	resource   string
	url        string
	statements map[string]Statement
}

// Namespace returns the namespace of the mapper.
func (m Mapper) Namespace() string {
	return m.namespace
}

// Mappers is a map of mappers.
type Mappers map[string]Statement

// GetStatementByID returns a statement by id.
// If the statement is not found, an error is returned.
func (m Mappers) GetStatementByID(id string) (Statement, error) {
	stmt, exists := m[id]
	if !exists {
		return nil, errors.New("statement not found")
	}
	return stmt, nil
}
