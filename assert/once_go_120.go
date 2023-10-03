//go:build !go1.21

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
// in the Go repository.

package assert

import "sync"

func onceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2) {
	var (
		once  sync.Once
		valid bool
		p     any
		r1    T1
		r2    T2
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		r1, r2 = f()
		valid = true
	}
	return func() (T1, T2) {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return r1, r2
	}
}
