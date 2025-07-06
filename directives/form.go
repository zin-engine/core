package directives

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

func FormDirective(content string, ctx *model.RequestContext) string {

	// No form directive, return unchanged
	if !strings.Contains(content, "<zin-form") {
		return content
	}

	// Regex to match <zin-form ...>...</zin-form>
	re := regexp.MustCompile(`(?s)<zin-form\s+([^>]*)>(.*?)</zin-form>`)

	content = re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract attributes and inner content
		subMatches := re.FindStringSubmatch(match)
		if len(subMatches) < 3 {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), "Given zin-form isn't configured well, missing attributes or content.")
		}

		attrString := subMatches[1]
		innerContent := subMatches[2]
		elmSuffix := ""
		zinFormAttr := utils.ExtractAttributesFromTag(attrString)
		zinFormId := GenerateRandom("MIXED", 32)

		var formAttrs []string
		formAttrs = append(formAttrs, `action="/zin-form"`)
		formAttrs = append(formAttrs, `onsubmit="zinFormSubmitHandler(event)"`)
		formAttrs = append(formAttrs, fmt.Sprintf(`id="%s"`, zinFormId))

		// Verify & set form action
		zinFormSession := ""
		if zinFormAction, ok := zinFormAttr["action"]; ok {
			zinFormAction = ReplaceVariables(zinFormAction, ctx)
			zinFormSession = zinFormAction

			if !strings.HasPrefix(zinFormAction, "http") {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("For action '%s' is not valid you can either use http(s) to submit form data", zinFormAction))
			}
		} else {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), "You haven't specified the form-action. It must be a http endpoint.")
		}

		// Set callback
		if callback, ok := zinFormAttr["callback"]; ok {
			formAttrs = append(formAttrs, fmt.Sprintf(`data-callback="%s"`, callback))
		}

		// Set Name of this form to be later used as source
		if formName, ok := zinFormAttr["name"]; ok {
			formAttrs = append(formAttrs, fmt.Sprintf(`data-source="%s"`, formName))
		} else {
			formAttrs = append(formAttrs, fmt.Sprintf(`data-source="form@%s"`, ctx.Host))
		}

		// Check if form is captcha enabled
		captchaProvider := "NONE"
		if val, ok := zinFormAttr["captcha"]; ok {
			val = strings.ToUpper(val)
			if val != "GOOGLE" {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), "Unsupported captcha provider. Currently we only support Google Recaptcha V3.")
			}

			// Check if configured properly
			siteKey := verifyAndGetGoogleCaptchaSiteKey(ctx)
			if siteKey == "" {
				return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), "Google recaptcha credentials not present on .env file")
			}

			captchaProvider = "GOOGLE"
			formAttrs = append(formAttrs, fmt.Sprintf(`data-captcha="%s"`, siteKey))
			elmSuffix += fmt.Sprintf(`<script src="https://www.google.com/recaptcha/api.js?render=%s"></script>`, siteKey)
		}

		// Extract validators from the form data-fields
		jsonOutput, err := ExtractAttributes(innerContent)
		if err != nil {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("Failed to parse form input validators, %v", err))
		}

		// Compose session-token
		zinFormSession += "::" + zinFormId + "::" + ctx.ClientIp + "::" + captchaProvider + "::" + jsonOutput
		token, err := utils.Encrypt(zinFormSession, zinFormId)
		if err != nil {
			return SetInlineError(fmt.Sprintf("Failed To Load: %s", match), fmt.Sprintf("Failed to generate form submission token, %v", err))
		}

		formAttrs = append(formAttrs, fmt.Sprintf(`data-session="%s"`, token))

		// Only if controller is not added before include it in main content
		formSubmitHandler := utils.GetFileFromExePath("form.js")
		elmSuffix += fmt.Sprintf(`<script>%s</script>`, formSubmitHandler)

		// Compose final tag
		formTag := "<form " + strings.Join(formAttrs, " ") + ">"
		return formTag + innerContent + "</form>" + elmSuffix
	})

	return content
}

func verifyAndGetGoogleCaptchaSiteKey(ctx *model.RequestContext) string {
	key := utils.GetValue(ctx, "GOOGLE_RECAPTCHA_KEY", "", true)
	secret := utils.GetValue(ctx, "GOOGLE_RECAPTCHA_SECRET", "", true)

	if key == "" || secret == "" {
		return ""
	}

	return key
}

// ExtractAttributes scans HTML and maps name="..." with its own data-validator if both exist
func ExtractAttributes(content string) (string, error) {
	// Match tags with both name and data-validator (input, textarea, select, etc.)
	tagRegex := regexp.MustCompile(`(?i)<(input|textarea|select)[^>]+>`)

	// Regex to extract attributes
	nameRegex := regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
	validatorRegex := regexp.MustCompile(`data-validator\s*=\s*"([^"]+)"`)

	result := make(map[string]string)

	tags := tagRegex.FindAllString(content, -1)
	for _, tag := range tags {
		nameMatch := nameRegex.FindStringSubmatch(tag)
		validatorMatch := validatorRegex.FindStringSubmatch(tag)

		if len(nameMatch) > 1 && len(validatorMatch) > 1 {
			name := nameMatch[1]
			validator := validatorMatch[1]
			result[name] = validator
		}
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
