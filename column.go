package juice

import "strings"

type columnTag struct {
	// Name of the column.
	Name string
	// Ignore the column.
	Ignore bool
}

func (c *columnTag) parse(tag string) {
	items := strings.Split(tag, ",")
	if len(items) > 0 {
		c.Name = items[0]
	}
	if len(items) > 1 {
		c.Ignore = items[1] == "ignore"
	}
}

func (c *columnTag) reset() {
	c.Name = ""
	c.Ignore = false
}

func newColumnTag(tag string) *columnTag {
	instance := &columnTag{}
	instance.parse(tag)
	return instance
}
