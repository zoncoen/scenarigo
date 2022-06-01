//go:build race
// +build race

package plugin

import (
	"testing"
)

func openPlugin(t *testing.T, p string) {
	t.Helper()
}
