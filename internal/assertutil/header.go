package assertutil

import (
	"fmt"
	"reflect"

	"github.com/goccy/go-yaml"

	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/errors"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
)

// BuildHeaderAssertion builds an assertion for headers.
func BuildHeaderAssertion(ctx *context.Context, in yaml.MapSlice) (assert.Assertion, error) {
	expects := make(yaml.MapSlice, len(in))
	for i, elem := range in {
		name, ok := stringify(elem.Key).(string)
		if !ok {
			return nil, errors.Errorf("name must be string but %T", elem.Key)
		}

		v, err := ctx.ExecuteTemplate(elem.Value)
		if err != nil {
			return nil, errors.WrapPath(err, name, "failed to execute template")
		}

		// Convert boolean and integer values to strings for ease of use.
		// All header values are strings.
		v = stringify(v)

		// Wrap with the "Contains" function to allow using not only an array but also just a string.
		if reflect.ValueOf(v).Kind() != reflect.Slice {
			v = assert.Contains(assert.Build(v))
		}

		elem.Value = v
		expects[i] = elem
	}
	return assert.Build(expects), nil
}

func stringify(i interface{}) interface{} {
	switch v := i.(type) {
	case yaml.MapSlice:
		for i, item := range v {
			v[i].Value = stringify(item.Value)
		}
		return v
	case []interface{}:
		for i, elm := range v {
			v[i] = stringify(elm)
		}
		return v
	}

	v := reflectutil.Elem(reflect.ValueOf(i))
	switch v.Kind() {
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return fmt.Sprint(v.Interface())
	default:
		return i
	}
}
