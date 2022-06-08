package reflectutil

import "reflect"

var (
	TypeError = reflect.TypeOf((*error)(nil)).Elem()
)
