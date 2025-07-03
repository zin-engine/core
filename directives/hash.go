package directives

import (
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

var (
	zinHashRegex         = regexp.MustCompile(`<zin-hash\s+([^>]+)\s*/?>`)
	fallbackZinHashRegex = regexp.MustCompile(`<zin-hash\b[^>]*\/?>`)
	zinHashTagExample2   = `<zin-hash algo="sha256" value="your text" />`
	zinHashTagExample    = `<zin-hash algo="md5|sha1|sha256|sha512" value="text" [salt="..."] [output="base64|hex"] />`
)

func HashDirective(content string, ctx *model.RequestContext) string {

	// No hash directive, return unchanged
	if !strings.Contains(content, "<zin-hash") {
		return content
	}

	content = zinHashRegex.ReplaceAllStringFunc(content, func(tag string) string {
		attrs := parseAttributes(tag)

		algo := strings.ToLower(attrs["algo"])
		value := attrs["value"]
		salt := attrs["salt"]
		output := strings.ToLower(attrs["output"])

		if value == "" || algo == "" {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Missing required attribute: 'algo' and/or 'value'. Example: %s", zinHashTagExample2))
		}

		// Append or prepend salt (for now, simple append)
		input := value + salt
		result, err := utils.ComposeHash(input, algo, output)

		if err != nil {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), err.Error())
		}

		return result
	})

	// Check if any hash-tag still left in content
	content = fallbackZinHashRegex.ReplaceAllStringFunc(content, func(tag string) string {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Invalid  `zin-hash` tag — use format: %s, with required 'algo' and 'value' attributes.", zinHashTagExample))
	})

	return content
}

// Extracts key="value" pairs from a tag
func parseAttributes(tag string) map[string]string {
	attrRegex := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := attrRegex.FindAllStringSubmatch(tag, -1)

	attrs := make(map[string]string)
	for _, match := range matches {
		if len(match) == 3 {
			key := strings.ToLower(match[1])
			attrs[key] = match[2]
		}
	}
	return attrs
}
