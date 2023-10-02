//go:build go1.21

package assert

import "sync"

func onceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2) {
	return sync.OnceValues(f)
}
