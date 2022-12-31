package cmd

import (
	"go/format"
)

func formatCode(code string) string {
	result, _ := format.Source([]byte(code))
	return string(result)
}
