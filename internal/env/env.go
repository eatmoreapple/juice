package env

import "os"

// DefaultParamKey is the default key of the parameter.
var DefaultParamKey = func() string {
	// try to get the key from environment variable
	key := os.Getenv("JUICE_PARAM_KEY")
	// if not found, use the default key
	if len(key) == 0 {
		key = "param"
	}
	return key
}()
