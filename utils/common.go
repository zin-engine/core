package utils

import (
	"path/filepath"
	"strings"
)

func GetFilePathFromURI(path string) string {

	if path == "/" {
		path = "/index.html"
	}

	ext := filepath.Ext(path)
	if ext == "" {
		return path + ".html"
	}

	return path
}

func SanitizeContent(content string) string {
	content = strings.ReplaceAll(content, "<", "&lt;")
	return strings.ReplaceAll(content, ">", "&gt;")
}

func ReplaceContent(content string, target string, value string) string {
	content = strings.ReplaceAll(content, target, value)
	return content
}
