package stats

import (
	"reflect"
	"strings"
)

type FlatMap map[string]interface{}

func isEmptyValue(v reflect.Value) bool {
	return v.Interface() == reflect.Zero(v.Type()).Interface()
}

func keyForField(field reflect.StructField, v reflect.Value) string {
	if tag := field.Tag.Get("json"); tag != "" {
		tokens := strings.SplitN(tag, ",", 2)
		name := tokens[0]
		opts := ""

		if len(tokens) > 1 {
			opts = tokens[1]
		}

		if name == "-" || strings.Contains(opts, "omitempty") && isEmptyValue(v) {
			return ""
		} else if name != "" {
			return name
		}
	}

	return field.Name
}

func recursiveFlatten(val reflect.Value, prefix string, output FlatMap) int {
	valType := val.Type()
	added := 0

	for i := 0; i < val.NumField(); i++ {
		child := val.Field(i)
		childType := valType.Field(i)
		key := prefix + keyForField(childType, child)

		if childType.PkgPath != "" || key == "" {
			continue
		} else if child.Kind() == reflect.Struct {
			if recursiveFlatten(child, key+".", output) != 0 {
				continue
			}
		}

		output[key] = child.Addr().Interface()
		added++
	}

	return added
}

func flattenPointer(val reflect.Value) FlatMap {
	if val.Kind() == reflect.Ptr {
		return flattenPointer(val.Elem())
	}

	if val.Kind() != reflect.Struct {
		panic("must be called with a struct type")
	}

	m := FlatMap{}
	recursiveFlatten(val, "", m)
	return m
}

func Flatten(val interface{}) FlatMap {
	return flattenPointer(reflect.ValueOf(val))
}
