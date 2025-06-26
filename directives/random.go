package directives

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"zin-engine/model"
)

var (
	zinRandomRegex         = regexp.MustCompile(`<zin-random(?:\s+type="([^"]*)")?(?:\s+len="([^"]*)")?\s*/?>`)
	fallbackZinRandomRegex = regexp.MustCompile(`<zin-random\b[^>]*\/?>`)
	seededRand             = rand.New(rand.NewSource(time.Now().UnixNano()))
	zinRandomTagExample    = `<zin-random type="int|string|mix|special" length="10" />`
)

func RandomDirective(content string, ctx *model.RequestContext) string {

	// No random directive, return unchanged
	if !strings.Contains(content, "<zin-random") {
		return content
	}

	// Replace all matches
	content = zinRandomRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := zinRandomRegex.FindStringSubmatch(match)

		randType := "ANY"
		length := 5

		if len(submatches) >= 2 && submatches[1] != "" {
			randType = strings.ToUpper(submatches[1])
		}

		if len(submatches) >= 3 && submatches[2] != "" {
			if _length, err := strconv.Atoi(submatches[2]); err == nil {
				length = _length
			}
		}

		// Call your custom generator
		return GenerateRandom(randType, length)
	})

	// Check if any random-tag still left in content
	content = fallbackZinRandomRegex.ReplaceAllStringFunc(content, func(tag string) string {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Invalid  `zin-random` tag — use format: %s, with optional, single-use 'type' and 'length' attributes.", zinRandomTagExample))
	})

	return content
}

func GenerateRandom(kind string, length int) string {
	var charset string

	switch strings.ToUpper(kind) {
	case "INT", "I":
		charset = "0123456789"
	case "STRING", "STR", "S":
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case "MIXED", "MIX", "MX":
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	case "SPECIAL", "X":
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	default:
		charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	}

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
