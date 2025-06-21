package engine

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"zin-engine/utils"
)

// ToDo: Cache pageContent to avoid template parsing on each request
func GetPageContent(root string, uri string, path string) (string, error) {
	page, err := utils.GetFileContent(path)
	if err != nil {
		return "", fmt.Errorf("content file not found: %v", err)
	}

	// Collect applicable templates from most specific to root
	templates := collectTemplates(root, path)

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
			return rendered.String(), nil
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

	return page, nil

}

// collectTemplates walks upward from requestPath to root and gathers existing template.html files.
func collectTemplates(rootDir, requestPath string) []string {
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
