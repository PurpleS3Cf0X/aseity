package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIntentOutput_Valid(t *testing.T) {
	output := `{
		"reasoning": "User wants to search for information",
		"intent_type": "search",
		"requires_tools": true,
		"entities": ["threat", "intel"],
		"complexity": "simple"
	}`

	intent, err := ValidateIntentOutput(output)
	assert.NoError(t, err)
	assert.Equal(t, "search", intent.IntentType)
	assert.Equal(t, "simple", intent.Complexity)
	assert.True(t, intent.RequiresTools)
	assert.Len(t, intent.Entities, 2)
}

func TestValidateIntentOutput_WithMarkdown(t *testing.T) {
	output := "```json\n{\"reasoning\":\"test\",\"intent_type\":\"general\",\"requires_tools\":false,\"entities\":[],\"complexity\":\"simple\"}\n```"

	intent, err := ValidateIntentOutput(output)
	assert.NoError(t, err)
	assert.Equal(t, "general", intent.IntentType)
}

func TestValidateIntentOutput_InvalidJSON(t *testing.T) {
	output := "This is not JSON"

	_, err := ValidateIntentOutput(output)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestValidateIntentOutput_InvalidIntentType(t *testing.T) {
	output := `{"reasoning":"test","intent_type":"invalid","requires_tools":true,"entities":[],"complexity":"simple"}`

	_, err := ValidateIntentOutput(output)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid intent_type")
}

func TestValidatePlan_Valid(t *testing.T) {
	output := `{
		"steps": [
			{
				"step_number": 1,
				"action": "web_search",
				"parameters": {"query": "test"},
				"reasoning": "Need to search"
			},
			{
				"step_number": 2,
				"action": "web_fetch",
				"parameters": {"url": "http://example.com"},
				"reasoning": "Need to fetch",
				"depends_on": [1]
			}
		],
		"expected_outcome": "Get search results and fetch page"
	}`

	plan, err := ValidatePlan(output, 10)
	assert.NoError(t, err)
	assert.Len(t, plan.Steps, 2)
	assert.Equal(t, "web_search", plan.Steps[0].Action)
	assert.Equal(t, []int{1}, plan.Steps[1].DependsOn)
}

func TestValidatePlan_TooManySteps(t *testing.T) {
	output := `{
		"steps": [
			{"step_number": 1, "action": "test", "parameters": {}, "reasoning": "test"},
			{"step_number": 2, "action": "test", "parameters": {}, "reasoning": "test"},
			{"step_number": 3, "action": "test", "parameters": {}, "reasoning": "test"}
		],
		"expected_outcome": "test"
	}`

	_, err := ValidatePlan(output, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan too long")
}

func TestValidatePlan_NonSequentialSteps(t *testing.T) {
	output := `{
		"steps": [
			{"step_number": 1, "action": "test", "parameters": {}, "reasoning": "test"},
			{"step_number": 3, "action": "test", "parameters": {}, "reasoning": "test"}
		],
		"expected_outcome": "test"
	}`

	_, err := ValidatePlan(output, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not sequential")
}

func TestValidateStepResult_Valid(t *testing.T) {
	output := `{
		"step_number": 1,
		"status": "success",
		"result": "data here",
		"observation": "Found the data"
	}`

	result, err := ValidateStepResult(output)
	assert.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "Found the data", result.Observation)
}

func TestValidateStepResult_InvalidStatus(t *testing.T) {
	output := `{
		"step_number": 1,
		"status": "pending",
		"result": "data",
		"observation": "test"
	}`

	_, err := ValidateStepResult(output)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestValidateValidationResult_Valid(t *testing.T) {
	output := `{
		"intent_fulfilled": true,
		"missing_information": [],
		"confidence": 85,
		"recommendation": "proceed"
	}`

	validation, err := ValidateValidationResult(output)
	assert.NoError(t, err)
	assert.True(t, validation.IntentFulfilled)
	assert.Equal(t, 85, validation.Confidence)
	assert.Equal(t, "proceed", validation.Recommendation)
}

func TestValidateValidationResult_InvalidConfidence(t *testing.T) {
	output := `{
		"intent_fulfilled": true,
		"missing_information": [],
		"confidence": 150,
		"recommendation": "proceed"
	}`

	_, err := ValidateValidationResult(output)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid confidence")
}

func TestCreateDefaultIntent(t *testing.T) {
	intent := CreateDefaultIntent("test query")
	assert.Equal(t, "general", intent.IntentType)
	assert.True(t, intent.RequiresTools)
	assert.Contains(t, intent.Entities, "test query")
}

func TestCreateDefaultValidation(t *testing.T) {
	validation := CreateDefaultValidation(true)
	assert.True(t, validation.IntentFulfilled)
	assert.Equal(t, "proceed", validation.Recommendation)

	validation = CreateDefaultValidation(false)
	assert.False(t, validation.IntentFulfilled)
	assert.Equal(t, "retry", validation.Recommendation)
}
