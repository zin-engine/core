package directives

import (
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

func SetVarDirectives(content string, ctx *model.RequestContext) string {
	// No set directive, return unchanged
	if !strings.Contains(content, "<zin-set") {
		return content
	}

	// Regular expression to find all <zin-set ... /> tags
	re := regexp.MustCompile(`<zin-set\s+[^>]*\/>`)

	// Find all matches
	matches := re.FindAllString(content, -1)

	for _, match := range matches {
		// Extract key and value using regex
		keyRe := regexp.MustCompile(`key="(.*?)"`)
		valRe := regexp.MustCompile(`value="(.*?)"`)

		keyMatch := keyRe.FindStringSubmatch(match)
		valMatch := valRe.FindStringSubmatch(match)

		if keyMatch == nil || valMatch == nil {
			// Missing key or value
			reason := SetInlineError(fmt.Sprintf("Failed to load: %s", match), "missing key or value attribute")
			content = strings.Replace(content, match, reason, 1)
			continue
		}

		key := keyMatch[1]
		val := valMatch[1]

		// Save to context
		if ctx.LocalVar == nil {
			ctx.LocalVar = make(map[string]string)
		}
		ctx.LocalVar[key] = val

		// Remove tag from content (or replace with empty string)
		content = strings.Replace(content, match, "", 1)
	}

	return content
}

// Replace all vars with actual value
// Match patterns like {{key}}, {{key.sub-key}}, {{key || "apple"}}
func ReplaceVariables(content string, ctx *model.RequestContext) string {

	// No vars? No drama. Just return it like a boss.
	if !strings.Contains(content, "{{") {
		return content
	}

	re := regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		fullMatch := match[0]               // e.g., {{key || "apple"}}
		expr := strings.TrimSpace(match[1]) // inside of {{...}}, e.g., key || "apple"

		// Split by ||
		parts := strings.SplitN(expr, "||", 2)

		key := strings.TrimSpace(parts[0])
		defaultVal := "undefined"

		if len(parts) == 2 {
			// Anything after || is treated as a string
			defaultVal = strings.ReplaceAll(strings.TrimSpace(parts[1]), `"`, "")
		}

		// Determine if it's an env variable
		isEnv := strings.HasPrefix(key, "process.env.")
		if isEnv {
			key = strings.ToUpper(strings.ReplaceAll(key, "process.env.", ""))
		}

		// Get the value
		resolved := utils.GetValue(ctx, key, defaultVal, isEnv)

		// Replace the match in content
		content = strings.Replace(content, fullMatch, resolved, 1)
	}

	return content
}
