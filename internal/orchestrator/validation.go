package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidIntentTypes defines allowed intent types
var ValidIntentTypes = map[string]bool{
	"search":  true,
	"fetch":   true,
	"analyze": true,
	"execute": true,
	"create":  true,
	"modify":  true,
	"general": true,
}

// ValidComplexityLevels defines allowed complexity levels
var ValidComplexityLevels = map[string]bool{
	"simple":   true,
	"moderate": true,
	"complex":  true,
}

// ValidStatuses defines allowed step statuses
var ValidStatuses = map[string]bool{
	"success": true,
	"failure": true,
}

// ValidRecommendations defines allowed validation recommendations
var ValidRecommendations = map[string]bool{
	"proceed": true,
	"retry":   true,
	"replan":  true,
}

// ValidateIntentOutput validates the intent parser output
func ValidateIntentOutput(output string) (*IntentOutput, error) {
	// Clean the output (remove markdown code blocks if present)
	output = cleanJSONOutput(output)

	var intent IntentOutput
	if err := json.Unmarshal([]byte(output), &intent); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate intent_type
	if !ValidIntentTypes[intent.IntentType] {
		return nil, fmt.Errorf("invalid intent_type: %s (must be one of: search, fetch, analyze, execute, create, modify, general)", intent.IntentType)
	}

	// Validate complexity
	if !ValidComplexityLevels[intent.Complexity] {
		return nil, fmt.Errorf("invalid complexity: %s (must be one of: simple, moderate, complex)", intent.Complexity)
	}

	// Validate reasoning is not empty
	if strings.TrimSpace(intent.Reasoning) == "" {
		return nil, fmt.Errorf("reasoning cannot be empty")
	}

	return &intent, nil
}

// ValidatePlan validates the task planner output
func ValidatePlan(output string, maxSteps int) (*Plan, error) {
	// Clean the output
	output = cleanJSONOutput(output)

	var plan Plan
	if err := json.Unmarshal([]byte(output), &plan); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate plan has steps
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan must have at least one step")
	}

	// Validate plan length
	if len(plan.Steps) > maxSteps {
		return nil, fmt.Errorf("plan too long: %d steps (max %d)", len(plan.Steps), maxSteps)
	}

	// Validate expected outcome is not empty
	if strings.TrimSpace(plan.ExpectedOutcome) == "" {
		return nil, fmt.Errorf("expected_outcome cannot be empty")
	}

	// Validate each step
	for i, step := range plan.Steps {
		// Check step number is sequential
		if step.StepNumber != i+1 {
			return nil, fmt.Errorf("step numbers not sequential: expected %d, got %d", i+1, step.StepNumber)
		}

		// Check action is not empty
		if strings.TrimSpace(step.Action) == "" {
			return nil, fmt.Errorf("step %d: action cannot be empty", i+1)
		}

		// Check reasoning is not empty
		if strings.TrimSpace(step.Reasoning) == "" {
			return nil, fmt.Errorf("step %d: reasoning cannot be empty", i+1)
		}

		// Validate dependencies
		for _, dep := range step.DependsOn {
			if dep < 1 || dep >= step.StepNumber {
				return nil, fmt.Errorf("step %d: invalid dependency %d (must be between 1 and %d)", step.StepNumber, dep, step.StepNumber-1)
			}
		}
	}

	return &plan, nil
}

// ValidateStepResult validates a step execution result
func ValidateStepResult(output string) (*StepResult, error) {
	// Clean the output
	output = cleanJSONOutput(output)

	var result StepResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate status
	if !ValidStatuses[result.Status] {
		return nil, fmt.Errorf("invalid status: %s (must be 'success' or 'failure')", result.Status)
	}

	// Validate observation is not empty
	if strings.TrimSpace(result.Observation) == "" {
		return nil, fmt.Errorf("observation cannot be empty")
	}

	return &result, nil
}

// ValidateValidationResult validates the result validator output
func ValidateValidationResult(output string) (*ValidationResult, error) {
	// Clean the output
	output = cleanJSONOutput(output)

	var validation ValidationResult
	if err := json.Unmarshal([]byte(output), &validation); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate confidence is in range
	if validation.Confidence < 0 || validation.Confidence > 100 {
		return nil, fmt.Errorf("invalid confidence: %d (must be 0-100)", validation.Confidence)
	}

	// Validate recommendation
	if !ValidRecommendations[validation.Recommendation] {
		return nil, fmt.Errorf("invalid recommendation: %s (must be one of: proceed, retry, replan)", validation.Recommendation)
	}

	return &validation, nil
}

// cleanJSONOutput removes common artifacts from LLM JSON output
func cleanJSONOutput(output string) string {
	// Remove markdown code blocks
	output = strings.TrimPrefix(output, "```json")
	output = strings.TrimPrefix(output, "```")
	output = strings.TrimSuffix(output, "```")

	// Trim whitespace
	output = strings.TrimSpace(output)

	// Remove any leading/trailing text before/after JSON
	// Find first { and last }
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")

	if start != -1 && end != -1 && end > start {
		output = output[start : end+1]
	}

	return output
}

// CreateDefaultIntent creates a fail-safe default intent
func CreateDefaultIntent(query string) *IntentOutput {
	return &IntentOutput{
		Reasoning:     "Could not parse intent from model output",
		IntentType:    "general",
		RequiresTools: true,
		Entities:      []string{query},
		Complexity:    "simple",
	}
}

// CreateDefaultValidation creates a fail-safe default validation
func CreateDefaultValidation(allSuccess bool) *ValidationResult {
	confidence := 50
	recommendation := "proceed"

	if !allSuccess {
		confidence = 30
		recommendation = "retry"
	}

	return &ValidationResult{
		IntentFulfilled:    allSuccess,
		MissingInformation: []string{},
		Confidence:         confidence,
		Recommendation:     recommendation,
	}
}
