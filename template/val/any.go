package val

import (
	"fmt"
	"reflect"
)

var anyType = createAnyType(nil)

func createAnyType(v any) Type {
	name := "any"
	if v != nil {
		name = fmt.Sprintf("any[%T]", v)
	}
	return basicType{
		name: name,
		newValue: func(v any) (Value, error) {
			return Any{v}, nil
		},
		convert: func(v Value) (Value, error) {
			if v == nil {
				return Any{nil}, nil
			}
			return Any{v.GoValue()}, nil
		},
	}
}

// Any represents an undefined type value.
type Any struct {
	v any
}

// Type implements Value interface.
func (a Any) Type() Type {
	return createAnyType(a.v)
}

// GoValue implements Value interface.
func (a Any) GoValue() any { return a.v }

// Equal implements Equaler interface.
func (a Any) Equal(v Value) (LogicalValue, error) {
	xv, yv := reflect.ValueOf(a.GoValue()), reflect.ValueOf(v.GoValue())
	if eq, ok := equal(xv, yv); ok {
		return Bool(eq), nil
	}
	return Bool(false), ErrOperationNotDefined
}

func equal(x, y reflect.Value) (bool, bool) {
	if comparable(x) {
		return reflectEqual(x, y), true
	}
	return false, false
}
