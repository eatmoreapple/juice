package juice

import "strings"

type columnTag struct {
	// Name of the column.
	Name string
	// Omitempty the column.
	Omitempty bool
}

func (c *columnTag) parse(tag string) {
	items := strings.Split(tag, ",")
	if len(items) > 0 {
		c.Name = items[0]
	}
	if len(items) > 1 {
		c.Omitempty = items[1] == "omitempty"
	}
}

func (c *columnTag) reset() {
	c.Name = ""
	c.Omitempty = false
}

func newColumnTag(tag string) *columnTag {
	instance := &columnTag{}
	instance.parse(tag)
	return instance
}
