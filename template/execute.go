package template

import (
	"fmt"
	"go/token"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
)

var (
	yamlMapItemType  = reflect.TypeOf(yaml.MapItem{})
	yamlMapSliceType = reflect.TypeOf(yaml.MapSlice{})
)

// Execute executes templates of i with data.
func Execute(i, data interface{}) (interface{}, error) {
	v, err := execute(reflect.ValueOf(i), data)
	if err != nil {
		return nil, err
	}
	if v.IsValid() {
		return v.Interface(), nil
	}
	return nil, nil
}

func structFieldName(field reflect.StructField) string {
	fieldName := field.Name
	tag := field.Tag.Get("yaml")
	if tag == "" {
		return fieldName
	}

	tagValues := strings.Split(tag, ",")
	if len(tagValues) > 0 && tagValues[0] != "" {
		return tagValues[0]
	}
	return fieldName
}

func execute(in reflect.Value, data interface{}) (reflect.Value, error) {
	v := reflectutil.Elem(in)
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			e := v.MapIndex(k)
			if !isNil(e) {
				x, err := convert(v.Type().Elem())(execute(e, data))
				if err != nil {
					key := fmt.Sprint(k.Interface())
					return reflect.Value{}, errors.WithPath(err, key)
				}
				v.SetMapIndex(k, x)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if !isNil(e) {
				x, err := convert(v.Type().Elem())(execute(e, data))
				if err != nil {
					if v.Type() != yamlMapSliceType {
						err = errors.WithPath(err, fmt.Sprintf("[%d]", i))
					}
					return reflect.Value{}, err
				}
				e.Set(x)
			}
		}
	case reflect.Struct:
		if !v.CanSet() {
			v = makePtr(v).Elem() // create pointer to enable to set values
		}
		switch v.Type() {
		case yamlMapItemType:
			value := v.FieldByName("Value")
			if !isNil(value) {
				x, err := execute(value, data)
				if err != nil {
					key := fmt.Sprint(v.FieldByName("Key").Interface())
					return reflect.Value{}, errors.WithPath(err, key)
				}
				value.Set(x)
			}
		default:
			for i := 0; i < v.NumField(); i++ {
				if !token.IsExported(v.Type().Field(i).Name) {
					continue // skip unexported field
				}
				field := v.Field(i)
				x, err := convert(field.Type())(execute(field, data))
				if err != nil {
					fieldName := structFieldName(v.Type().Field(i))
					return reflect.Value{}, errors.WithPath(err, fieldName)
				}
				if err := reflectutil.Set(field, x); err != nil {
					fieldName := structFieldName(v.Type().Field(i))
					return reflect.Value{}, errors.WithPath(err, fieldName)
				}
			}
		}
	case reflect.String:
		tmpl, err := New(v.String())
		if err != nil {
			return reflect.Value{}, err
		}
		x, err := tmpl.Execute(data)
		if err != nil {
			return reflect.Value{}, err
		}
		v = reflect.ValueOf(x)
	default:
	}

	// keep the original type as much as possible
	if in.IsValid() {
		if converted, err := convert(in.Type())(v, nil); err == nil {
			v = converted
		}
		// keep the original address
		if in.Type().Kind() == reflect.Ptr && v.Type().Kind() == reflect.Ptr {
			if v.Elem().Type().AssignableTo(in.Elem().Type()) {
				reflectutil.Set(in.Elem(), v.Elem())
				v = in
			}
		}
	}
	return v, nil
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// convert returns a function that converts a value to the given type t.
func convert(t reflect.Type) func(reflect.Value, error) (reflect.Value, error) {
	return func(v reflect.Value, err error) (result reflect.Value, resErr error) {
		if err != nil {
			return v, err
		}
		vv, _, err := reflectutil.Convert(t, v)
		if err != nil {
			return v, err
		}
		return vv, nil
	}
}

func makePtr(v reflect.Value) reflect.Value {
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	return ptr
}
