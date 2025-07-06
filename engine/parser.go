package engine

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"zin-engine/utils"
)

var extractedTitle = ""

// ToDo: Cache pageContent to avoid template parsing on each request
func GetPageContent(root string, uri string, path string) (string, error) {
	page, err := utils.GetFileContent(path)
	if err != nil {
		return "", fmt.Errorf("content file not found: %v", err)
	}

	// Collect applicable templates from most specific to root
	extractedTitle = ""
	templates := collectTemplates(root, uri)

	// If no templates found, return raw content
	if len(templates) == 0 {
		return page, nil
	}

	// Loop though each template till <html>
	for i := 0; i < len(templates); i++ {
		if _, err := os.Stat(templates[i]); os.IsNotExist(err) {
			continue // Skip if the template file doesn't exist
		}

		tplBytes, err := os.ReadFile(templates[i])
		if err != nil {
			return "", fmt.Errorf("template read error: %v", err)
		}
		tplContent := string(tplBytes)

		// Cache & removed zin-page tags if present in template-content
		tplContent = ExtractTileFormZinPageTag(tplContent)

		// Check if this is the final wrapping template
		if strings.Contains(strings.ToLower(tplContent), "<html") {
			tpl, err := template.New("tpl").Parse(tplContent)
			if err != nil {
				return "", fmt.Errorf("template parse error: %v", err)
			}

			var rendered strings.Builder
			err = tpl.Execute(&rendered, map[string]interface{}{
				"children": template.HTML(page),
			})

			if err != nil {
				return "", fmt.Errorf("template execution error: %v", err)
			}

			// Done - This is the final HTML wrapper
			return InjectTitleFromZinPage(rendered.String(), uri), nil
		}

		// Embed and continue upward
		tpl, err := template.New("tpl").Parse(tplContent)
		if err != nil {
			return "", fmt.Errorf("template parse error: %v", err)
		}

		var rendered strings.Builder
		err = tpl.Execute(&rendered, map[string]interface{}{
			"children": template.HTML(page),
		})
		if err != nil {
			return "", fmt.Errorf("template execution error: %v", err)
		}

		page = rendered.String()
	}

	// Done
	return InjectTitleFromZinPage(page, uri), nil

}

// collectTemplates walks upward from requestPath to root and gathers existing template.html files.
func collectTemplates(rootDir string, requestPath string) []string {
	var templates []string
	pathParts := strings.Split(filepath.Clean(requestPath), string(os.PathSeparator))

	for i := len(pathParts); i >= 0; i-- {
		subPath := filepath.Join(pathParts[:i]...)
		templatePath := filepath.Join(rootDir, subPath, "template.html")
		if _, err := os.Stat(templatePath); err == nil {
			templates = append(templates, templatePath)
		}
	}
	return templates
}

// Page title
func InjectTitleFromZinPage(content string, path string) string {

	fmt.Printf("\n>> Page Title: %s", extractedTitle)
	// Fallback to <title>...</title>
	if extractedTitle == "" {
		reTitle := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
		match := reTitle.FindStringSubmatch(content)
		if len(match) >= 2 {
			extractedTitle = match[1]
		}
	}

	// If page-title is still blank then set current path as title
	if extractedTitle == "" {
		extractedTitle = path
	}

	// Replace or insert <title>
	reTitleTag := regexp.MustCompile(`(?i)<title>.*?</title>`)
	if reTitleTag.MatchString(content) {
		// Replace existing title
		content = reTitleTag.ReplaceAllString(content, "<title>"+extractedTitle+"</title>")
	} else {
		// Insert <title> inside <head>
		reHead := regexp.MustCompile(`(?i)<head[^>]*>`)
		if loc := reHead.FindStringIndex(content); loc != nil {
			// Insert title right after <head>
			insertPos := loc[1]
			content = content[:insertPos] + "\n<title>" + extractedTitle + "</title>" + content[insertPos:]
		} else {
			// No <head> found, fallback to prepending title
			content = "<title>" + extractedTitle + "</title>\n" + content
		}
	}

	return content
}

func ExtractTileFormZinPageTag(content string) string {
	if !strings.Contains(content, "<zin-page") {
		return content
	}

	reZin := regexp.MustCompile(`<zin-page\s+name=["']([^"']+)["']\s*/?>`)
	match := reZin.FindStringSubmatch(content)

	if len(match) >= 2 {
		extractedTitle = match[1]
		content = reZin.ReplaceAllString(content, "")
	}

	return content
}
