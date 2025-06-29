package directives

import (
	"zin-engine/model"
	"zin-engine/utils"
)

type Directive func(string, *model.RequestContext) string

var (
	currentFilePath  string
	showInlineErrors string
)

func ParseAndApply(content string, ctx *model.RequestContext) string {

	// Config
	currentFilePath = ctx.ContentSource
	showInlineErrors = utils.GetValue(ctx, "SHOW_ERRORS", "OFF", true)

	// List of directives to apply
	directives := []Directive{
		IncludeDirective,
		SetVarDirectives,
		TimeDirectives,
		RandomDirective,
		HashDirective,
		ReplaceVariables,
		HighlightUnsupportedTags,
	}

	// Apply each directive in order, stop if errors found
	for _, directive := range directives {
		if len(ctx.ServerError) > 0 {
			break
		}
		content = directive(content, ctx)
	}

	return content
}

func SetServerError(ctx *model.RequestContext, title string, code string, reason string) {
	ctx.ServerError["title"] = title
	ctx.ServerError["code"] = code
	ctx.ServerError["reason"] = reason
}

func SetInlineError(title string, content string) string {
	if showInlineErrors != "ON" {
		return ""
	}
	return utils.ComposeInlineErrorContent(title, content, currentFilePath)
}
