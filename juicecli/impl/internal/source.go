package internal

import (
	"go/format"
	"log"
)

func formatCode(code string) string {
	result, err := format.Source([]byte(code))
	if err != nil {
		log.Fatal(err)
	}
	return string(result)
}
