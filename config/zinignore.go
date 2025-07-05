package config

import (
	"path/filepath"
	"strings"
	"zin-engine/utils"
)

func CheckZinIgnore(rootDir string, path string) bool {
	ignoreFile := filepath.Join(rootDir, ".zinignore")
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "/")

	// .zinignore isn't at the root? Cool, guess we're open-sourcing the whole damn folder
	if !utils.FileExists(ignoreFile) {
		return false
	}

	// Load ignored files/dir list
	ignoreList := loadIgnoreList(rootDir)

	for _, ignore := range ignoreList {
		ignore = filepath.ToSlash(ignore)

		// Exact match
		if path == ignore {
			return true
		}

		// Directory prefix match (e.g. components/, static/)
		if strings.HasSuffix(ignore, "/") && strings.HasPrefix(path, ignore) {
			return true
		}
	}

	return false
}
