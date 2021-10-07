package filepathutil

import "path/filepath"

// From returns the path as the path from the base directory.
func From(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
