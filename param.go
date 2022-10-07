package pillow

import (
	"reflect"
	"strconv"
	"strings"
)

type Param map[string]reflect.Value

func (p Param) Get(name string) (reflect.Value, bool) {
	items := strings.Split(name, ".")
	var value reflect.Value
	for i, item := range items {
		if i == 0 {
			var exists bool
			value, exists = p[item]
			if !exists {
				return reflect.Value{}, false
			}
			continue
		}

		value = reflect.Indirect(value)

		switch value.Kind() {
		case reflect.Map:
			value = value.MapIndex(reflect.ValueOf(item))
		case reflect.Struct:
			field := value.FieldByName(item)
			if !field.IsValid() {
				// try to find it from tag
				for i := 0; i < value.NumField(); i++ {
					field := value.Type().Field(i)
					if field.Tag.Get("param") == item {
						value = value.Field(i)
						break
					}
				}
			} else {
				value = field
			}
			if !value.IsValid() {
				return reflect.Value{}, false
			}
		case reflect.Slice, reflect.Array:
			index, err := strconv.Atoi(item)
			if err != nil {
				return reflect.Value{}, false
			}
			value = value.Index(index)
		default:
			return reflect.Value{}, false
		}

		for value.Kind() == reflect.Interface {
			value = value.Elem()
		}
	}
	return value, value.IsValid()
}
