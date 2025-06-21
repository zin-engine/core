package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type RouteResult struct {
	Path string
	Type string // "internal" or "external"
}

func GetReWriteTarget(rootDir, currentPath string) (RouteResult, error) {
	zinConfig := filepath.Join(rootDir, "zin.config")

	// Case 1: file doesn't exist
	if _, err := os.Stat(zinConfig); os.IsNotExist(err) {
		return RouteResult{}, errors.New("rewrite config file not found")
	}

	file, err := os.Open(zinConfig)
	if err != nil {
		return RouteResult{}, err
	}
	defer file.Close()

	re := regexp.MustCompile(`<zin-rewrite\s+path="([^"]+)"\s+to="([^"]+)"\s*/>`)
	scanner := bufio.NewScanner(file)

	// Case 2: default to not configured
	foundRoute := false
	var result RouteResult

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if matches := re.FindStringSubmatch(line); len(matches) == 3 {
			path := matches[1]
			target := matches[2]

			if path == currentPath {
				foundRoute = true
				result.Path = target
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
					result.Type = "external"
				} else {
					result.Type = "internal"
					result.Path = filepath.Join(rootDir, target)
				}
				break
			}
		}
	}

	if !foundRoute {
		return RouteResult{}, errors.New("no matching route found")
	}

	return result, nil
}
