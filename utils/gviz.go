package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"zin-engine/model"
)

func ParseSheetQuery(query string) (*model.SheetQueryResult, error) {
	// Case-insensitive regex to match FROM clause: FROM SheetId.SheetName
	re := regexp.MustCompile(`(?i)\s+from\s+([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid query format, expected FROM SheetId.SheetName")
	}

	sheetID := matches[1]
	sheetName := matches[2]

	// Remove the FROM clause
	fromClause := matches[0] // full matched " from SheetID.SheetName" (case-insensitive)
	cleanedQuery := strings.Replace(query, fromClause, "", 1)

	return &model.SheetQueryResult{
		SheetID:   sheetID,
		SheetName: sheetName,
		Query:     strings.TrimSpace(cleanedQuery),
	}, nil
}

func EncodeSheetsQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	if !strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") {
		return ""
	}
	return url.QueryEscape(trimmed)
}

func FetchDataFromSheets(sheetId string, sheetName string, query string) string {
	baseURL := "https://docs.google.com/spreadsheets/d/%s/gviz/tq?tqx=out:csv&sheet=%s&tq=%s"
	finalURL := fmt.Sprintf(baseURL, sheetId, sheetName, query)

	resp, err := http.Get(finalURL)
	if err != nil {
		return fmt.Sprintf("Error: Failed to fetch data using GViz API. %s", err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Sprintf("Error: Failed to fetch data using GViz API. Received a %d status code.", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error: Failed to read response body. %s", err.Error())
	}

	result := string(body)

	// Check for any potential errors in the GViz response
	if strings.Contains(strings.ToLower(result), `"status":"error"`) {
		return "Error: Google Spreadsheet returned an error while processing the query"
	}

	return result
}

// Parse CSV & keep it in context
func CsvToContextList(ctx *model.RequestContext, varName string, csvStr string) error {
	reader := csv.NewReader(strings.NewReader(csvStr))
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("CSV parsing failed: %v", err)
	}

	if len(records) < 1 {
		return fmt.Errorf("CSV is empty or malformed")
	}

	headers := records[0]
	rows := make([]any, 0, len(records)-1)

	for _, record := range records[1:] {
		row := make(map[string]any)
		for i, val := range record {
			if i < len(headers) {
				row[headers[i]] = val
			}
		}
		rows = append(rows, row)
	}

	ctx.CustomVar.LIST[varName] = rows
	return nil
}
