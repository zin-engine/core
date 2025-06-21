package config

import (
	"os"
	"path/filepath"
	"strings"
)

func LoadEnvironmentVars(rootDir string) map[string]string {
	envMap := make(map[string]string)
	envFile := filepath.Join(rootDir, ".env")

	data, err := os.ReadFile(envFile)
	if err != nil {
		return envMap
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Ignore comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		envMap[key] = value
	}

	return envMap
}
