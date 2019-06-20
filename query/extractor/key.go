package extractor

import (
	"reflect"
	"strings"

	"github.com/zoncoen/query-go"
)

// Key returns a new key extractor.
func Key(key string) query.Extractor {
	return &keyExtractor{key}
}

type keyExtractor struct {
	key string
}

// Extract implements query.Extractor interface.
func (e *keyExtractor) Extract(v reflect.Value) (reflect.Value, bool) {
	if v.IsValid() {
		if i, ok := v.Interface().(query.KeyExtractor); ok {
			x, ok := i.ExtractByKey(e.key)
			return reflect.ValueOf(x), ok
		}
	}
	return e.extract(v)
}

func (e *keyExtractor) extract(v reflect.Value) (reflect.Value, bool) {
	v = elem(v)
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			k := elem(k)
			if k.String() == e.key {
				return v.MapIndex(k), true
			}
		}
	case reflect.Struct:
		inlines := []int{}
		for i := 0; i < v.Type().NumField(); i++ {
			field := v.Type().FieldByIndex([]int{i})
			name := strings.ToLower(field.Name)
			if tag, ok := field.Tag.Lookup("yaml"); ok {
				strs := strings.Split(tag, ",")
				for _, opt := range strs[1:] {
					if opt == "inline" {
						inlines = append(inlines, i)
					}
				}
				name = strs[0]
			}
			if name == e.key {
				return v.FieldByIndex([]int{i}), true
			}
		}
		for _, i := range inlines {
			if val, ok := e.Extract(v.Field(i)); ok {
				return val, true
			}
		}
	}
	return reflect.Value{}, false
}

// String implements query.Extractor interface.
func (e *keyExtractor) String() string {
	return "." + e.key
}

func elem(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		switch v.Type().Elem().Kind() {
		case reflect.Map, reflect.Struct, reflect.Slice, reflect.Array:
			v = v.Elem()
		}
	}
	return v
}
