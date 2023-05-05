//go:build go1.20

package template

import "reflect"

func comparable(v reflect.Value) bool {
	return v.Comparable()
}

func reflectEqual(v, u reflect.Value) bool {
	return v.Equal(u)
}
