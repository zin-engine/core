package directives

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"zin-engine/model"
)

func TimeDirectives(content string, ctx *model.RequestContext) string {
	// No time directive, return unchanged
	if !strings.Contains(content, "<zin-time") {
		return content
	}

	// Match all self-closing zin-time tags
	re := regexp.MustCompile(`<zin-time([^>]*)\/>`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		fullTag := match[0]
		attrStr := match[1]

		attrs := parseTimeAttributes(attrStr)

		when := attrs["when"]
		view := attrs["view"]
		tz := attrs["tz"]

		// Defaults
		if when == "" {
			when = "now"
		}
		if view == "" {
			view = "datetime"
		}

		// Parse time
		t, err := parseWhen(when)
		if err != nil {
			replacement := SetInlineError(fmt.Sprintf("Failed to load: %s", fullTag), fmt.Sprintf("Invalid when: %v", err))
			content = strings.Replace(content, fullTag, replacement, 1)
			continue
		}

		// Handle timezone
		if tz != "" {
			loc, err := time.LoadLocation(tz)
			if err != nil {
				replacement := SetInlineError(fmt.Sprintf("Failed to load: %s", fullTag), fmt.Sprintf("Invalid tz: %v", err))
				content = strings.Replace(content, fullTag, replacement, 1)
				continue
			}
			t = t.In(loc)
		}

		// Format view
		formatted, err := formatView(t, view)
		if err != nil {
			replacement := SetInlineError(fmt.Sprintf("Failed to load: %s", fullTag), fmt.Sprintf("Invalid view: %v", err))
			content = strings.Replace(content, fullTag, replacement, 1)
			continue
		}

		// Replace tag with formatted time
		content = strings.Replace(content, fullTag, formatted, 1)
	}

	return content
}

func parseWhen(input string) (time.Time, error) {
	input = strings.TrimSpace(input)

	// Relative patterns: now +5d, today -3w, etc.
	parts := strings.Fields(input)
	var base time.Time

	// Handle base (now/today/yesterday/tomorrow/date)
	switch parts[0] {
	case "now":
		base = time.Now()
	case "today":
		base = time.Now().Truncate(24 * time.Hour)
	case "yesterday":
		base = time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	case "tomorrow":
		base = time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	default:
		parts[0] = strings.ReplaceAll(parts[0], "T", " ")
		var t time.Time
		var err error

		layouts := []string{
			time.RFC3339,          // 2006-01-02T15:04:05Z07:00
			"2006-01-02",          // date only
			"2006-01-02 15:04",    // space-separated datetime
			"2006-01-02 15:04:05", // space-separated full datetime
		}

		for _, layout := range layouts {
			t, err = time.Parse(layout, parts[0])
			if err == nil {
				break
			}
		}

		if err != nil {
			return time.Time{}, fmt.Errorf("invalid base time, it only supports date, datetime & full-datetime formats")
		}

		base = t
	}

	// Always return base (even if no parts[1..n] exist)
	if len(parts) == 1 {
		return base, nil
	}

	// Apply math like +5d
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "@startOf:") || strings.HasPrefix(part, "@endOf:") {
			base = applyStartOrEnd(part, base)
			continue
		}
		if len(part) < 2 {
			continue
		}
		sign := part[0]
		amountUnit := part[1:]
		n := 0
		for i, c := range amountUnit {
			if c < '0' || c > '9' {
				nPart := amountUnit[:i]
				unit := amountUnit[i:]
				n, _ = strconv.Atoi(nPart)
				if sign == '-' {
					n = -n
				}
				base = applyOffset(base, n, unit)
				break
			}
		}
	}

	return base, nil
}

func applyOffset(t time.Time, val int, unit string) time.Time {
	switch unit {
	case "d":
		return t.AddDate(0, 0, val)
	case "w":
		return t.AddDate(0, 0, 7*val)
	case "m":
		return t.AddDate(0, val, 0)
	case "y":
		return t.AddDate(val, 0, 0)
	case "h":
		return t.Add(time.Duration(val) * time.Hour)
	case "min":
		return t.Add(time.Duration(val) * time.Minute)
	case "s":
		return t.Add(time.Duration(val) * time.Second)
	default:
		return t
	}
}

func applyStartOrEnd(part string, t time.Time) time.Time {
	switch part {
	case "@startOf:month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "@endOf:month":
		firstOfNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
		return firstOfNextMonth.Add(-time.Nanosecond)
	case "@startOf:year":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	case "@endOf:year":
		return time.Date(t.Year()+1, 1, 1, 0, 0, 0, -1, t.Location())
	case "@startOf:week":
		offset := int(t.Weekday())
		return t.AddDate(0, 0, -offset).Truncate(24 * time.Hour)
	case "@endOf:week":
		offset := 6 - int(t.Weekday())
		return t.AddDate(0, 0, offset).Truncate(24 * time.Hour)
	default:
		return t
	}
}

func parseTimeAttributes(attrStr string) map[string]string {
	attrs := make(map[string]string)
	re := regexp.MustCompile(`(\w+)="([^"]*)"`)
	pairs := re.FindAllStringSubmatch(attrStr, -1)
	for _, pair := range pairs {
		key := strings.ToLower(pair[1])
		val := pair[2]
		attrs[key] = val
	}
	return attrs
}

func formatView(t time.Time, view string) (string, error) {
	switch view {
	case "date":
		return t.Format("2006-01-02"), nil
	case "time":
		return t.Format("15:04"), nil
	case "time:12":
		return t.Format("3:04 PM"), nil
	case "datetime":
		return t.Format("2006-01-02 15:04"), nil
	case "datetime:12":
		return t.Format("Jan 2, 2006, 3:04 PM"), nil
	case "day":
		return strconv.Itoa(t.Day()), nil
	case "dayname":
		return t.Weekday().String(), nil
	case "month":
		return fmt.Sprintf("%02d", t.Month()), nil
	case "monthname":
		return t.Month().String(), nil
	case "year":
		return strconv.Itoa(t.Year()), nil
	case "week":
		_, week := t.ISOWeek()
		return strconv.Itoa(week), nil
	case "unix":
		return strconv.FormatInt(t.UnixMilli(), 10), nil
	case "unix:sec":
		return strconv.FormatInt(t.Unix(), 10), nil
	default:
		return "", fmt.Errorf("unsupported view")
	}
}
