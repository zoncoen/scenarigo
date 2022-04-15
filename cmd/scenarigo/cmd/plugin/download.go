//go:build go1.18
// +build go1.18

package plugin

func downloadCmd(mod string) []string {
	return []string{"get", mod}
}
