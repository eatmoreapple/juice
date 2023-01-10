package main

import (
	"fmt"

	"github.com/eatmoreapple/juice/cmd/juice/cmd"
)

func main() {
	if err := cmd.Do(); err != nil {
		fmt.Println(err)
	}
}
