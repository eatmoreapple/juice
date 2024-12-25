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
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/eatmoreapple/juice/eval"

	"github.com/eatmoreapple/juice/driver"
)

var (
	// paramRegex matches parameter placeholders in SQL queries using #{...} syntax.
	// Examples:
	//   - #{id}         -> matches
	//   - #{user.name}  -> matches
	//   - #{  age  }    -> matches (whitespace is ignored)
	//   - #{}           -> doesn't match (requires identifier)
	//   - #{123}        -> matches
	paramRegex = regexp.MustCompile(`#{\s*(\w+(?:\.\w+)*)\s*}`)

	// formatRegexp matches string interpolation placeholders using ${...} syntax.
	// Unlike paramRegex, these are replaced directly in the SQL string.
	// WARNING: Be careful with this as it can lead to SQL injection if not properly sanitized.
	// Examples:
	//   - ${tableName}  -> matches
	//   - ${db.schema}  -> matches
	//   - ${  field  }  -> matches (whitespace is ignored)
	//   - ${}           -> doesn't match (requires identifier)
	//   - ${123}        -> matches
	formatRegexp = regexp.MustCompile(`\${\s*(\w+(?:\.\w+)*)\s*}`)
)

// Node is the fundamental interface for all SQL generation components.
// It defines the contract for converting dynamic SQL structures into
// concrete SQL queries with their corresponding parameters.
//
// The Accept method follows the Visitor pattern, allowing different
// SQL dialects to be supported through the translator parameter.
//
// Parameters:
//   - translator: Handles dialect-specific SQL translations
//   - p: Contains parameter values for SQL generation
//
// Returns:
//   - query: The generated SQL fragment
//   - args: Slice of arguments for prepared statement
//   - err: Any error during SQL generation
//
// Implementing types include:
//   - SQLNode: Complete SQL statements
//   - WhereNode: WHERE clause handling
//   - SetNode: SET clause for updates
//   - IfNode: Conditional inclusion
//   - ChooseNode: Switch-like conditionals
//   - ForeachNode: Collection iteration
//   - TrimNode: String manipulation
//   - IncludeNode: SQL fragment reuse
//
// Example usage:
//
//	query, args, err := node.Accept(mysqlTranslator, params)
//	if err != nil {
//	  // handle error
//	}
//	// use query and args with database
//
// Note: Implementations should handle their specific SQL generation
// logic while maintaining consistency with the overall SQL structure.
type Node interface {
	// Accept processes the node with given translator and parameters
	// to produce a SQL fragment and its arguments.
	Accept(translator driver.Translator, p Parameter) (query string, args []any, err error)
}

// NodeGroup wraps multiple nodes into a single node.
type NodeGroup []Node

// Accept processes all nodes in the group and combines their results.
// The method ensures proper spacing between node outputs and trims any extra whitespace.
// If the group is empty or no nodes produce output, it returns empty results.
func (g NodeGroup) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	// Return early if group is empty
	if len(g) == 0 {
		return "", nil, nil
	}

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	// Pre-allocate args slice to avoid reallocations
	args = make([]any, 0, len(g))

	lastIdx := len(g) - 1

	// Process each node in the group
	for i, node := range g {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		if len(q) > 0 {
			builder.WriteString(q)

			// Add space between nodes, but not after the last one
			if i < lastIdx && !strings.HasSuffix(q, " ") {
				builder.WriteString(" ")
			}
		}
		if len(a) > 0 {
			args = append(args, a...)
		}
	}

	// Return empty results if no content was generated
	if builder.Len() == 0 {
		return "", nil, nil
	}

	return strings.TrimSpace(builder.String()), args, nil
}

// pureTextNode is a node of pure text.
var _ Node = (*pureTextNode)(nil)

// pureTextNode is a node of pure text.
// It is used to avoid unnecessary parameter replacement.
type pureTextNode string

func (p pureTextNode) Accept(_ driver.Translator, _ Parameter) (query string, args []any, err error) {
	return string(p), nil, nil
}

var _ Node = (*TextNode)(nil)

// TextNode is a node of text.
// What is the difference between TextNode and pureTextNode?
// TextNode is used to replace parameters with placeholders.
// pureTextNode is used to avoid unnecessary parameter replacement.
type TextNode struct {
	value            string
	placeholder      [][]string // for example, #{id}
	textSubstitution [][]string // for example, ${id}
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *TextNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	// If there is no parameter, return the value as it is.
	if len(c.placeholder) == 0 && len(c.textSubstitution) == 0 {
		return c.value, nil, nil
	}
	// Otherwise, replace the parameter with a placeholder.
	query, args, err = c.replaceHolder(c.value, args, translator, p)
	if err != nil {
		return "", nil, err
	}
	query, err = c.replaceTextSubstitution(query, p)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

func (c *TextNode) replaceHolder(query string, args []interface{}, translator driver.Translator, p Parameter) (string, []any, error) {
	for _, param := range c.placeholder {
		if len(param) != 2 {
			return "", nil, fmt.Errorf("invalid parameter %v", param)
		}
		matched, name := param[0], param[1]

		// try to get value from parameter
		value, exists := p.Get(name)
		if !exists {
			return "", nil, fmt.Errorf("parameter %s not found", name)
		}
		query = strings.Replace(query, matched, translator.Translate(name), 1)
		args = append(args, value.Interface())
	}
	return query, args, nil
}

// replaceTextSubstitution replaces text substitution.
func (c *TextNode) replaceTextSubstitution(query string, p Parameter) (string, error) {
	for _, sub := range c.textSubstitution {
		if len(sub) != 2 {
			return "", fmt.Errorf("invalid text substitution %v", sub)
		}
		matched, name := sub[0], sub[1]
		value, exists := p.Get(name)
		if !exists {
			return "", fmt.Errorf("parameter %s not found", name)
		}
		query = strings.Replace(query, matched, reflectValueToString(value), 1)
	}
	return query, nil
}

// build builds TextNode.
func (c *TextNode) build() {
	placeholder := paramRegex.FindAllStringSubmatch(c.value, -1)
	if len(placeholder) > 0 {
		c.placeholder = placeholder
	}
	textSubstitution := formatRegexp.FindAllStringSubmatch(c.value, -1)
	if len(textSubstitution) > 0 {
		c.textSubstitution = textSubstitution
	}
}

// NewTextNode creates a new text node based on the input string.
// It returns either a lightweight pureTextNode for static SQL,
// or a full TextNode for dynamic SQL with placeholders/substitutions.
func NewTextNode(str string) Node {
	var node = &TextNode{value: str}
	node.build()
	if len(node.placeholder) == 0 && len(node.textSubstitution) == 0 {
		return pureTextNode(str)
	}
	return node
}

// ConditionNode represents a conditional SQL fragment with its evaluation expression and child nodes.
// It is used to conditionally include or exclude SQL fragments based on runtime parameters.
type ConditionNode struct {
	expr  eval.Expression
	Nodes NodeGroup
}

// Parse compiles the given expression string into an evaluable expression.
// The expression syntax supports various operations like:
//   - Comparison: ==, !=, >, <, >=, <=
//   - Logical: &&, ||, !
//   - Null checks: != null, == null
//   - Property access: user.age, order.status
//
// Examples:
//
//	"id != nil"              // Check for non-null
//	"age >= 18"               // Numeric comparison
//	"status == "ACTIVE""      // String comparison
//	"user.role == "ADMIN""    // Property access
func (c *ConditionNode) Parse(test string) (err error) {
	c.expr, err = eval.Compile(test)
	return err
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *ConditionNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	matched, err := c.Match(p)
	if err != nil {
		return "", nil, err
	}
	if !matched {
		return "", nil, nil
	}
	return c.Nodes.Accept(translator, p)
}

// Match evaluates if the condition is true based on the provided parameter.
// It handles different types of values and converts them to boolean results:
//   - Bool: returns the boolean value directly
//   - Integers (signed/unsigned): returns true if non-zero
//   - Floats: returns true if non-zero
//   - String: returns true if non-empty
func (c *ConditionNode) Match(p Parameter) (bool, error) {
	value, err := c.expr.Execute(p)
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

var _ Node = (*ConditionNode)(nil)

// IfNode is an alias for ConditionNode, representing a conditional SQL fragment.
// It evaluates a condition and determines whether its content should be included in the final SQL.
//
// The condition can be based on various types:
//   - Boolean: direct condition
//   - Numbers: non-zero values are true
//   - Strings: non-empty strings are true
//
// Example usage:
//
//	<if test="id > 0">
//	    AND id = #{id}
//	</if>
//
// See ConditionNode for detailed behavior of condition evaluation.
type IfNode = ConditionNode

var _ Node = (*IfNode)(nil)

// WhereNode represents a SQL WHERE clause and its conditions.
// It manages a group of condition nodes that form the complete WHERE clause.
type WhereNode struct {
	Nodes NodeGroup
}

// Accept processes the WHERE clause and its conditions.
// It handles several special cases:
//  1. Removes leading "AND" or "OR" from the first condition
//  2. Ensures the clause starts with "WHERE" if not already present
//  3. Properly handles spacing between conditions
//
// Examples:
//
//	Input:  "AND id = ?"        -> Output: "WHERE id = ?"
//	Input:  "OR name = ?"       -> Output: "WHERE name = ?"
//	Input:  "WHERE age > ?"     -> Output: "WHERE age > ?"
//	Input:  "status = ?"        -> Output: "WHERE status = ?"
func (w WhereNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	query, args, err = w.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}

	// A space is required at the end; otherwise, it is meaningless.
	switch {
	case strings.HasPrefix(query, "and ") || strings.HasPrefix(query, "AND "):
		query = query[4:]
	case strings.HasPrefix(query, "or ") || strings.HasPrefix(query, "OR "):
		query = query[3:]
	}

	// A space is required at the end; otherwise, it is meaningless.
	if !(strings.HasPrefix(query, "where ") || strings.HasPrefix(query, "WHERE ")) {
		query = "WHERE " + query
	}
	return
}

var _ Node = (*WhereNode)(nil)

// TrimNode handles SQL fragment cleanup by managing prefixes, suffixes, and their overrides.
// It's particularly useful for dynamically generated SQL where certain prefixes or suffixes
// might need to be added or removed based on the context.
//
// Fields:
//   - Nodes: Group of child nodes containing the SQL fragments
//   - Prefix: String to prepend to the result if content exists
//   - PrefixOverrides: Strings to remove if found at the start
//   - Suffix: String to append to the result if content exists
//   - SuffixOverrides: Strings to remove if found at the end
//
// Common use cases:
//  1. Removing leading AND/OR from WHERE clauses
//  2. Managing commas in clauses
//  3. Handling dynamic UPDATE SET statements
//
// Example XML:
//
//	<trim prefix="WHERE" prefixOverrides="AND|OR">
//	  <if test="id > 0">
//	    AND id = #{id}
//	  </if>
//	  <if test='name != ""'>
//	    AND name = #{name}
//	  </if>
//	</trim>
//
// Example Result:
//
//	Input:  "AND id = ? AND name = ?"
//	Output: "WHERE id = ? AND name = ?"
type TrimNode struct {
	Nodes           NodeGroup
	Prefix          string
	PrefixOverrides []string
	Suffix          string
	SuffixOverrides []string
}

// Accept accepts parameters and returns query and arguments.
func (t TrimNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	query, args, err = t.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}

	if len(query) == 0 {
		return "", nil, nil
	}

	// Handle prefix overrides before adding prefix
	if len(t.PrefixOverrides) > 0 {
		for _, prefix := range t.PrefixOverrides {
			if strings.HasPrefix(query, prefix) {
				query = query[len(prefix):]
				break
			}
		}
	}

	// Handle suffix overrides before adding suffix
	if len(t.SuffixOverrides) > 0 {
		for _, suffix := range t.SuffixOverrides {
			if strings.HasSuffix(query, suffix) {
				query = query[:len(query)-len(suffix)]
				break
			}
		}
	}

	// Build final query with prefix and suffix
	var builder strings.Builder
	builder.Grow(len(t.Prefix) + len(query) + len(t.Suffix))

	if t.Prefix != "" {
		builder.WriteString(t.Prefix)
	}
	builder.WriteString(query)
	if t.Suffix != "" {
		builder.WriteString(t.Suffix)
	}

	return builder.String(), args, nil
}

var _ Node = (*TrimNode)(nil)

// ForeachNode represents a dynamic SQL fragment that iterates over a collection.
// It's commonly used for IN clauses, batch inserts, or any scenario requiring
// iteration over a collection of values in SQL generation.
//
// Fields:
//   - Collection: Expression to get the collection to iterate over
//   - Nodes: SQL fragments to be repeated for each item
//   - Item: Variable name for the current item in iteration
//   - Index: Variable name for the current index (optional)
//   - Open: String to prepend before the iteration results
//   - Close: String to append after the iteration results
//   - Separator: String to insert between iterations
//
// Example XML:
//
//	<foreach collection="list" item="item" index="i" open="(" separator="," close=")">
//	  #{item}
//	</foreach>
//
// Usage scenarios:
//
//  1. IN clauses:
//     WHERE id IN (#{item})
//
//  2. Batch inserts:
//     INSERT INTO users VALUES
//     <foreach collection="users" item="user" separator=",">
//     (#{user.id}, #{user.name})
//     </foreach>
//
//  3. Multiple conditions:
//     <foreach collection="ids" item="id" separator="OR">
//     id = #{id}
//     </foreach>
//
// Example results:
//
//	Input collection: [1, 2, 3]
//	Configuration: open="(", separator=",", close=")"
//	Output: "(1,2,3)"
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
func (f ForeachNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {

	// if item already exists
	if _, exists := p.Get(f.Item); exists {
		return "", nil, fmt.Errorf("item %s already exists", f.Item)
	}

	// one collection from parameter
	value, exists := p.Get(f.Collection)
	if !exists {
		return "", nil, fmt.Errorf("collection %s not found", f.Collection)
	}

	// if valueItem can not be iterated
	if !value.CanInterface() {
		return "", nil, fmt.Errorf("collection %s can not be iterated", f.Collection)
	}

	// if valueItem is not a slice
	for value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Array, reflect.Slice:
		return f.acceptSlice(value, translator, p)
	case reflect.Map:
		return f.acceptMap(value, translator, p)
	default:
		return "", nil, fmt.Errorf("collection %s is not a slice or map", f.Collection)
	}
}

func (f ForeachNode) acceptSlice(value reflect.Value, translator driver.Translator, p Parameter) (query string, args []any, err error) {
	sliceLength := value.Len()

	if sliceLength == 0 {
		return "", nil, nil
	}

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.WriteString(f.Open)

	end := sliceLength - 1

	// group wraps parameter
	// nil is for placeholder
	group := eval.ParamGroup{nil, p}

	for i := 0; i < sliceLength; i++ {

		item := value.Index(i).Interface()

		group[0] = eval.H{f.Item: item, f.Index: i}.AsParam()

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, group)
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

	// if sliceLength is not zero, add close
	builder.WriteString(f.Close)

	return builder.String(), args, nil
}

func (f ForeachNode) acceptMap(value reflect.Value, translator driver.Translator, p Parameter) (query string, args []any, err error) {
	keys := value.MapKeys()

	if len(keys) == 0 {
		return "", nil, nil
	}

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.WriteString(f.Open)

	end := len(keys) - 1

	var index int

	// group wraps parameter
	// nil is for placeholder
	group := eval.ParamGroup{nil, p}

	for _, key := range keys {

		item := value.MapIndex(key).Interface()

		group[0] = eval.H{f.Item: item, f.Index: key.Interface()}.AsParam()

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, group)
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

		if index < end {
			builder.WriteString(f.Separator)
		}

		index++
	}

	builder.WriteString(f.Close)

	return builder.String(), args, nil
}

var _ Node = (*ForeachNode)(nil)

// SetNode represents an SQL SET clause for UPDATE statements.
// It manages a group of assignment expressions and automatically handles
// the comma separators and SET prefix.
//
// Features:
//   - Automatically adds "SET" prefix
//   - Manages comma separators between assignments
//   - Handles dynamic assignments based on conditions
//
// Example XML:
//
//	<update id="updateUser">
//	  UPDATE users
//	  <set>
//	    <if test='name != ""'>
//	      name = #{name},
//	    </if>
//	    <if test="age > 0">
//	      age = #{age},
//	    </if>
//	    <if test="status != 0">
//	      status = #{status}
//	    </if>
//	  </set>
//	  WHERE id = #{id}
//	</update>
//
// Example results:
//
//	Case 1 (name and age set):
//	  UPDATE users SET name = ?, age = ? WHERE id = ?
//
//	Case 2 (only status set):
//	  UPDATE users SET status = ? WHERE id = ?
//
// Note: The node automatically handles trailing commas and ensures
// proper formatting of the SET clause regardless of which fields
// are included dynamically.
type SetNode struct {
	Nodes NodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (s SetNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	query, args, err = s.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}
	if query != "" {
		query = "SET " + query
	}
	query = strings.TrimSuffix(query, ",")
	return query, args, nil
}

var _ Node = (*SetNode)(nil)

// SQLNode represents a complete SQL statement with its metadata and child nodes.
// It serves as the root node for a single SQL operation (SELECT, INSERT, UPDATE, DELETE)
// and manages the entire SQL generation process.
//
// Fields:
//   - id: Unique identifier for the SQL statement within the mapper
//   - nodes: Collection of child nodes that form the complete SQL
//   - mapper: Reference to the parent Mapper for context and configuration
//
// Example XML:
//
//	<select id="getUserById">
//	  SELECT *
//	  FROM users
//	  <where>
//	    <if test="id != 0">
//	      id = #{id}
//	    </if>
//	  </where>
//	</select>
//
// Usage scenarios:
//  1. SELECT statements with dynamic conditions
//  2. INSERT statements with optional fields
//  3. UPDATE statements with dynamic SET clauses
//  4. DELETE statements with complex WHERE conditions
//
// Features:
//   - Manages complete SQL statement generation
//   - Handles parameter binding
//   - Supports dynamic SQL through child nodes
//   - Maintains connection to mapper context
//   - Enables statement reuse through ID reference
//
// Note: The id must be unique within its mapper context to allow
// proper statement lookup and execution.
type SQLNode struct {
	id     string    // Unique identifier for the SQL statement
	nodes  NodeGroup // Child nodes forming the SQL statement
	mapper *Mapper   // Parent mapper reference
}

// ID returns the id of the node.
func (s SQLNode) ID() string {
	return s.id
}

// Accept accepts parameters and returns query and arguments.
func (s SQLNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	return s.nodes.Accept(translator, p)
}

var _ Node = (*SQLNode)(nil)

// IncludeNode represents a reference to another SQL fragment, enabling SQL reuse.
// It allows common SQL fragments to be defined once and included in multiple places,
// promoting code reuse and maintainability.
//
// Fields:
//   - sqlNode: The referenced SQL fragment node
//   - mapper: Reference to the parent Mapper for context
//   - refId: ID of the SQL fragment to include
//
// Example XML:
//
//	<!-- Common WHERE clause -->
//	<sql id="userFields">
//	  id, name, age, status
//	</sql>
//
//	<!-- Using the include -->
//	<select id="getUsers">
//	  SELECT
//	  <include refid="userFields"/>
//	  FROM users
//	  WHERE status = #{status}
//	</select>
//
// Features:
//   - Enables SQL fragment reuse
//   - Supports cross-mapper references
//   - Maintains consistent SQL patterns
//   - Reduces code duplication
//
// Usage scenarios:
//  1. Common column lists
//  2. Shared WHERE conditions
//  3. Reusable JOIN clauses
//  4. Standard filtering conditions
//
// Note: The refId must reference an existing SQL fragment defined with
// the <sql> tag. The reference can be within the same mapper or from
// another mapper if properly configured.
type IncludeNode struct {
	sqlNode Node
	mapper  *Mapper
	refId   string
}

// Accept accepts parameters and returns query and arguments.
func (i *IncludeNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	if i.sqlNode == nil {
		// lazy loading
		// does it need to be thread safe?
		sqlNode, err := i.mapper.GetSQLNodeByID(i.refId)
		if err != nil {
			return "", nil, err
		}
		i.sqlNode = sqlNode
	}
	return i.sqlNode.Accept(translator, p)
}

var _ Node = (*IncludeNode)(nil)

// ChooseNode implements a switch-like conditional structure for SQL generation.
// It evaluates multiple conditions in order and executes the first matching case,
// with an optional default case (otherwise).
//
// Fields:
//   - WhenNodes: Ordered list of conditional branches to evaluate
//   - OtherwiseNode: Default branch if no when conditions match
//
// Example XML:
//
//	<choose>
//	  <when test="id != 0">
//	    AND id = #{id}
//	  </when>
//	  <when test='name != ""'>
//	    AND name LIKE CONCAT('%', #{name}, '%')
//	  </when>
//	  <otherwise>
//	    AND status = 'ACTIVE'
//	  </otherwise>
//	</choose>
//
// Behavior:
//  1. Evaluates each <when> condition in order
//  2. Executes SQL from first matching condition
//  3. If no conditions match, executes <otherwise> if present
//  4. If no conditions match and no otherwise, returns empty result
//
// Usage scenarios:
//  1. Complex conditional logic in WHERE clauses
//  2. Dynamic sorting options
//  3. Different JOIN conditions
//  4. Status-based queries
//
// Example results:
//
//	Case 1 (id present):
//	  AND id = ?
//	Case 2 (only name present):
//	  AND name LIKE ?
//	Case 3 (neither present):
//	  AND status = 'ACTIVE'
//
// Note: Similar to a switch statement in programming languages,
// only the first matching condition is executed.
type ChooseNode struct {
	WhenNodes     []Node
	OtherwiseNode Node
}

// Accept accepts parameters and returns query and arguments.
func (c ChooseNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
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

var _ Node = (*ChooseNode)(nil)

// WhenNode is an alias for ConditionNode, representing a conditional branch
// within a <choose> statement. It evaluates a condition and executes its
// content if the condition is true and it's the first matching condition
// in the choose block.
//
// Behavior:
//   - Evaluates condition using same rules as ConditionNode
//   - Only executes if it's the first true condition in choose
//   - Subsequent true conditions are ignored
//
// Example XML:
//
//	<choose>
//	  <when test='type == "PREMIUM"'>
//	    AND membership_level = 'PREMIUM'
//	  </when>
//	  <when test='type == "BASIC"'>
//	    AND membership_level IN ('BASIC', 'STANDARD')
//	  </when>
//	</choose>
//
// Supported conditions:
//   - Boolean expressions
//   - Numeric comparisons
//   - String comparisons
//   - Null checks
//   - Property access
//
// Note: Unlike a standalone ConditionNode, WhenNode's execution
// is controlled by its parent ChooseNode and follows choose-when
// semantics similar to switch-case statements.
//
// See ConditionNode for detailed condition evaluation rules.
type WhenNode = ConditionNode

var _ Node = (*WhenNode)(nil)

// OtherwiseNode represents the default branch in a <choose> statement,
// which executes when none of the <when> conditions are met.
// It's similar to the 'default' case in a switch statement.
//
// Fields:
//   - Nodes: Group of nodes containing the default SQL fragments
//
// Example XML:
//
//	<choose>
//	  <when test="status != nil">
//	    AND status = #{status}
//	  </when>
//	  <when test="type != nil">
//	    AND type = #{type}
//	  </when>
//	  <otherwise>
//	    AND is_deleted = 0
//	    AND status = 'ACTIVE'
//	  </otherwise>
//	</choose>
//
// Behavior:
//   - Executes only if all <when> conditions are false
//   - No condition evaluation needed
//   - Can contain multiple SQL fragments
//   - Optional within <choose> block
//
// Usage scenarios:
//  1. Default filtering conditions
//  2. Fallback sorting options
//  3. Default join conditions
//  4. Error prevention (ensuring non-empty WHERE clauses)
//
// Example results:
//
//	When no conditions match:
//	  AND is_deleted = 0 AND status = 'ACTIVE'
//
// Note: Unlike WhenNode, OtherwiseNode doesn't evaluate any conditions.
// It simply provides default SQL fragments when needed.
type OtherwiseNode struct {
	Nodes NodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (o OtherwiseNode) Accept(translator driver.Translator, p Parameter) (query string, args []any, err error) {
	return o.Nodes.Accept(translator, p)
}

var _ Node = (*OtherwiseNode)(nil)

// valueItem is a element of ValuesNode.
type valueItem struct {
	column string
	value  string
}

// ValuesNode is a node of values.
// only support for insert.
type ValuesNode []*valueItem

// Accept accepts parameters and returns query and arguments.
func (v ValuesNode) Accept(translator driver.Translator, param Parameter) (query string, args []any, err error) {
	if len(v) == 0 {
		return "", nil, nil
	}
	builder := getStringBuilder()
	defer putStringBuilder(builder)
	builder.WriteString("(")
	builder.WriteString(v.columns())
	builder.WriteString(") VALUES (")
	builder.WriteString(v.values())
	builder.WriteString(")")
	node := NewTextNode(builder.String())
	return node.Accept(translator, param)
}

// columns returns columns of values.
func (v ValuesNode) columns() string {
	columns := make([]string, 0, len(v))
	for _, item := range v {
		columns = append(columns, item.column)
	}
	return strings.Join(columns, ", ")
}

// values returns values of values.
func (v ValuesNode) values() string {
	values := make([]string, 0, len(v))
	for _, item := range v {
		values = append(values, item.value)
	}
	return strings.Join(values, ", ")
}

// selectFieldAliasItem is a element of SelectFieldAliasNode.
type selectFieldAliasItem struct {
	column string
	alias  string
}

// SelectFieldAliasNode is a node of select field alias.
type SelectFieldAliasNode []*selectFieldAliasItem

// Accept accepts parameters and returns query and arguments.
func (s SelectFieldAliasNode) Accept(_ driver.Translator, _ Parameter) (query string, args []any, err error) {
	if len(s) == 0 {
		return "", nil, nil
	}
	fields := make([]string, 0, len(s))
	for _, item := range s {
		field := item.column
		if item.alias != "" && item.alias != item.column {
			field = field + " AS " + item.alias
		}
		fields = append(fields, field)
	}
	return strings.Join(fields, ", "), nil, nil
}
