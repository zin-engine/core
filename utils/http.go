package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"zin-engine/model"
)

func Get(ctx *model.RequestContext, url string, varName string) error {

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching URL: %s Error:%v", url, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to load GET \"%s\": Status Code %d: Failed to fetch data", url, resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response content.  %v", err)
	}

	// Set data according to response content
	// JSON
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err == nil {
			ctx.CustomVar.JSON[varName] = obj
		} else {
			// If not a map, try as Array (list)
			var arr []any
			if err := json.Unmarshal(body, &arr); err == nil {
				if ctx.CustomVar.LIST == nil {
					ctx.CustomVar.LIST = make(map[string][]any)
				}
				ctx.CustomVar.LIST[varName] = arr
			}
		}

		return nil
	}

	// CSV
	content := string(body)
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/csv") {
		return CsvToContextList(ctx, varName, content)
	}

	// Consider all other as Text
	ctx.CustomVar.Raw[varName] = content

	return nil
}
