package juice

import (
	"fmt"
	"go/ast"
	"go/parser"
	"reflect"
	"regexp"
	"strings"

	"github.com/eatmoreapple/juice/driver"
)

// paramRegex is a regular expression for parameter.
var paramRegex = regexp.MustCompile(`\#\{([a-zA-Z0-9_\.]+)\}`)

// Node is a node of SQL.
type Node interface {
	// Accept accepts parameters and returns query and arguments.
	Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error)
}

var _ Node = (*TextNode)(nil)

// TextNode is a node of text.
type TextNode string

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c TextNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	query = paramRegex.ReplaceAllStringFunc(string(c), func(s string) string {
		if err != nil {
			return s
		}
		param := paramRegex.FindStringSubmatch(s)[1]

		value, exists := p.Get(param)
		if !exists {
			err = fmt.Errorf("parameter %s not found", param)
			return s
		}
		args = append(args, value.Interface())
		return translator.Translate(s)
	})
	return query, args, err
}

var _ Node = (*IfNode)(nil)

// IfNode is a node of if.
type IfNode struct {
	Test     string
	testExpr ast.Expr
	Nodes    []Node
}

func (c *IfNode) init() (err error) {
	c.testExpr, err = parser.ParseExpr(c.Test)
	if err != nil {
		return &SyntaxError{err}
	}
	return nil
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *IfNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	matched, err := c.Match(p)
	if err != nil {
		return "", nil, err
	}
	if matched {
		var builder = getBuilder()
		defer putBuilder(builder)
		for _, node := range c.Nodes {
			q, a, err := node.Accept(translator, p)
			if err != nil {
				return "", nil, err
			}
			builder.WriteString(q)
			args = append(args, a...)
		}
		return builder.String(), args, nil
	}
	return "", nil, err
}

// Match returns true if test is matched.
func (c *IfNode) Match(p Param) (bool, error) {
	value, err := eval(c.testExpr, p)
	if err != nil {
		return false, err
	}
	switch value.Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() != 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint() != 0, nil
	case reflect.Float32, reflect.Float64:
		return value.Float() != 0, nil
	case reflect.String:
		return value.String() != "", nil
	default:
		return false, fmt.Errorf("unsupported type %s", value.Kind())
	}
}

var _ Node = (*WhereNode)(nil)

// WhereNode is a node of where.
type WhereNode struct {
	Nodes []Node
}

// Accept accepts parameters and returns query and arguments.
func (w WhereNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	var builder = getBuilder()
	defer putBuilder(builder)
	for i, node := range w.Nodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		if len(q) > 0 {
			builder.WriteString(q)
		}
		if len(a) > 0 {
			args = append(args, a...)
		}
		if i < len(w.Nodes)-1 && len(q) > 0 && !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
	}
	query = builder.String()
	if query != "" {
		if strings.HasPrefix(query, "and") || strings.HasPrefix(query, "AND") {
			query = query[3:]
			query = "WHERE" + query
		} else if strings.HasPrefix(query, "or") || strings.HasPrefix(query, "OR") {
			query = query[2:]
			query = "WHERE" + query
		}
	}

	if !(strings.HasPrefix(query, "where") || strings.HasPrefix(query, "WHERE")) {
		query = "WHERE " + query
	}

	return query, args, nil
}

var _ Node = (*TrimNode)(nil)

// TrimNode is a node of trim.
type TrimNode struct {
	Nodes           []Node
	Prefix          string
	PrefixOverrides string
	Suffix          string
	SuffixOverrides string
}

// Accept accepts parameters and returns query and arguments.
func (t TrimNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	var builder = getBuilder()
	defer putBuilder(builder)
	for _, node := range t.Nodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		builder.WriteString(q)
		if !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
		args = append(args, a...)
	}
	query = builder.String()
	if t.Prefix != "" {
		query = strings.TrimPrefix(query, t.Prefix)
	}
	if t.PrefixOverrides != "" {
		query = strings.TrimPrefix(query, t.PrefixOverrides)
	}
	if t.Suffix != "" {
		query = strings.TrimSuffix(query, t.Suffix)
	}
	if t.SuffixOverrides != "" {
		query = strings.TrimSuffix(query, t.SuffixOverrides)
	}
	return query, args, nil
}

var _ Node = (*ForeachNode)(nil)

// ForeachNode is a node of foreach.
type ForeachNode struct {
	Collection string
	Nodes      []Node
	Item       string
	Index      string
	Open       string
	Close      string
	Separator  string
}

// Accept accepts parameters and returns query and arguments.
func (f ForeachNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {

	// if item already exists
	if _, exists := p.Get(f.Item); exists {
		return "", nil, fmt.Errorf("item %s already exists", f.Item)
	}

	// get collection from parameter
	value, exists := p.Get(f.Collection)
	if !exists {
		return "", nil, fmt.Errorf("collection %s not found", f.Collection)
	}

	// if value can not be iterated
	if !value.CanInterface() {
		return "", nil, fmt.Errorf("collection %s can not be iterated", f.Collection)
	}

	// if value is not a slice
	for value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	if value.Kind() != reflect.Slice {
		return "", nil, fmt.Errorf("collection %s is not a slice", f.Collection)
	}

	var builder = getBuilder()
	defer putBuilder(builder)

	length := value.Len()

	// if length is not zero, add open
	if length > 0 {
		builder.WriteString(f.Open)
	}

	end := length - 1

	for i := 0; i < length; i++ {

		// set or replace item
		p[f.Item] = reflect.Indirect(value.Index(i))

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, p)
			if err != nil {
				return "", nil, err
			}
			if len(q) > 0 {
				builder.WriteString(q)
			}
			if len(a) > 0 {
				args = append(args, a...)
			}
		}

		if i < end {
			builder.WriteString(f.Separator)
		}
	}

	// delete item from parameter
	delete(p, f.Item)

	// if length is not zero, add close
	if length > 0 {
		builder.WriteString(f.Close)
	}

	return builder.String(), args, nil
}
