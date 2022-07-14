//go:build !darwin
// +build !darwin

package reporter

import "time"

const durationTestUnit = 10 * time.Millisecond
