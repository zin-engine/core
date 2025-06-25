package utils

import (
	"fmt"
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

func ComposeInlineErrorContent(title string, content string, path string) string {
	content = fmt.Sprintf(`<span class="zin-engine inline-error" data-source="%s" data-summary="%s">%s</span>`, path, SanitizeHTML(content), SanitizeHTML(title))
	return content
}

func InjectZinScriptAndStyle(content string) string {
	content = ReplaceContent(content, "</head>", `<script src="/zin-assets/engine.js"></script><link rel="stylesheet" href="/zin-assets/engine.css"></head>`)
	return content
}
