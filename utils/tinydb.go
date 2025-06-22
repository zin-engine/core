package utils

import (
	"fmt"
	"strconv"
	"strings"
	"zin-engine/model"
)

func SetValue(ctx *model.RequestContext, key string, value string) {
	ctx.LocalVar[key] = value
}

func GetValue(ctx *model.RequestContext, key string, defaultValue string, includeEnv bool) string {

	// Find key in default vars
	switch key {
	case "ClientIp":
		return ctx.ClientIp
	case "Method":
		return ctx.Method
	case "Host":
		return ctx.Host
	case "Path":
		return ctx.Path
	}

	// Find key in LocalVar
	if val, ok := ctx.LocalVar[key]; ok {
		return val
	}

	// Find key in CustomVar.Raw
	if val, ok := ctx.CustomVar.Raw[key]; ok {
		return val
	}

	// Find key in query-params of current request
	if val, ok := ctx.Query[key]; ok {
		return val[0]
	}

	// If env allowed find key in env too
	if includeEnv {
		if val, ok := ctx.ENV[key]; ok {
			return val
		}
	}

	// Handle dot or index notation (e.g. user.name or users[0].email)
	parts := parseKeyParts(key)
	if len(parts) == 0 {
		return defaultValue
	}

	// Split keys into root & rest
	root := parts[0]
	rest := parts[1:]

	// Let's find the root in CustomVar.JSON
	if data, ok := ctx.CustomVar.JSON[root]; ok {
		val := resolveJSON(data, rest)
		if val != nil {
			return fmt.Sprintf("%v", val)
		}
	}

	// Now let's check CustomVar.LIST for keys like users[0].name
	if list, ok := ctx.CustomVar.LIST[root]; ok && len(rest) > 0 {
		indexStr := rest[0]
		if strings.HasPrefix(indexStr, "[") && strings.HasSuffix(indexStr, "]") {
			indexStr = strings.Trim(indexStr, "[]")
			if idx, err := strconv.Atoi(indexStr); err == nil && idx >= 0 && idx < len(list) {
				if len(rest) == 1 {
					return fmt.Sprintf("%v", list[idx])
				}
				if m, ok := list[idx].(map[string]interface{}); ok {
					val := resolveJSON(m, rest[1:])
					if val != nil {
						return fmt.Sprintf("%v", val)
					}
				}
			}
		}
	}

	// Hmm.. as key not found. Let's return the default value
	return defaultValue
}

// Split key like user.name or users[1].name into usable parts
func parseKeyParts(key string) []string {
	// Replace [n] with .[n] so we can split cleanly
	key = strings.ReplaceAll(key, "[", ".[")
	return strings.Split(key, ".")
}

// Walk nested maps based on path like name, profile.name etc.
func resolveJSON(data map[string]interface{}, path []string) interface{} {
	var current any = data
	for _, part := range path {
		if currentMap, ok := current.(map[string]interface{}); ok {
			part = strings.Trim(part, "[]")
			if val, ok := currentMap[part]; ok {
				current = val
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return current
}
