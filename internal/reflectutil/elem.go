package reflectutil

import "reflect"

// Elem calls v.Elem() recursively.
// It returns v as it is if v's Kind is not reflect.Interface or reflect.Ptr.
func Elem(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		return Elem(v.Elem())
	default:
		return v
	}
}
