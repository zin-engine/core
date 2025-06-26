package directives

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

const maxIncludeDepth = 5

var (
	zinIncludeRegex         = regexp.MustCompile(`<zin-include\s+file="([^"]+)"(?:\s+type="([^"]+)")?\s*/?>`)
	fallbackZinIncludeRegex = regexp.MustCompile(`<zin-include\b[^>]*\/?>`)
)

func IncludeDirective(content string, ctx *model.RequestContext) string {

	// No include directive, return unchanged
	if !strings.Contains(content, "<zin-include") {
		return content
	}

	// Included file-content in raw-content
	content = processZinIncludes(ctx, content, 0, nil)

	// Remove malformed, incomplete, or unhandled zin-includes
	content = removeUnparsedZinIncludes(content)

	return content
}

func processZinIncludes(ctx *model.RequestContext, input string, depth int, seen map[string]bool) string {
	if depth > maxIncludeDepth {
		return input // prevent infinite recursion
	}
	if seen == nil {
		seen = make(map[string]bool)
	}

	return zinIncludeRegex.ReplaceAllStringFunc(input, func(match string) string {
		parts := zinIncludeRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		includedFile := parts[1]
		formatter := parts[2]
		fileType := utils.GetFileType(includedFile)

		if fileType == "" {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("Unsupported file '%s' type. Only .html, .css, .js, .md, and .txt files are allowed.", includedFile))
		}

		// Prevent circular includes
		uniqueKey := filepath.Join(ctx.Root, includedFile)
		if seen[uniqueKey] {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("File '%s' is already included at recursion depth %d", includedFile, depth))
		}
		seen[uniqueKey] = true

		content, err := utils.GetFileContent(uniqueKey)
		if err != nil {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("Failed to read file '%s': %v", includedFile, err))
		}

		// Recursively process includes in the included file
		content = processZinIncludes(ctx, content, depth+1, seen)

		if formatter == "RAW_CONTENT" {
			if fileType == "HTML" {
				return utils.SanitizeHTML(content)
			}
			return content
		}

		switch fileType {
		case "HTML":
			return content
		case "CSS":
			return "<style>\n" + content + "\n</style>"
		case "JS":
			return "<script type=\"text/javascript\">\n" + content + "\n</script>"
		case "TXT":
			return content
		case "MD":
			return utils.ParseMdToHTML(content)
		default:
			return ""
		}
	})
}

func removeUnparsedZinIncludes(input string) string {
	warning := ""
	matches := fallbackZinIncludeRegex.FindAllString(input, -1)
	for _, tag := range matches {
		warning = SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Removed malformed <zin-include>: %s", tag))
	}
	return fallbackZinIncludeRegex.ReplaceAllString(input, warning)
}
