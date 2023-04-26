package colorformat

import (
	"fmt"
)

// colorFormat is a wrapper of fmt.Print with color

func colorFormat(color string, a ...interface{}) string {
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", color, fmt.Sprint(a...))
}

// Red print red text
func Red(a ...interface{}) string {
	return colorFormat("31", a...)
}

// Green print green text
func Green(a ...interface{}) string {
	return colorFormat("32", a...)
}

// Yellow print yellow text
func Yellow(a ...interface{}) string {
	return colorFormat("33", a...)
}

// Blue print blue text
func Blue(a ...interface{}) string {
	return colorFormat("34", a...)
}

// Magenta print magenta text
func Magenta(a ...interface{}) string {
	return colorFormat("35", a...)
}

// Cyan print cyan text
func Cyan(a ...interface{}) string {
	return colorFormat("36", a...)
}
