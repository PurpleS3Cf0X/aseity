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
- deep_research: User wants deep investigation (search + read + summarize) into a topic
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

	return fmt.Sprintf(`You are a Task Planner. Your goal is to create a valid JSON plan for the user's intent.

INSTRUCTIONS:
1. First, ANALYZE the request and available tools in a <reasoning> text block.
2. Then, GENERATE the JSON plan based on your reasoning.
3. The JSON must be valid and adhere to the schema.
4. "depends_on" field must use 1-BASED step numbers referring to PREVIOUS steps. Never use 0.
5. Do NOT include comments (like // or /* */) inside the JSON. Put all remarks in the <reasoning> block.

5. Do NOT include comments (like // or /* */) inside the JSON. Put all remarks in the <reasoning> block.

CRITICAL TOOL USAGE RULES:
- Do NOT use 'bash' to run 'open', 'xdg-open', 'start', 'curl', or 'wget'.
- ALWAYS use 'web_fetch' to read websites.
- ALWAYS use 'web_search' to find information.

Available Tools:
%s

Special Instructions for 'deep_research':
If the intent is 'deep_research', you MUST create a multi-step plan:
1. web_search: Search for the topic.
2. read_page: Read the content of the most relevant 2-3 search results.
3. (Implicit): The results will be synthesized in the final phase.

Required JSON Schema:
{
  "steps": [
    {
      "step_number": 1,
      "action": "tool_name",
      "parameters": {"param": "value"},
      "reasoning": "Why this step is needed",
      "depends_on": []
    }
  ],
  "expected_outcome": "Goal description"
}

Special Instructions for 'search' and 'deep_research':
If the intent is 'search' or 'deep_research', you MUST create a multi-step plan:
1. web_search: Search for the topic.
2. web_fetch / read_page: Read the content of the most relevant results. Do NOT stop at search results.
3. (Implicit): The results will be synthesized in the final phase.

Intent:
Type: %s
Goal: %s
Entities: %s
Complexity: %s

Output your reasoning followed by the JSON plan now:`, toolsSection, intent.IntentType, intent.Reasoning, entities, intent.Complexity)
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

		fmt.Printf("\n[DEBUG] Plan Creation Attempt %d Failed.\nOutput:\n%s\nError: %v\n", attempt, output, err)

		if attempt == maxRetries {
			fmt.Printf("\n[DEBUG] Plan Creation Failed. Last Output:\n%s\n[DEBUG] Error: %v\n", output, err)
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
