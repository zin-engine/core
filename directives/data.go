package directives

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"zin-engine/model"
)

// Regex to match <zin-data src="..." as="..."/>
var (
	zinDataRegex = regexp.MustCompile(`<zin-data\s+src="([^"]+)"\s+as="([^"]+)"\s*/?>`)
	zinDataTag   = `<zin-data src="path/to/file.exe" into="varName" />`
)

// ProcessZinDataTags validates and rewrites <zin-data> tags
func DataDirectives(content string, ctx *model.RequestContext) string {
	// No hash directive, return unchanged
	if !strings.Contains(content, "<zin-data") {
		return content
	}

	// Change all
	return zinDataRegex.ReplaceAllStringFunc(content, func(tag string) string {
		matches := zinDataRegex.FindStringSubmatch(tag)
		if len(matches) < 3 {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Invalid zin-data tag format. Example: %s", zinDataTag))
		}

		src := matches[1]
		varName := matches[2]
		fullPath := filepath.Join(ctx.Root, src)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Data source file not found (%s)", src))
		}

		// Validate file extension
		ext := strings.ToLower(filepath.Ext(src))
		if ext != ".json" && ext != ".csv" {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Only .json or .csv files are supported for data loading. Invalid file: %s", src))
		}

		// Extract data from file
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Failed to read contents of file: %s. Error: %v", fullPath, err))
		}

		// Set data into var
		var parsed interface{}

		if ext == ".json" {
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Failed to parse JSON content of file: %s. Error: %v", fullPath, err))
			}
		} else if ext == ".csv" {
			reader := csv.NewReader(strings.NewReader(string(data)))
			records, err := reader.ReadAll()
			if err != nil || len(records) < 1 {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Failed to parse CSV content of file: %s. Error: %v", fullPath, err))
			}

			headers := records[0]
			var arr []map[string]any
			for _, row := range records[1:] {
				obj := make(map[string]any)
				for i, cell := range row {
					if i < len(headers) {
						obj[headers[i]] = cell
					}
				}
				arr = append(arr, obj)
			}
			parsed = arr
		}

		// Store parsed data if varName is not already set
		switch v := parsed.(type) {
		case map[string]any:
			if ctx.CustomVar.JSON != nil {
				if _, exists := ctx.CustomVar.JSON[varName]; !exists {
					ctx.CustomVar.JSON[varName] = v
				}
			}
		case []any:
			if ctx.CustomVar.LIST != nil {
				if _, exists := ctx.CustomVar.LIST[varName]; !exists {
					ctx.CustomVar.LIST[varName] = v
				}
			}
		default:
			// fallback for other valid JSON structures (like []map[string]any)
			if arr, ok := v.([]map[string]any); ok {
				if ctx.CustomVar.LIST != nil {
					if _, exists := ctx.CustomVar.LIST[varName]; !exists {
						// Convert to []any
						out := make([]any, len(arr))
						for i := range arr {
							out[i] = arr[i]
						}
						ctx.CustomVar.LIST[varName] = out
					}
				}
			}
		}

		// Done
		return ""
	})
}
