package juice

import (
	"reflect"
)

type tagParser struct {
	tag     string
	columns []string
	value   reflect.Value
}

func (p *tagParser) destPoint() ([]interface{}, error) {
	_type := p.value.Type()
	var dest = make([]interface{}, len(p.columns))
	for i := 0; i < _type.NumField(); i++ {
		field := _type.Field(i)
		tag := field.Tag.Get(p.tag)
		if tag == "" || tag == "-" {
			continue
		}
		var found bool
		for _, column := range p.columns {
			if found = tag == column; found {
				dest[i] = p.value.Field(i).Addr().Interface()
				break
			}
		}
		if !found {
			dest[i] = new(any)
		}
	}
	return dest, nil
}
