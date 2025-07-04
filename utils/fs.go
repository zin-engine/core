package utils

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
)

func GetFileContent(path string) (string, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read file: %s. %v", path, err)
	}
	return string(f), nil
}

func GetFileFromExePath(filePath string) string {
	content := ""

	// Get exe dir path
	exePath, err := os.Executable()
	if err != nil {
		return content
	}

	// Get final file path
	filePath = filepath.Join(filepath.Dir(exePath), "public", "assets", filePath)
	content, err = GetFileContent(filePath)
	if err != nil {
		return content
	}

	return content
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

func GetFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".js":
		return "JS"
	case ".md":
		return "MD"
	case ".txt":
		return "TXT"
	default:
		return ""
	}
}

func ParseMdToHTML(content string) string {
	var buf strings.Builder
	if err := goldmark.Convert([]byte(content), &buf); err != nil {
		return content
	}

	return buf.String()
}

func GetExeAssetPath(file string) string {
	exePath, err := os.Executable()
	if err != nil {
		return file
	}

	file = filepath.Join(filepath.Dir(exePath), "public", "assets", file)
	return file
}
