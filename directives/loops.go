package directives

import (
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
)

var (
	repeatTagRegex   = regexp.MustCompile(`(?s)<zin-repeat\s+for="([^"]+)">(.*?)</zin-repeat>`)
	variableRegex    = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)(\s*\|\|\s*(.*?))?\s*}}`)
	repeatTagExample = `<zin-repeat for="key"><p>Name: {{ name }} </p></zin-repeat>`
)

// LoopDirective processes zin-repeat directives in the input content
func LoopDirectives(content string, ctx *model.RequestContext) string {
	// No hash directive, return unchanged
	if !strings.Contains(content, "<zin-repeat") {
		return content
	}

	// Find & replace all zin-repeat tags
	return repeatTagRegex.ReplaceAllStringFunc(content, func(fullMatch string) string {
		// Extract the for variable and inner HTML
		matches := repeatTagRegex.FindStringSubmatch(fullMatch)
		if len(matches) < 3 {
			return SetInlineError("Failed To Load: <zin-repeat ... > ... </zin-repeat>", fmt.Sprintf("The <zin-repeat> tag is invalid. It must contain a 'for' attribute referencing a predefined list variable, and child elements to repeat. Example: %s", repeatTagExample))
		}

		varName := matches[1]
		innerHTML := matches[2]

		// Access ctx.CustomVar.LIST[varName]
		items, ok := ctx.CustomVar.LIST[varName]
		if !ok {
			return SetInlineError("Failed To Load: <zin-repeat ... > ... </zin-repeat>", fmt.Sprintf(`Variable '%s' not found or is not iterable.`, varName))
		}

		var builder strings.Builder

		// Loop through each item in the array
		for _, item := range items {
			obj, isMap := item.(map[string]interface{})
			if !isMap {
				builder.WriteString(`not-an-object`)
				continue
			}

			// Replace {{key}} or {{key || "default"}} inside the innerHTML
			parsed := variableRegex.ReplaceAllStringFunc(innerHTML, func(varExpr string) string {
				subMatches := variableRegex.FindStringSubmatch(varExpr)
				if len(subMatches) < 2 {
					return "undefined"
				}

				key := subMatches[1]
				defaultVal := "undefined"
				if len(subMatches) >= 4 && strings.TrimSpace(subMatches[3]) != "" {
					defaultVal = strings.Trim(subMatches[3], `"`)
				}

				val, exists := obj[key]
				if !exists {
					return defaultVal
				}
				return fmt.Sprintf("%v", val)
			})

			builder.WriteString(parsed)
		}

		return builder.String()
	})
}
