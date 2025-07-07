package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ModuleCache struct {
	lastModTime time.Time
	modules     map[string]string
	mu          sync.Mutex
}

var (
	mc       ModuleCache
	tagRegex = regexp.MustCompile(`<zin-([a-zA-Z0-9_-]+)[^>]*\/?>`)
)

func RunExternalModules(rootDir string, tag string) (string, error) {
	modules, _ := mc.ListModules(rootDir)

	matches := tagRegex.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return "", fmt.Errorf("unrecognized or malformed '%s' tag detected. It may be due to a typo, missing attributes, or use of an unsupported tag. For guidance, please refer to the official Zin tag documentation", tag)
	}

	tagName := "zin-" + matches[1]
	exePath, exists := modules[tagName]
	if !exists {
		return "", fmt.Errorf("unrecognized or malformed '%s' tag detected. It may be due to a typo, missing attributes, or use of an unsupported tag. For guidance, please refer to the official Zin tag documentation", tag)
	}

	// Run external formatter with full tag input
	cmd := exec.Command(exePath)
	cmd.Stdin = strings.NewReader(tag)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("an error occurred while running external module '%s'. Error: %v", tagName, err.Error())
	}

	return string(output), nil

}

func (mc *ModuleCache) ListModules(rootDir string) (map[string]string, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	dir := filepath.Join(rootDir, "modules")
	stat, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	// Only reScan if directory has changed
	if !stat.ModTime().After(mc.lastModTime) {
		return mc.modules, nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	newModules := make(map[string]string)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if strings.HasSuffix(name, ".exe") {
			tag := strings.TrimSuffix(name, ".exe")
			newModules[tag] = filepath.Join(dir, name)
		}
	}

	mc.lastModTime = stat.ModTime()
	mc.modules = newModules
	return newModules, nil
}
