package orchestrator

import (
	"testing"
)

func TestExtractFirstURL(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "simple URL",
			text:     "Check out https://example.com for more info",
			expected: "https://example.com",
		},
		{
			name:     "URL with path",
			text:     "Visit https://weather.com/weather/today/l/Bengaluru for weather",
			expected: "https://weather.com/weather/today/l/Bengaluru",
		},
		{
			name:     "multiple URLs - returns first",
			text:     "Links: https://first.com and https://second.com",
			expected: "https://first.com",
		},
		{
			name:     "URL with trailing punctuation",
			text:     "See https://example.com.",
			expected: "https://example.com",
		},
		{
			name:     "no URL",
			text:     "No links here",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstURL(tt.text)
			if result != tt.expected {
				t.Errorf("extractFirstURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractStepNumber(t *testing.T) {
	tests := []struct {
		pattern  string
		expected int
	}{
		{"$EXTRACT_URL_FROM_STEP_1", 1},
		{"$EXTRACT_URL_FROM_STEP_2", 2},
		{"$EXTRACT_URL_FROM_STEP_10", 10},
		{"$EXTRACT_FIRST_URL", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := extractStepNumber(tt.pattern)
			if result != tt.expected {
				t.Errorf("extractStepNumber(%s) = %d, want %d", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestExtractJSONField(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		fieldName string
		expected  string
	}{
		{
			name:      "simple JSON",
			text:      `{"temperature": "25°C", "city": "Bengaluru"}`,
			fieldName: "temperature",
			expected:  "25°C",
		},
		{
			name:      "JSON in text",
			text:      `The data is: {"status": "success", "value": 42}`,
			fieldName: "value",
			expected:  "42",
		},
		{
			name:      "field not found",
			text:      `{"foo": "bar"}`,
			fieldName: "missing",
			expected:  "",
		},
		{
			name:      "invalid JSON",
			text:      `not json`,
			fieldName: "any",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONField(tt.text, tt.fieldName)
			if result != tt.expected {
				t.Errorf("extractJSONField() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractWithRegex(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pattern  string
		expected string
	}{
		{
			name:     "temperature extraction",
			text:     "Current temperature: 25°C",
			pattern:  `(\d+)°C`,
			expected: "25",
		},
		{
			name:     "email extraction",
			text:     "Contact: user@example.com",
			pattern:  `([a-z]+@[a-z]+\.[a-z]+)`,
			expected: "user@example.com",
		},
		{
			name:     "no match",
			text:     "No numbers here",
			pattern:  `\d+`,
			expected: "",
		},
		{
			name:     "invalid regex",
			text:     "text",
			pattern:  `[invalid`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWithRegex(tt.text, tt.pattern)
			if result != tt.expected {
				t.Errorf("extractWithRegex() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractDynamicParams(t *testing.T) {
	// Setup previous results
	previousResults := []StepResult{
		{
			StepNumber: 1,
			Status:     "success",
			Result:     "1. Weather in Bengaluru\n   https://weather.com/weather/today/l/Bengaluru\n   Current conditions",
		},
		{
			StepNumber: 2,
			Status:     "success",
			Result:     `{"temperature": "25°C", "humidity": "65%"}`,
		},
	}

	tests := []struct {
		name     string
		params   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "extract URL from step 1",
			params: map[string]interface{}{
				"url": "$EXTRACT_URL_FROM_STEP_1",
			},
			expected: map[string]interface{}{
				"url": "https://weather.com/weather/today/l/Bengaluru",
			},
		},
		{
			name: "extract first URL",
			params: map[string]interface{}{
				"url": "$EXTRACT_FIRST_URL",
			},
			expected: map[string]interface{}{
				"url": "", // No URL in step 2 (most recent)
			},
		},
		{
			name: "extract JSON field",
			params: map[string]interface{}{
				"temp": "$EXTRACT_JSON_FIELD:step_2:temperature",
			},
			expected: map[string]interface{}{
				"temp": "25°C",
			},
		},
		{
			name: "no extraction pattern",
			params: map[string]interface{}{
				"query": "weather Bengaluru",
			},
			expected: map[string]interface{}{
				"query": "weather Bengaluru",
			},
		},
		{
			name: "mixed parameters",
			params: map[string]interface{}{
				"url":   "$EXTRACT_URL_FROM_STEP_1",
				"query": "static value",
				"count": 5,
			},
			expected: map[string]interface{}{
				"url":   "https://weather.com/weather/today/l/Bengaluru",
				"query": "static value",
				"count": 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDynamicParams(tt.params, previousResults)

			// Compare each field
			for key, expectedVal := range tt.expected {
				if result[key] != expectedVal {
					t.Errorf("extractDynamicParams()[%s] = %v, want %v", key, result[key], expectedVal)
				}
			}
		})
	}
}

func TestExtractPattern(t *testing.T) {
	previousResults := []StepResult{
		{
			StepNumber: 1,
			Result:     "Visit https://example.com for details",
		},
		{
			StepNumber: 2,
			Result:     `{"price": "$99.99", "stock": "in stock"}`,
		},
	}

	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "extract URL from step 1",
			pattern:  "$EXTRACT_URL_FROM_STEP_1",
			expected: "https://example.com",
		},
		{
			name:     "extract JSON field",
			pattern:  "$EXTRACT_JSON_FIELD:step_2:price",
			expected: "$99.99",
		},
		{
			name:     "regex extraction",
			pattern:  "$REGEX:step_2:\"price\": \"([^\"]+)\"",
			expected: "$99.99",
		},
		{
			name:     "no pattern",
			pattern:  "static value",
			expected: "static value",
		},
		{
			name:     "invalid step number",
			pattern:  "$EXTRACT_URL_FROM_STEP_99",
			expected: "$EXTRACT_URL_FROM_STEP_99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPattern(tt.pattern, previousResults)
			if result != tt.expected {
				t.Errorf("extractPattern(%s) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}
