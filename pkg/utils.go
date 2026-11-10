package pkg

import (
	"os"
	"path/filepath"
	"strings"
)

func Contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// ExpandHome resolves a leading ~ to the user's home directory. Paths in the config and in the
// Raycast preferences are written as ~/.config/... so they stay portable across machines; every
// path read goes through here instead of relying on the shell to expand the tilde.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(homeDir, path[1:])
}
