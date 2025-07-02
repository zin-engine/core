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
	"zin-engine/utils"
)

// Regex to match <zin-data src="..." as="..."/>
var (
	zinDataRegex = regexp.MustCompile(`<zin-data\s+src="([^"]+)"\s+as="([^"]+)"\s*/?>`)
	zinDataTag   = `<zin-data src="file://path/to/file.exe" into="varName" />`
)

// ProcessZinDataTags validates and rewrites <zin-data> tags
func DataDirectives(content string, ctx *model.RequestContext) string {
	// No data directive, return unchanged
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
		parts := strings.SplitN(src, "://", 2)

		// Check if src has operator defined
		if len(parts) != 2 {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Invalid 'src' value. It must start with a supported operator type such as 'file:', 'sql:', 'http:', 'https:', or 'sheets:'. Example: %s", zinDataTag))
		}

		// import data into var from local file
		if parts[0] == "file" {
			return importDataFromLocalFile(ctx, parts[1], varName, tag)
		}

		// import data from google-sheets using google visualization api
		if parts[0] == "sheets" {
			return importDataFromGoogleSheets(ctx, parts[1], varName, tag)
		}

		// import data from external api over http using google visualization api
		if parts[0] == "http" || parts[0] == "https" {
			return importDataFromExternalAPI(ctx, src, varName, tag)
		}

		// import data from mysql-database
		if parts[0] == "mysql" {
			return importDataFromMySQL(ctx, parts[1], varName, tag)
		}

		// Done
		return ""
	})
}

func importDataFromLocalFile(ctx *model.RequestContext, src string, varName string, tag string) string {

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

	return ""
}

func importDataFromGoogleSheets(ctx *model.RequestContext, src string, varName string, tag string) string {

	// Replace all vars with actual values
	src = ReplaceVariables(src, ctx)

	// Parse src to get sheet Name, Id & query separately
	result, err := utils.ParseSheetQuery(src)
	if err != nil {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Error: %v", err))

	}

	// Encode it to run over http
	result.Query = utils.EncodeSheetsQuery(result.Query)
	if result.Query == "" {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), "Error: You can only run 'SELECT' query to fetch data from google sheets.")
	}

	// Fetch data from sheets
	response := utils.FetchDataFromSheets(result.SheetID, result.SheetName, result.Query)
	if strings.Contains(response, "Error: ") {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), response)
	}

	// Set data to context list for later use
	err = utils.CsvToContextList(ctx, varName, response)
	if err != nil {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), err.Error())
	}

	// Done
	return ""
}

func importDataFromExternalAPI(ctx *model.RequestContext, src string, varName string, tag string) string {
	// Replace all vars with actual values
	src = ReplaceVariables(src, ctx)

	// Call given endpoint to fetch data
	err := utils.Get(ctx, src, varName)
	if err != nil {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), err.Error())
	}

	return ""
}

func importDataFromMySQL(ctx *model.RequestContext, src string, varName string, tag string) string {
	// Replace all vars with actual values
	src = ReplaceVariables(src, ctx)

	// Call given endpoint to fetch data
	err := utils.RunQuery(ctx, src, varName)
	if err != nil {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), err.Error())
	}

	return ""
}
