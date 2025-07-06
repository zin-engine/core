package directives

import (
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

func CryptDirective(content string, ctx *model.RequestContext) string {

	// No hash directive, return unchanged
	if !strings.Contains(content, "<zin-crypt") {
		return content
	}

	zinHashRegex := regexp.MustCompile(`<zin-crypt\s+([^>]+)\s*/?>`)
	content = zinHashRegex.ReplaceAllStringFunc(content, func(tag string) string {
		zinCryptAttr := utils.ExtractAttributesFromTag(tag)

		action, ok := zinCryptAttr["action"]
		if !ok {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), "Missing required attribute: 'action' in zin-crypt tag.")
		}

		action = strings.ToUpper(action)
		if action == "HASH" {
			value, err := composeHash(ctx, zinCryptAttr)
			if err != nil {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Unable to compose hash: %v", err))
			}

			return value
		}

		if action == "ENCRYPT" || action == "DECRYPT" || action == "ENC" || action == "DEC" {
			value, err := encodeDecodeValue(ctx, zinCryptAttr, action)
			if err != nil {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Unable to '%s' given data. Error: %v", strings.ToLower(action), err))
			}

			return value
		}

		return SetInlineError(fmt.Sprintf("Failed To Load: %s", tag), fmt.Sprintf("Given action for zin-crypt '%s' is not supported. You can performs actions like encrypt, decrypt & hash", action))
	})

	return content
}

func composeHash(ctx *model.RequestContext, attr map[string]string) (string, error) {

	algorithm, ok1 := attr["algorithm"]
	data, ok2 := attr["data"]
	format, ok3 := attr["format"]
	salt, ok4 := attr["salt"]

	if !ok1 || algorithm == "" {
		return "", fmt.Errorf("missing required params 'algorithm'. Use one of: md5, sha1, sha256, sha512")
	}
	if !ok2 || data == "" {
		return "", fmt.Errorf("missing or empty 'data' attribute. Provide the input string to be hashed")
	}

	if !ok3 || format == "" {
		format = "hex"
	}

	format = strings.ToLower(format)
	if format != "hex" && format != "base64" {
		return "", fmt.Errorf("incorrect 'format' attribute for output. Use one of: hex, base64")
	}

	if !ok4 {
		salt = "" // let's keep it blank by default
	}

	// Put variable-values if needed
	salt = ReplaceVariables(salt, ctx)
	data = ReplaceVariables(data, ctx)

	algorithm = strings.ToLower(algorithm)
	data = data + salt

	result, err := utils.ComposeHash(data, algorithm, format)
	if err != nil {
		return "", err
	}

	return result, nil
}

func encodeDecodeValue(ctx *model.RequestContext, attr map[string]string, action string) (string, error) {

	data, ok1 := attr["data"]
	key, ok2 := attr["key"]

	if !ok1 || data == "" {
		return "", fmt.Errorf("missing or empty 'data' attribute. It should be a non-empty string")
	}

	if !ok2 || key == "" {
		return "", fmt.Errorf("missing or empty 'key' attribute. It should be a non-empty string, ideally a strong random value")
	}

	// Put variable-values if needed
	key = ReplaceVariables(key, ctx)
	data = ReplaceVariables(data, ctx)

	// Check if action is to encrypt the data
	if action == "ENC" || action == "ENCRYPT" {
		encoded, err := utils.Encrypt(data, key)
		if err != nil {
			return "", err
		}

		return encoded, nil
	}

	decoded, err := utils.Decrypt(data, key)
	if err != nil {
		return "", err
	}
	return decoded, nil
}
