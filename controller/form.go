package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"zin-engine/model"
	"zin-engine/utils"
)

// Define a reasonable maximum upload size (e.g., 1 MB)
const MAX_UPLOAD_SIZE = 1024 * 1024

var defaultValidators = map[string]string{
	"required": `.+`, // must not be empty
	"email":    `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
	"mobile":   `^\+?[1-9]\d{9,14}$`,
}

type formData map[string]any

func HandleFormSubmission(cReq *http.Request, ctx *model.RequestContext) (int, string) {
	// Read post body
	body, err := io.ReadAll(cReq.Body)
	if err != nil {
		return 400, `{"error":"Failed to read form content."}`

	}
	defer cReq.Body.Close()

	if len(body) == 0 {
		return 400, `{"error":"Form content is empty."}`
	}

	var formData formData
	if err := json.Unmarshal(body, &formData); err != nil {
		return 400, `{"error":"Unable to parse form data, invalid json format"}`
	}

	// Get all required fields first
	zinFormId, ok1 := formData["zinFormId"].(string)
	zinFormSession, ok2 := formData["zinFormSession"].(string)
	zinFormSource, ok3 := formData["zinFormSource"].(string)
	zinFormCaptcha, ok4 := formData["zinFormCaptcha"].(string)

	if !ok1 || !ok2 || !ok3 || zinFormId == "" || zinFormSession == "" || zinFormSource == "" {
		return 404, `{"error":"Form submission invalid: data was tampered with or not from a valid ZinForm."}`
	}

	// Validate Session
	sessionData, err := utils.Decrypt(zinFormSession, zinFormId)
	if err != nil {
		return 401, fmt.Sprintf(`{"error":"%v"}`, err)
	}

	// Extract form submission link form session data
	zinFormSessionData := strings.Split(sessionData, "::")
	if len(zinFormSessionData) != 5 || zinFormSessionData[2] != ctx.ClientIp {
		return 401, `{"error":"Form session token is tempered or expired, try again"}`
	}

	// Validate inputs submitted by client
	validInputs := validateInputs(zinFormSessionData[4], formData)
	if validInputs != nil {
		return 401, fmt.Sprintf(`{"error":"Validation Error: %v"}`, validInputs)
	}

	// Check if captcha-verification is applicable
	zinFormURL := zinFormSessionData[0]
	zinFormValidatorService := zinFormSessionData[3]
	if zinFormValidatorService == "GOOGLE" {
		recaptchaSecret := utils.GetValue(ctx, "GOOGLE_RECAPTCHA_SECRET", "", true)
		if recaptchaSecret != "" {
			if !ok4 || zinFormCaptcha == "" {
				return 401, `{"error":"ReCAPTCHA token missing or invalid, try again"}`
			}

			err := verifyCaptchaTokne(recaptchaSecret, zinFormCaptcha, ctx.ClientIp)
			if err != nil {
				return 401, fmt.Sprintf(`{"error":"%v"}`, err)
			}

			zinFormValidatorService = "Google reCAPTCHA v3"
		}
	}

	// Remove form-defaults from form payload
	delete(formData, "zinFormId")
	delete(formData, "zinFormSession")
	delete(formData, "zinFormSource")
	delete(formData, "zinFormCaptcha")

	// Forward form data to configured endpoint
	formPayload, _ := json.Marshal(formData)
	req, err := http.NewRequest("POST", zinFormURL, bytes.NewBuffer(formPayload))
	if err != nil {
		return 500, fmt.Sprintf(`{"error":"%v"}`, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", strings.ReplaceAll(ctx.ServerVersion, "zin", "zin-http-client"))
	req.Header.Set("X-ZIN-Form", zinFormSource)
	req.Header.Set("X-ZIN-Ref", zinFormId)
	req.Header.Set("X-ZIN-Validator", zinFormValidatorService)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 502, fmt.Sprintf(`{"error":"%v"}`, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 200, `{"message":"Form submitted successfully"}`
	}

	if resp.Header.Get("Content-Type") == "application/json" {
		return resp.StatusCode, string(respBody)
	}

	// External API returned error
	return resp.StatusCode, `{"error":"` + string(respBody) + `"}`

}

func verifyCaptchaTokne(secret string, token string, clientIp string) error {

	recaptchaResp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify",
		map[string][]string{
			"secret":   {secret},
			"response": {token},
			"remoteip": {clientIp},
		})

	if err != nil {
		return err
	}
	defer recaptchaResp.Body.Close()

	var recaptchaResult map[string]any
	json.NewDecoder(recaptchaResp.Body).Decode(&recaptchaResult)

	if success, ok := recaptchaResult["success"].(bool); !ok || !success {
		return fmt.Errorf("failed ReCAPTCHA verification")
	}

	return nil
}

// Main validation function
func validateInputs(validator string, data formData) error {
	if strings.TrimSpace(validator) == "" {
		return nil // No validators, nothing to do
	}

	var rules map[string]string
	if err := json.Unmarshal([]byte(validator), &rules); err != nil {
		return fmt.Errorf("invalid validator JSON: %v", err)
	}

	for field, ruleKey := range rules {
		// Get value from formData
		rawVal, ok := data[field]
		if !ok {
			rawVal = "" // Treat missing as blank
		}

		// Convert value to string
		valStr := fmt.Sprintf("%v", rawVal)
		valStr = strings.TrimSpace(valStr)

		// Multiple rules (e.g., "required|email")
		ruleParts := strings.Split(ruleKey, "|")
		for _, rule := range ruleParts {
			rule = strings.TrimSpace(rule)

			regexStr, err := getValidatorRegex(rule)
			if err != nil {
				return err
			}

			matched, _ := regexp.MatchString(regexStr, valStr)
			if !matched {
				if rule == "required" && valStr == "" {
					return fmt.Errorf("input field '%s' is required and cannot be blank", field)
				}
				return fmt.Errorf("input field '%s' is invalid for rule '%s'", field, rule)
			}
		}
	}

	return nil
}

// Function to get regex from validator key, supporting zin.somekey
func getValidatorRegex(key string) (string, error) {

	if val, ok := defaultValidators[key]; ok {
		return val, nil
	}

	return "", fmt.Errorf("validator not found: %s", key)
}
