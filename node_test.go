package juice

import (
	"reflect"
	"testing"

	"github.com/eatmoreapple/juice/driver"
)

func TestForeachNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node := ForeachNode{
		Nodes:      []Node{TextNode("(#{item.id}, #{item.name})")},
		Item:       "item",
		Collection: "list",
		Separator:  ", ",
	}
	params := map[string]reflect.Value{"list": reflect.ValueOf([]map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	})}
	query, args, err := node.Accept(drv.Translate(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(?, ?), (?, ?)" {
		t.Error("query error")
		return
	}
	if len(args) != 4 {
		t.Error("args error")
		return
	}
	if args[0] != 1 || args[1] != "a" || args[2] != 2 || args[3] != "b" {
		t.Error("args error")
		return
	}
}

func TestIfNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node := IfNode{
		Nodes: []Node{TextNode("select * from user where id = #{id}")},
	}

	if node.Parse("id > 0") != nil {
		t.Error("init error")
		return
	}

	query, args, err := node.Accept(drv.Translate(), map[string]reflect.Value{"id": reflect.ValueOf(1)})
	if err != nil {
		t.Error(err)
		return
	}
	if query != "select * from user where id = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 1 {
		t.Error("args error")
		return
	}
	if args[0] != 1 {
		t.Error("args error")
		return
	}
}

func TestTextNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node := TextNode("select * from user where id = #{id}")
	query, args, err := node.Accept(drv.Translate(), map[string]reflect.Value{"id": reflect.ValueOf(1)})
	if err != nil {
		t.Error(err)
		return
	}
	if query != "select * from user where id = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 1 {
		t.Error("args error")
		return
	}
	if args[0] != 1 {
		t.Error("args error")
		return
	}
}

func TestWhereNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node := WhereNode{
		Nodes: []Node{
			TextNode("AND id = #{id}"),
			TextNode("AND name = #{name}"),
		},
	}
	params := map[string]reflect.Value{
		"id":   reflect.ValueOf(1),
		"name": reflect.ValueOf("a"),
	}
	query, args, err := node.Accept(drv.Translate(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "WHERE id = ? AND name = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 2 {
		t.Error("args error")
		return
	}
	if args[0] != 1 || args[1] != "a" {
		t.Error("args error")
		return
	}

	node = WhereNode{
		Nodes: []Node{
			TextNode("id = #{id}"),
			TextNode("AND name = #{name}"),
		},
	}

	if query != "WHERE id = ? AND name = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 2 {
		t.Error("args error")
		return
	}
	if args[0] != 1 || args[1] != "a" {
		t.Error("args error")
		return
	}

}

func TestTrimNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	ifNode := &IfNode{
		Nodes: []Node{TextNode("name,")},
	}
	if err := ifNode.Parse("id > 0"); err != nil {
		t.Error(err)
		return
	}
	node := TrimNode{
		Nodes: []Node{
			ifNode,
		},
		Prefix:          "(",
		Suffix:          ")",
		SuffixOverrides: []string{","},
	}
	params := map[string]reflect.Value{
		"id":   reflect.ValueOf(1),
		"name": reflect.ValueOf("a"),
	}
	query, args, err := node.Accept(drv.Translate(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(name)" {
		t.Log(query)
		t.Error("query error")
		return
	}
	if len(args) != 0 {
		t.Error("args error")
		return
	}

}

func TestSetNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node := SetNode{
		Nodes: []Node{
			TextNode("id = #{id},"),
			TextNode("name = #{name},"),
		},
	}
	params := map[string]reflect.Value{
		"id":   reflect.ValueOf(1),
		"name": reflect.ValueOf("a"),
	}
	query, args, err := node.Accept(drv.Translate(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "SET id = ?, name = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 2 {
		t.Error("args error")
		return
	}
	if args[0] != 1 || args[1] != "a" {
		t.Error("args error")
		return
	}
}
