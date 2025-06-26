package directives

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
)

var (
	zinHashRegex         = regexp.MustCompile(`<zin-hash\s+([^>]+)\s*/?>`)
	fallbackZinHashRegex = regexp.MustCompile(`<zin-hash\b[^>]*\/?>`)
	zinHashTagExample2   = `<zin-hash algo="sha256" value="your text" />`
	zinHashTagExample    = `<zin-hash algo="md5|sha1|sha256|sha512" value="text" [salt="..."] [output="base64|hex"] />`
)

func HashDirective(content string, ctx *model.RequestContext) string {

	// No hash directive, return unchanged
	if !strings.Contains(content, "<zin-hash") {
		return content
	}

	content = zinHashRegex.ReplaceAllStringFunc(content, func(tag string) string {
		attrs := parseAttributes(tag)

		algo := strings.ToLower(attrs["algo"])
		value := attrs["value"]
		salt := attrs["salt"]
		output := strings.ToLower(attrs["output"])

		if value == "" || algo == "" {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Missing required attribute: 'algo' and/or 'value'. Example: %s", zinHashTagExample2))
		}

		// Append or prepend salt (for now, simple append)
		input := value + salt

		// Choose hashing algorithm
		var hash []byte
		switch algo {
		case "md5":
			h := md5.Sum([]byte(input))
			hash = h[:]
		case "sha1":
			h := sha256.Sum224([]byte(input))
			hash = h[:]
		case "sha256":
			h := sha256.Sum256([]byte(input))
			hash = h[:]
		case "sha512":
			h := sha512.Sum512([]byte(input))
			hash = h[:]
		default:
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), "Unsupported algorithm. Use one of: md5, sha1, sha256, sha512.")
		}

		// Default to hex if not specified or invalid
		var result string
		switch output {
		case "base64":
			result = base64.StdEncoding.EncodeToString(hash)
		default:
			result = hex.EncodeToString(hash)
		}

		return result
	})

	// Check if any hash-tag still left in content
	content = fallbackZinHashRegex.ReplaceAllStringFunc(content, func(tag string) string {
		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Invalid  `zin-hash` tag — use format: %s, with required 'algo' and 'value' attributes.", zinHashTagExample))
	})

	return content
}

// Extracts key="value" pairs from a tag
func parseAttributes(tag string) map[string]string {
	attrRegex := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := attrRegex.FindAllStringSubmatch(tag, -1)

	attrs := make(map[string]string)
	for _, match := range matches {
		if len(match) == 3 {
			key := strings.ToLower(match[1])
			attrs[key] = match[2]
		}
	}
	return attrs
}
