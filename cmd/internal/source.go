package internal

import (
	"go/format"
)

func formatCode(code string) string {
	result, err := format.Source([]byte(code))
	if err != nil {
		panic(err)
	}
	return string(result)
}
