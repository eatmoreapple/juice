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
	if len(p) > 0 {
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
	} else {
		query = string(c)
	}
	return query, args, err
}

type ConditionNode struct {
	testExpr ast.Expr
	Nodes    []Node
}

// Parse with given expression.
func (c *ConditionNode) Parse(test string) (err error) {
	c.testExpr, err = parser.ParseExpr(test)
	if err != nil {
		return &SyntaxError{err}
	}
	return nil
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *ConditionNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
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
			if len(a) > 0 {
				args = append(args, a...)
			}
		}
		return builder.String(), args, nil
	}
	return "", nil, err
}

// Match returns true if test is matched.
func (c *ConditionNode) Match(p Param) (bool, error) {
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

var _ Node = (*IfNode)(nil)

// IfNode is a node of if.
type IfNode = ConditionNode

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
		} else if strings.HasPrefix(query, "or") || strings.HasPrefix(query, "OR") {
			query = query[2:]
		}
	}

	query = "WHERE" + query

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
	if t.Prefix != "" {
		builder.WriteString(t.Prefix)
	}
	for i, node := range t.Nodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		builder.WriteString(q)
		if !strings.HasSuffix(q, " ") && i < len(t.Nodes)-1 {
			builder.WriteString(" ")
		}
		args = append(args, a...)
	}
	query = builder.String()
	if t.PrefixOverrides != "" {
		query = strings.TrimPrefix(query, t.PrefixOverrides)
	}
	if t.SuffixOverrides != "" {
		query = strings.TrimSuffix(query, t.SuffixOverrides)
	}
	if t.Suffix != "" {
		query += t.Suffix
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

// SetNode is a node of set.
type SetNode struct {
	Nodes []Node
}

// Accept accepts parameters and returns query and arguments.
func (s SetNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	var builder = getBuilder()
	defer putBuilder(builder)
	for i, node := range s.Nodes {
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
		if i < len(s.Nodes)-1 && len(q) > 0 && !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
	}
	query = builder.String()
	if query != "" {
		query = "SET " + query
	}
	if strings.HasSuffix(query, ",") {
		query = query[:len(query)-1]
	}
	return query, args, nil
}

// SQLNode is a node of sql.
// SQLNode defines a SQL query.
type SQLNode struct {
	id     string
	nodes  []Node
	mapper *Mapper
}

// ID returns the id of the node.
func (s SQLNode) ID() string {
	return s.id
}

// Accept accepts parameters and returns query and arguments.
func (s SQLNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	var builder = getBuilder()
	defer putBuilder(builder)
	for i, node := range s.nodes {
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
		if i < len(s.nodes)-1 && len(q) > 0 && !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
	}
	return builder.String(), args, nil
}

// IncludeNode is a node of include.
// It includes another SQL.
type IncludeNode struct {
	RefId  string
	mapper *Mapper
	nodes  []Node
}

// Accept accepts parameters and returns query and arguments.
func (i IncludeNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	sqlNode, err := i.mapper.GetSQLNodeByID(i.RefId)
	if err != nil {
		return "", nil, fmt.Errorf("sql node %s not found", i.RefId)
	}
	return sqlNode.Accept(translator, p)
}

// ChooseNode is a node of choose.
// ChooseNode can have multiple when nodes and one otherwise node.
// WhenNode is executed when test is true.
// OtherwiseNode is executed when all when nodes are false.
type ChooseNode struct {
	WhenNodes     []Node
	OtherwiseNode Node
}

// Accept accepts parameters and returns query and arguments.
func (c ChooseNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	for _, node := range c.WhenNodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		// if one of when nodes is true, return query and arguments
		if len(q) > 0 {
			return q, a, nil
		}
	}
	// if all when nodes are false, return otherwise node
	if c.OtherwiseNode != nil {
		return c.OtherwiseNode.Accept(translator, p)
	}
	return "", nil, nil
}

// WhenNode is a node of when.
// WhenNode like if node, but it can not be used alone.
// While one of WhenNode is true, the query of ChooseNode will be returned.
type WhenNode = ConditionNode

// OtherwiseNode is a node of otherwise.
// OtherwiseNode like else node, but it can not be used alone.
// If all WhenNode is false, the query of OtherwiseNode will be returned.
type OtherwiseNode struct {
	Nodes []Node
}

// Accept accepts parameters and returns query and arguments.
func (o OtherwiseNode) Accept(translator driver.Translator, p Param) (query string, args []interface{}, err error) {
	var builder = getBuilder()
	defer putBuilder(builder)
	for i, node := range o.Nodes {
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
		if i < len(o.Nodes)-1 && len(q) > 0 && !strings.HasSuffix(q, " ") {
			builder.WriteString(" ")
		}
	}
	return builder.String(), args, nil
}
