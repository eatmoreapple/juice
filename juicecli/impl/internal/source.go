package internal

import (
	"fmt"
	"go/format"
)

func formatCode(code string) string {
	result, err := format.Source([]byte(code))
	if err != nil {
		fmt.Println(code)
		panic(err)
	}
	return string(result)
}
