package main

import (
	"fmt"

	"github.com/eatmoreapple/juice/cmd"
)

func main() {
	parser := cmd.Parser{}
	impl, err := parser.Parse()
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := impl.Generate(); err != nil {
		fmt.Println(err)
	}
}
