package main

import (
	"log"

	_ "github.com/eatmoreapple/juice/cmd/juice/impl"
	"github.com/eatmoreapple/juice/cmd/juice/internal/cmd"
)

func main() {
	if err := cmd.Do(); err != nil {
		log.Println(err)
	}
}
