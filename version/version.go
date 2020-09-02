package version

import "fmt"

var (
	version  string = "0.4.0"
	revision string = "dev"
)

// String returns scenarigo version as string.
func String() string {
	if revision == "" {
		return fmt.Sprintf("v%s", version)
	}
	return fmt.Sprintf("v%s-%s", version, revision)
}
