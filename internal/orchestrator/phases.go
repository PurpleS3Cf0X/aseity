package orchestrator

import (
	"fmt"
	"strings"
)

// Intent Parser System Prompt
const IntentParserPrompt = `You are an Intent Parser. Your ONLY job is to analyze the user request and output valid JSON.

CRITICAL RULES:
1. Output ONLY valid JSON - no explanations, no markdown, no extra text
2. Use ONLY the exact field names specified
3. Do NOT add any fields not in the schema

Required JSON format:
{
  "reasoning": "Your understanding of what the user wants",
  "intent_type": "search|fetch|analyze|execute|create|modify|general",
  "requires_tools": true|false,
  "entities": ["list", "of", "key", "entities"],
  "complexity": "simple|moderate|complex"
}

Intent Types:
- search: User wants to find information (web search, documentation lookup)
- fetch: User wants to retrieve specific content (webpage, file, API data)
- analyze: User wants analysis of existing data
- execute: User wants to run commands or scripts
- create: User wants to create new files/resources
- modify: User wants to edit existing files/resources
- general: Conversational or unclear intent

Complexity Levels:
- simple: Single action, clear goal
- moderate: 2-3 steps, some dependencies
- complex: Multiple steps, complex dependencies

Now analyze this user request and output ONLY the JSON:`

// ParseIntent extracts intent from user query with retry logic
func ParseIntent(query string, callModel func(string) string, maxRetries int) (*IntentOutput, error) {
	prompt := IntentParserPrompt + "\n\nUser: " + query

	for attempt := 1; attempt <= maxRetries; attempt++ {
		output := callModel(prompt)
		intent, err := ValidateIntentOutput(output)
		if err == nil {
			return intent, nil
		}

		if attempt == maxRetries {
			return CreateDefaultIntent(query), fmt.Errorf("intent parsing failed after %d attempts, using default: %w", maxRetries, err)
		}

		prompt = fmt.Sprintf(`%s

Previous attempt produced invalid output: %s
Error: %s

Please output ONLY valid JSON matching the schema exactly.

User: %s`, IntentParserPrompt, output, err.Error(), query)
	}

	return CreateDefaultIntent(query), fmt.Errorf("unexpected error in intent parsing")
}

// Task Planner System Prompt Template
func BuildPlannerPrompt(intent *IntentOutput, toolsSection string, maxSteps int) string {
	entities := strings.Join(intent.Entities, ", ")

	return fmt.Sprintf(`You are a Task Planner. Based on the intent, create a step-by-step plan using ONLY available tools.

CRITICAL RULES:
1. Output ONLY valid JSON - no explanations, no markdown, no extra text
2. Use ONLY tools from the list below
3. Each step must have valid parameters for that tool
4. Steps must be in logical order
5. Maximum %d steps

Available Tools:
%s

Required JSON format:
{
  "steps": [
    {
      "step_number": 1,
      "action": "tool_name",
      "parameters": {"param": "value"},
      "reasoning": "Why this step is needed",
      "depends_on": [optional array of step numbers this depends on]
    }
  ],
  "expected_outcome": "What the plan should achieve"
}

Now create a plan for this intent:

Intent Type: %s
Reasoning: %s
Entities: %s
Complexity: %s

Output ONLY the JSON plan:`, maxSteps, toolsSection, intent.IntentType, intent.Reasoning, entities, intent.Complexity)
}

// CreatePlan generates a plan from the intent
func CreatePlan(intent *IntentOutput, toolsSection string, callModel func(string) string, maxRetries int, maxSteps int) (*Plan, error) {
	prompt := BuildPlannerPrompt(intent, toolsSection, maxSteps)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		output := callModel(prompt)
		plan, err := ValidatePlan(output, maxSteps)
		if err == nil {
			return plan, nil
		}

		if attempt == maxRetries {
			return nil, fmt.Errorf("plan creation failed after %d attempts: %w", maxRetries, err)
		}

		prompt = fmt.Sprintf(`%s

Previous attempt produced invalid output: %s
Error: %s

Please output ONLY valid JSON matching the schema exactly.`, BuildPlannerPrompt(intent, toolsSection, maxSteps), output, err.Error())
	}

	return nil, fmt.Errorf("unexpected error in plan creation")
}

// Validation Prompt Template
func BuildValidationPrompt(intent *IntentOutput, plan *Plan, results []StepResult) string {
	resultsText := ""
	for _, r := range results {
		resultsText += fmt.Sprintf("Step %d: %s - %s\n", r.StepNumber, r.Status, r.Observation)
	}

	return fmt.Sprintf(`You are a Result Validator.

Original Intent: %s
Expected Outcome: %s

Results from execution:
%s

Output ONLY valid JSON:
{
  "intent_fulfilled": true|false,
  "missing_information": ["list", "of", "gaps"],
  "confidence": 0-100,
  "recommendation": "proceed|retry|replan"
}`, intent.Reasoning, plan.ExpectedOutcome, resultsText)
}

// ValidateResults checks if the plan achieved the intent
func ValidateResults(intent *IntentOutput, plan *Plan, results []StepResult, callModel func(string) string) (*ValidationResult, error) {
	prompt := BuildValidationPrompt(intent, plan, results)
	output := callModel(prompt)

	validation, err := ValidateValidationResult(output)
	if err != nil {
		// Fail-safe: do basic validation ourselves
		allSuccess := true
		for _, r := range results {
			if r.Status != "success" {
				allSuccess = false
				break
			}
		}
		return CreateDefaultValidation(allSuccess), nil
	}

	return validation, nil
}

// Synthesis Prompt Template
func BuildSynthesisPrompt(query string, results []StepResult) string {
	resultsText := ""
	for _, r := range results {
		if r.Status == "success" {
			resultsText += fmt.Sprintf("- %s\n", r.Observation)
		}
	}

	return fmt.Sprintf(`You are a Response Synthesizer.

Original Query: %s

Results Summary:
%s

Create a clear, concise response to the user. Include:
1. Direct answer to their question
2. Key findings from the results
3. Any relevant context

Keep it under 200 words. Be specific and factual.`, query, resultsText)
}

// SynthesizeResponse creates a natural language response
func SynthesizeResponse(query string, results []StepResult, callModel func(string) string) string {
	prompt := BuildSynthesisPrompt(query, results)
	response := callModel(prompt)

	// Sanitize output
	response = strings.TrimSpace(response)
	response = cleanJSONOutput(response)

	// If model failed to synthesize, create basic response
	if len(response) < 20 {
		return CreateBasicResponse(results)
	}

	return response
}

// CreateBasicResponse creates a fallback response
func CreateBasicResponse(results []StepResult) string {
	var parts []string
	for _, r := range results {
		if r.Status == "success" {
			parts = append(parts, r.Observation)
		}
	}

	if len(parts) == 0 {
		return "No results available."
	}

	response := "Here's what I found:\n\n"
	for i, part := range parts {
		response += fmt.Sprintf("%d. %s\n", i+1, part)
	}

	return response
}
