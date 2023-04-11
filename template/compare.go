//go:build !go1.19

package template

import "reflect"

func comparable(v reflect.Value) bool {
	return v.Comparable()
}

func equal(v, u reflect.Value) bool {
	return v.Equal(u)
}
