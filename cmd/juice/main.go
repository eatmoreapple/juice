package main

import (
	"fmt"

	"github.com/eatmoreapple/juice/cmd/juice/iface"
	_ "github.com/eatmoreapple/juice/cmd/juice/impl"
)

func main() {
	if err := iface.Do(); err != nil {
		fmt.Println(err)
	}
}
