package juice

import (
	"strconv"
)

// Settings is a slice of Setting.
// It is used to store the settings for your application.
// Settings won't be so large, so we don't need to use map.
type Settings []*Setting

// Get returns the value of the key.
func (s Settings) Get(name string) StringValue {
	for _, setting := range s {
		if setting.Name == name {
			return setting.Value
		}
	}
	return emptyStringValue
}

// Setting is a setting element.
type Setting struct {
	// The name of the setting.
	Name string `xml:"name,attr"`
	// The value of the setting.
	Value StringValue `xml:"value,attr"`
}

// emptyStringValue defines an empty string value.
const emptyStringValue = StringValue("")

// StringValue is a string value which can be converted to other types.
type StringValue string

// Bool returns true if the value is "true".
func (s StringValue) Bool() bool {
	value, _ := strconv.ParseBool(string(s))
	return value
}

// Int64 returns the value as int64.
func (s StringValue) Int64() int64 {
	value, _ := strconv.ParseInt(string(s), 10, 64)
	return value
}

// String returns the value as string.
func (s StringValue) String() string {
	return string(s)
}

// Float64 returns the value as float64.
func (s StringValue) Float64() float64 {
	value, _ := strconv.ParseFloat(string(s), 64)
	return value
}
