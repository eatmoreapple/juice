package main

import (
	"log"

	_ "github.com/eatmoreapple/juice/cmd/juice/impl"
	"github.com/eatmoreapple/juice/cmd/juice/internal/cmd"
	_ "github.com/eatmoreapple/juice/cmd/juice/namespace"
)

func main() {
	if err := cmd.Do(); err != nil {
		log.Println(err)
	}
}
