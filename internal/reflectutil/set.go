package reflectutil

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// Set assigns v to the value target.
func Set(target, v reflect.Value) (retErr error) {
	if !target.IsValid() {
		return errors.New("can not set to invalid value")
	}
	if !v.IsValid() {
		return nil
	}
	defer func() {
		if err := recover(); err != nil {
			retErr = fmt.Errorf("can not set %s to %s: %s", v.Type().String(), target.Type().String(), err)
		}
	}()
	if vv, ok, err := Convert(target.Type(), v); err == nil && ok {
		v = vv
	}
	if !target.CanSet() {
		if !target.CanAddr() {
			return fmt.Errorf("can not set to unaddressable value")
		}
		return fmt.Errorf("can not set to unexported struct field")
	}
	if !v.Type().AssignableTo(target.Type()) {
		return fmt.Errorf("%s is not assignable to %s", v.Type().String(), target.Type().String())
	}
	target.Set(v)
	return nil
}

// Convert returns the value v converted to type t.
func Convert(t reflect.Type, v reflect.Value) (_ reflect.Value, _ bool, retErr error) {
	if !v.IsValid() {
		return v, false, nil
	}
	defer func() {
		if err := recover(); err != nil {
			retErr = errors.Errorf("failed to convert %T to %s: %s", v.Interface(), t.Name(), err)
		}
	}()

	if v.Type().ConvertibleTo(t) {
		return v.Convert(t), true, nil
	}
	if v.Type().Kind() == reflect.Ptr {
		if v.Elem().Type().ConvertibleTo(t) {
			return v.Elem().Convert(t), true, nil
		}
	} else {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		if ptr.Type().ConvertibleTo(t) {
			return ptr.Convert(t), true, nil
		}
	}

	return v, false, nil
}

// ConvertInterface returns the value v converted to type t.
func ConvertInterface(t reflect.Type, v interface{}) (interface{}, bool, error) {
	vv, ok, err := Convert(t, reflect.ValueOf(v))
	if err != nil {
		return vv, ok, err
	}
	if ok {
		if vv.IsValid() && vv.CanInterface() {
			return vv.Interface(), ok, nil
		}
	}
	return v, ok, nil
}
