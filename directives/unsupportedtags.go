package directives

import (
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

func HighlightUnsupportedTags(content string, ctx *model.RequestContext) string {

	// No incorrect zin-tags, return unchanged
	if !strings.Contains(content, "<zin") {
		return content
	}

	rg := regexp.MustCompile(`<zin\b[^>]*\/?>`)
	matches := rg.FindAllString(content, -1)
	for _, tag := range matches {
		replace, err := utils.RunExternalModules(ctx.Root, tag)
		if err != nil {
			replace = SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Oops! %v.", err))
		}

		content = strings.Replace(content, tag, replace, 1)
	}

	return content
}
