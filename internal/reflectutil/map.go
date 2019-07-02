package reflectutil

import (
	"reflect"

	"github.com/pkg/errors"
)

// ConvertStringsMap converts map[string]string to map[string][]string.
func ConvertStringsMap(v reflect.Value) (map[string][]string, error) {
	v = Elem(v)
	if !v.IsValid() {
		return nil, errors.New("invalid")
	}
	if v.Kind() != reflect.Map {
		return nil, errors.Errorf("expected map but got %T", v.Interface())
	}

	m := map[string][]string{}
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key().Interface()
		key, ok := k.(string)
		if !ok {
			return nil, errors.Errorf("expected key is string but got %T", key)
		}

		strs, err := convertStrings(iter.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "%s is invalid", key)
		}
		m[key] = strs
	}

	return m, nil
}

func convertStrings(v reflect.Value) ([]string, error) {
	if !v.IsValid() {
		return nil, errors.New("invalid")
	}
	if k := v.Kind(); k == reflect.Interface || k == reflect.Ptr {
		if v.IsNil() {
			return nil, errors.New("value is nil")
		}
	}
	v = Elem(v)
	switch v.Kind() {
	case reflect.String:
		return []string{v.String()}, nil
	case reflect.Slice:
		var strs []string
		for i := 0; i < v.Len(); i++ {
			x := v.Index(i)
			if k := x.Kind(); k == reflect.Interface || k == reflect.Ptr {
				if x.IsNil() {
					return nil, errors.New("value is nil")
				}
			}
			x = Elem(x)
			if x.Kind() != reflect.String {
				return nil, errors.Errorf("expected string but got %T", x.Interface())
			}
			strs = append(strs, x.String())
		}
		return strs, nil
	}
	return nil, errors.Errorf("expected string or []string but got %T", v.Interface())
}
