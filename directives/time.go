package directives

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"zin-engine/model"
	"zin-engine/utils"
)

var timeShortcuts = map[string]string{
	"YEAR":          "2006",
	"MONTH":         "01",
	"MONTH_NAME":    "January",
	"DAY":           "02",
	"DAY_NAME":      "Monday",
	"DATE":          "2006-01-02",
	"TIME":          "15:04:05",
	"TIME_12H":      "03:04:05 PM",
	"DATE_TIME":     "2006-01-02 15:04:05",
	"DATE_TIME_12H": "2006-01-02 03:04:05 PM",
	"WEEK":          "Monday (Week 02)",
	"WEEK_NO":       "02",
	"TODAY":         "2006-01-02",
	"NOW":           "2006-01-02T15:04:05Z07:00",
}

var (
	local                = time.Now().Location()
	zinTime              = regexp.MustCompile(`<zin-time\s*/?>`)
	zinTimeRegex         = regexp.MustCompile(`<zin-time(?:\s+format="([^"]*)")?\s*/?>`)
	zinTimeCustomUnits   = regexp.MustCompile(`^([+-]?)(\d+)(sec|min|hr|day|d|week|w|month|m|year|y)$`)
	fallbackZinTimeRegex = regexp.MustCompile(`<zin-time\b[^>]*\/?>`)
	zinTimeTagExample    = `<zin-time format="shortcut" />`
)

func TimeDirectives(content string, ctx *model.RequestContext) string {

	// No time directive, return unchanged
	if !strings.Contains(content, "<zin-time") {
		return content
	}

	// Set Time-Zone
	timeZone := utils.GetValue(ctx, "TIME_ZONE", "Local", true)
	if loc, err := time.LoadLocation(timeZone); err == nil {
		local = loc
	}

	// Insert Unix Time
	content = zinTime.ReplaceAllStringFunc(content, func(_ string) string {
		return strconv.FormatInt(time.Now().UnixMilli(), 10)
	})

	// Find all time-directives & insert real value
	content = zinTimeRegex.ReplaceAllStringFunc(content, func(match string) string {
		subMatches := zinTimeRegex.FindStringSubmatch(match)

		if len(subMatches) != 2 {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("Failed to parse the `zin-time` tag. Please ensure the syntax is correct. Expected format: %s.", zinTimeTagExample))
		}

		format := subMatches[1]
		if alias, ok := timeShortcuts[strings.ToUpper(format)]; ok {
			return time.Now().In(local).Format(alias)
		}

		// Parse relative time formats
		if parsed, err := parseRelativeTime(format); err == nil {
			return parsed.In(local).Format(time.RFC3339)
		}

		return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("The time shortcut %s is not recognized. Use a supported value like 'now', 'today', or a relative time such as '5m'.", format))

	})

	// Check if any time-tag still left in content
	content = fallbackZinTimeRegex.ReplaceAllStringFunc(content, func(tag string) string {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Failed to parse the `zin-time` tag. Please ensure the syntax is correct and use a supported value like 'now', 'today', or a relative time such as '5m'. Expected format: %s.", zinTimeTagExample))
	})

	return content
}

func parseRelativeTime(input string) (time.Time, error) {
	now := time.Now()

	// Normalize input
	input = strings.TrimSpace(strings.ToLower(input))

	// Regex for custom units with optional sign
	matches := zinTimeCustomUnits.FindStringSubmatch(input)
	if len(matches) != 4 {
		return time.Time{}, fmt.Errorf("unsupported relative time format: %s", input)
	}

	sign := matches[1] // "+" or "-" or ""
	value, _ := strconv.Atoi(matches[2])
	unit := matches[3]

	// Determine direction: default to future (+)
	multiplier := 1
	if sign == "-" {
		multiplier = -1
	}

	// Adjust time based on unit
	switch unit {
	case "sec":
		return now.Add(time.Duration(multiplier*value) * time.Second), nil
	case "min":
		return now.Add(time.Duration(multiplier*value) * time.Minute), nil
	case "hr":
		return now.Add(time.Duration(multiplier*value) * time.Hour), nil
	case "day", "d":
		return now.AddDate(0, 0, multiplier*value), nil
	case "week", "w":
		return now.AddDate(0, 0, 7*multiplier*value), nil
	case "month", "m":
		return now.AddDate(0, multiplier*value, 0), nil
	case "year", "y":
		return now.AddDate(multiplier*value, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported unit: %s", unit)
	}
}
