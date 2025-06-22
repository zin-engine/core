package config

import (
	"path/filepath"
	"strings"
	"zin-engine/utils"
)

func CheckZinIgnore(rootDir string, path string) bool {
	ignoreFile := filepath.Join(rootDir, ".zinignore")
	path = filepath.Clean(path)

	if path == "/.zinignore" {
		return true
	}

	// As .zinignore is not on root, expose all files
	if !utils.FileExists(ignoreFile) {
		return false
	}

	// Get .zinignore file content
	// Block all request, as .zinignore is there but failed to get contents
	data, err := utils.GetFileContent(ignoreFile)
	if err != nil {
		return true
	}

	// Split into lines & loop to check if current path is blocked
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}

		// Normalize and compare zinignore entry with the path
		entry = filepath.Clean("/" + strings.TrimSpace(entry))
		if entry == path {
			return true
		}
	}

	return false
}
