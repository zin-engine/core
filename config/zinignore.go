package config

import (
	"os"
	"path/filepath"
	"strings"
)

func CheckZinIgnore(rootDir string, requestPath string) bool {
	ignoreFile := filepath.Join(rootDir, ".zinignore")

	if requestPath == "/.zinignore" {
		return true
	}

	// Check if the file exists,
	// If not then allow all files to be served to client
	data, err := os.ReadFile(ignoreFile)
	if err != nil {
		return false
	}

	// Split into lines
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}

		// When requestPath matches a zinignore entry
		entry = "/" + entry
		if entry == requestPath {
			return true
		}
	}

	return false
}
