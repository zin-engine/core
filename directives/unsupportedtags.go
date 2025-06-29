package directives

import (
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
)

func HighlightUnsupportedTags(content string, ctx *model.RequestContext) string {

	// No incorrect zin-tags, return unchanged
	if !strings.Contains(content, "<zin") {
		return content
	}

	rg := regexp.MustCompile(`<zin\b[^>]*\/?>`)
	matches := rg.FindAllString(content, -1)
	for _, tag := range matches {
		replace := SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Unrecognized or malformed %s tag detected. It may be due to a typo, missing attributes, or use of an unsupported tag. For guidance, please refer to the official Zin tag documentation.", tag))
		content = strings.Replace(content, tag, replace, 1)
	}

	return content
}
