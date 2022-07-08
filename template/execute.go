package template

import (
	"fmt"
	"go/token"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/zoncoen/query-go"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
)

var (
	yamlMapItemType  = reflect.TypeOf(yaml.MapItem{})
	yamlMapSliceType = reflect.TypeOf(yaml.MapSlice{})
	lazyFuncType     = reflect.TypeOf(lazyFunc{})
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
	case reflect.Invalid:
		return in, nil
	case reflect.Map:
		for _, k := range v.MapKeys() {
			e := v.MapIndex(k)
			if !isNil(e) {
				keyStr := fmt.Sprint(k.Interface())
				key, err := execute(k, data)
				if err != nil {
					return reflect.Value{}, errors.WithPath(err, keyStr)
				}
				// left arrow function
				if ke := reflectutil.Elem(key); ke.IsValid() && ke.Type() == lazyFuncType && ke.CanInterface() {
					if len(v.MapKeys()) != 1 {
						return reflect.Value{}, errors.New("invalid left arrow function call")
					}
					if !isNil(e) {
						x, err := execute(e, data)
						if err != nil {
							return reflect.Value{}, errors.WithPath(err, keyStr)
						}
						e = x
					}
					f := ke.Interface().(lazyFunc)
					res, err := executeLeftArrowFunction(f.f, e)
					if err != nil {
						return reflect.Value{}, fmt.Errorf("failed to execute left arrow function: %w", err)
					}
					v = res
					break
				}
				x, err := convert(e.Type())(execute(e, data))
				if err != nil {
					return reflect.Value{}, errors.WithPath(err, keyStr)
				}
				v.SetMapIndex(k, reflect.Value{}) // delete old value
				v.SetMapIndex(key, x)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if !isNil(e) {
				if !e.CanSet() {
					e = makePtr(e).Elem() // create pointer to enable to set values
				}
				if e.Type() == yamlMapItemType {
					key := e.FieldByName("Key")
					keyStr := fmt.Sprint(key.Interface())
					value := e.FieldByName("Value")
					if !isNil(key) {
						k, err := execute(key, data)
						if err != nil {
							return reflect.Value{}, errors.WithPath(err, keyStr)
						}
						key = k
					}
					// left arrow function
					if ke := reflectutil.Elem(key); ke.IsValid() && ke.Type() == lazyFuncType && ke.CanInterface() {
						if v.Len() != 1 {
							return reflect.Value{}, errors.New("invalid left arrow function call")
						}
						if !isNil(value) {
							x, err := execute(value, data)
							if err != nil {
								return reflect.Value{}, errors.WithPath(err, keyStr)
							}
							value = x
						}
						f := ke.Interface().(lazyFunc)
						res, err := executeLeftArrowFunction(f.f, value)
						if err != nil {
							return reflect.Value{}, fmt.Errorf("failed to execute left arrow function: %w", err)
						}
						v = res
						break
					}
				}
				x, err := convert(e.Type())(execute(e, data))
				if err != nil {
					return reflect.Value{}, errors.WithQuery(err, query.New().Index(i))
				}
				e.Set(x)
			}
		}
	case reflect.Struct:
		if !v.CanSet() {
			v = makePtr(v).Elem() // create pointer to enable to set values
		}
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
	if in.IsValid() && v.IsValid() {
		if converted, err := convert(in.Type())(v, nil); err == nil {
			v = converted
		}
		// keep the original address
		if in.Type().Kind() == reflect.Ptr && v.Type().Kind() == reflect.Ptr {
			if v.Elem().Type().AssignableTo(in.Elem().Type()) {
				if err := reflectutil.Set(in.Elem(), v.Elem()); err != nil {
					return reflect.Value{}, err
				}
				v = in
			}
		}
	}
	return v, nil
}

func executeLeftArrowFunction(f Func, v reflect.Value) (reflect.Value, error) {
	s := new(funcStash)
	x, err := replaceFuncs(v, s)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to stash functions to marshal argument to YAML: %w", err)
	}
	b, err := yaml.Marshal(x.Interface())
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to marshal argument to YAML: %w", err)
	}
	arg, err := f.UnmarshalArg(func(v interface{}) error {
		if err := yaml.UnmarshalWithOptions(b, v, yaml.UseOrderedMap(), yaml.Strict()); err != nil {
			return err
		}

		// Restore functions that are replaced into strings.
		// See the "HACK" comment of *Template.executeParameterExpr method.
		arg, err := Execute(v, s)
		if err != nil {
			return fmt.Errorf("failed to restore functions: %w", err)
		}
		// NOTE: Decode method ensures that v is a pointer.
		rv := reflect.ValueOf(v).Elem()
		ev, err := convert(rv.Type())(reflect.ValueOf(arg), nil)
		if err != nil {
			return err
		}
		rv.Set(ev)

		return nil
	})
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to unmarshal argument: %w", err)
	}
	res, err := f.Exec(arg)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to execute function: %w", err)
	}
	return reflect.ValueOf(res), nil
}

func replaceFuncs(in reflect.Value, s *funcStash) (reflect.Value, error) {
	v := reflectutil.Elem(in)

	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			e := v.MapIndex(k)
			if !isNil(e) {
				x, err := replaceFuncs(e, s)
				if err != nil {
					return reflect.Value{}, err
				}
				v.SetMapIndex(k, x)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if !isNil(e) {
				x, err := replaceFuncs(e, s)
				if err != nil {
					return reflect.Value{}, err
				}
				e.Set(x)
			}
		}
	case reflect.Struct:
		if !v.CanSet() {
			v = makePtr(v).Elem() // create pointer to enable to set values
		}
		for i := 0; i < v.NumField(); i++ {
			if !token.IsExported(v.Type().Field(i).Name) {
				continue // skip unexported field
			}
			field := v.Field(i)
			x, err := replaceFuncs(field, s)
			if err != nil {
				return reflect.Value{}, err
			}
			if err := reflectutil.Set(field, x); err != nil {
				fieldName := structFieldName(v.Type().Field(i))
				return reflect.Value{}, errors.WithPath(err, fieldName)
			}
		}
	case reflect.Func:
		return reflect.ValueOf(fmt.Sprintf("{{%s}}", s.save(v.Interface()))), nil
	default:
		return in, nil
	}

	// keep the original type as much as possible
	if in.IsValid() && v.IsValid() {
		if converted, err := convert(in.Type())(v, nil); err == nil {
			v = converted
		}
		// keep the original address
		if in.Type().Kind() == reflect.Ptr && v.Type().Kind() == reflect.Ptr {
			if v.Elem().Type().AssignableTo(in.Elem().Type()) {
				if err := reflectutil.Set(in.Elem(), v.Elem()); err != nil {
					return reflect.Value{}, err
				}
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
