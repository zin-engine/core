package utils

import (
	"html"
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

func SanitizeHTML(content string) string {
	return html.EscapeString(content)
}

func ReplaceContent(content string, target string, value string) string {
	content = strings.ReplaceAll(content, target, value)
	return content
}
