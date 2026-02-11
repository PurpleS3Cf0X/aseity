package orchestrator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// extractDynamicParams processes parameters and replaces placeholders with values from previous steps
func extractDynamicParams(params map[string]interface{}, previousResults []StepResult) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range params {
		if str, ok := value.(string); ok {
			// Check for extraction patterns
			extracted := extractPattern(str, previousResults)
			result[key] = extracted
		} else {
			result[key] = value
		}
	}

	return result
}

// extractPattern handles various extraction patterns
func extractPattern(pattern string, previousResults []StepResult) string {
	// Pattern: $EXTRACT_URL_FROM_STEP_N
	if strings.HasPrefix(pattern, "$EXTRACT_URL_FROM_STEP_") {
		stepNum := extractStepNumber(pattern)
		if stepNum > 0 && stepNum <= len(previousResults) {
			return extractFirstURL(previousResults[stepNum-1].Result)
		}
		return pattern
	}

	// Pattern: $EXTRACT_FIRST_URL
	if pattern == "$EXTRACT_FIRST_URL" && len(previousResults) > 0 {
		// Get URL from the most recent step
		return extractFirstURL(previousResults[len(previousResults)-1].Result)
	}

	// Pattern: $EXTRACT_JSON_FIELD:step_N:field_name
	if strings.HasPrefix(pattern, "$EXTRACT_JSON_FIELD:") {
		parts := strings.Split(pattern, ":")
		if len(parts) >= 3 {
			stepNum := extractStepNumberFromString(parts[1])
			fieldName := parts[2]
			if stepNum > 0 && stepNum <= len(previousResults) {
				return extractJSONField(previousResults[stepNum-1].Result, fieldName)
			}
		}
		return pattern
	}

	// Pattern: $REGEX:step_N:pattern
	if strings.HasPrefix(pattern, "$REGEX:") {
		parts := strings.SplitN(pattern, ":", 3)
		if len(parts) >= 3 {
			stepNum := extractStepNumberFromString(parts[1])
			regexPattern := parts[2]
			if stepNum > 0 && stepNum <= len(previousResults) {
				return extractWithRegex(previousResults[stepNum-1].Result, regexPattern)
			}
		}
		return pattern
	}

	// No pattern matched, return as-is
	return pattern
}

// extractStepNumber extracts the step number from patterns like "$EXTRACT_URL_FROM_STEP_1"
func extractStepNumber(pattern string) int {
	re := regexp.MustCompile(`STEP_(\d+)`)
	matches := re.FindStringSubmatch(pattern)
	if len(matches) >= 2 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}

// extractStepNumberFromString extracts step number from strings like "step_1" or "1"
func extractStepNumberFromString(s string) int {
	s = strings.TrimPrefix(s, "step_")
	num, _ := strconv.Atoi(s)
	return num
}

// extractFirstURL finds the first URL in the text
func extractFirstURL(text string) string {
	// Match http:// or https:// URLs
	urlRe := regexp.MustCompile(`https?://[^\s\)]+`)
	matches := urlRe.FindStringSubmatch(text)
	if len(matches) > 0 {
		// Clean up common trailing characters
		url := matches[0]
		url = strings.TrimRight(url, ".,;:!?")
		return url
	}
	return ""
}

// extractJSONField extracts a specific field from JSON in the result
func extractJSONField(text string, fieldName string) string {
	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		// If not valid JSON, try to find JSON block in text
		jsonRe := regexp.MustCompile(`\{[^{}]*\}`)
		jsonMatch := jsonRe.FindString(text)
		if jsonMatch != "" {
			if err := json.Unmarshal([]byte(jsonMatch), &data); err != nil {
				return ""
			}
		} else {
			return ""
		}
	}

	// Extract field
	if value, ok := data[fieldName]; ok {
		return fmt.Sprintf("%v", value)
	}

	return ""
}

// extractWithRegex extracts text using a regex pattern
func extractWithRegex(text string, pattern string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}

	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		// Return first capture group
		return matches[1]
	} else if len(matches) > 0 {
		// Return full match
		return matches[0]
	}

	return ""
}
