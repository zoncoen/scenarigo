//go:build go1.19

//nolint:all
package template

import "reflect"

func comparable(v reflect.Value) bool {
	k := v.Kind()
	switch k {
	case reflect.Invalid:
		return false

	case reflect.Array:
		switch v.Type().Elem().Kind() {
		case reflect.Interface, reflect.Array, reflect.Struct:
			for i := 0; i < v.Type().Len(); i++ {
				if !comparable(v.Index(i)) {
					return false
				}
			}
			return true
		}
		return typeComparable(v.Type())

	case reflect.Interface:
		return comparable(v.Elem())

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !comparable(v.Field(i)) {
				return false
			}
		}
		return true

	default:
		return typeComparable(v.Type())
	}
}

func typeComparable(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String, reflect.UnsafePointer:
		return true
	}
	return false
}

func equal(v, u reflect.Value) bool {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if u.Kind() == reflect.Interface {
		u = u.Elem()
	}

	if !v.IsValid() || !u.IsValid() {
		return v.IsValid() == u.IsValid()
	}

	if v.Kind() != u.Kind() || v.Type() != u.Type() {
		return false
	}

	// Handle each Kind directly rather than calling valueInterface
	// to avoid allocating.
	switch v.Kind() {
	default:
		panic("reflect.Value.Equal: invalid Kind")
	case reflect.Bool:
		return v.Bool() == u.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == u.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == u.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float() == u.Float()
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == u.Complex()
	case reflect.String:
		return v.String() == u.String()
	case reflect.Chan, reflect.Pointer, reflect.UnsafePointer:
		return v.Pointer() == u.Pointer()
	case reflect.Array:
		// u and v have the same type so they have the same length
		vl := v.Len()
		if vl == 0 {
			// panic on [0]func()
			if !v.Type().Elem().Comparable() {
				break
			}
			return true
		}
		for i := 0; i < vl; i++ {
			if !equal(v.Index(i), u.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		// u and v have the same type so they have the same fields
		nf := v.NumField()
		for i := 0; i < nf; i++ {
			if !equal(v.Field(i), u.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Func, reflect.Map, reflect.Slice:
		break
	}
	panic("reflect.Value.Equal: values of type " + v.Type().String() + " are not comparable")
}
