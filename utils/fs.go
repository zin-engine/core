package utils

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func GetFileContent(path string) (string, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("template read error: %v", err)
	}
	return string(f), nil
}

func GetStatusCodeFileContent(status int, path string, content string) string {
	fileName := strconv.Itoa(status) + ".html"
	filePath := filepath.Join(path, fileName)

	// Check if the file exists at the given path,
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		exePath, err := os.Executable()
		if err != nil {
			return content
		}
		// Set path to default from the executable root under /public/views/
		filePath = filepath.Join(filepath.Dir(exePath), "public", "views", fileName)

	}

	fileContent, err := GetFileContent(filePath)
	if err != nil {
		fmt.Println(err.Error(), "utils/fs.go")
		return content
	}

	// Replace the file content to show error on client
	fileContent = strings.Replace(fileContent, "{{.ERROR_SUMMARY}}", content, 1)

	return fileContent
}

func GetFaviconIconPath(rootDir string, path string) string {
	// Check if favicon exists at given path,
	path = filepath.Join(rootDir, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exePath, err := os.Executable()
		if err != nil {
			return path
		}

		// Use the default path from the zin-exe root
		path = filepath.Join(filepath.Dir(exePath), "public", "favicon.ico")
	}

	return path
}

func GetMineTypeFromPath(path string) string {

	mimeType := mime.TypeByExtension(filepath.Ext(path))

	// Change mimeType for md files to text/plain
	if strings.HasSuffix(path, ".md") {
		mimeType = "text/plain"
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return mimeType
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}
