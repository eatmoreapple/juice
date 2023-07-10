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
	"testing"

	"github.com/eatmoreapple/juice/driver"
)

func TestForeachNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	textNode, err := NewTextNode("(#{item.id}, #{item.name})")
	if err != nil {
		t.Error(err)
	}
	node := ForeachNode{
		Nodes:      []Node{textNode},
		Item:       "item",
		Collection: "list",
		Separator:  ", ",
	}
	params := H{"list": []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	}}
	query, args, err := node.Accept(drv.Translator(), params.AsParam())
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

func TestForeachMapNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	textNode, err := NewTextNode("(#{item}, #{index})")
	if err != nil {
		t.Error(err)
	}
	node := ForeachNode{
		Nodes:      []Node{textNode},
		Item:       "item",
		Index:      "index",
		Collection: "map",
		Separator:  ", ",
	}
	params := H{"map": map[string]any{"a": 1}}
	query, args, err := node.Accept(drv.Translator(), params.AsParam())
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(?, ?)" {
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

	params = H{"map": map[string]any{"a": 1, "b": 2}}
	query, args, err = node.Accept(drv.Translator(), params.AsParam())
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
}

func TestIfNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node1, _ := NewTextNode("select * from user where id = #{id}")
	node := &IfNode{
		Nodes: []Node{node1},
	}

	if node.Parse("id > 0") != nil {
		t.Error("init error")
		return
	}

	h := H{"id": 1}

	query, args, err := node.Accept(drv.Translator(), h.AsParam())
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
	node, _ := NewTextNode("select * from user where id = #{id}")
	param := newGenericParam(H{"id": 1}, "")
	query, args, err := node.Accept(drv.Translator(), param)
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
	node1, _ := NewTextNode("AND id = #{id}")
	node2, _ := NewTextNode("AND name = #{name}")
	node := WhereNode{
		Nodes: []Node{
			node1,
			node2,
		},
	}
	params := H{
		"id":   1,
		"name": "a",
	}
	query, args, err := node.Accept(drv.Translator(), newGenericParam(params, ""))
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
	node1, _ := NewTextNode("name,")
	ifNode := &IfNode{
		Nodes: []Node{node1},
	}
	if err := ifNode.Parse("id > 0"); err != nil {
		t.Error(err)
		return
	}
	node := &TrimNode{
		Nodes: []Node{
			ifNode,
		},
		Prefix:          "(",
		Suffix:          ")",
		SuffixOverrides: []string{","},
	}
	params := H{"id": 1, "name": "a"}
	query, args, err := node.Accept(drv.Translator(), params.AsParam())
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
	node1, _ := NewTextNode("id = #{id},")
	node2, _ := NewTextNode("name = #{name},")
	node := SetNode{
		Nodes: []Node{
			node1, node2,
		},
	}
	params := H{
		"id":   1,
		"name": "a",
	}
	query, args, err := node.Accept(drv.Translator(), newGenericParam(params, ""))
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
