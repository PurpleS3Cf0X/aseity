package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// Orchestrator coordinates the multi-agent execution
type Orchestrator struct {
	provider       provider.Provider
	registry       *tools.Registry
	maxRetries     int
	maxSteps       int
	stateDir       string
	EnableParallel bool // Enable parallel execution of independent steps
}

// Config holds orchestrator configuration
type Config struct {
	MaxRetries int
	MaxSteps   int
	StateDir   string
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(prov provider.Provider, reg *tools.Registry, cfg *Config) *Orchestrator {
	if cfg == nil {
		cfg = &Config{
			MaxRetries: 3,
			MaxSteps:   10,
			StateDir:   "",
		}
	}

	return &Orchestrator{
		provider:   prov,
		registry:   reg,
		maxRetries: cfg.MaxRetries,
		maxSteps:   cfg.MaxSteps,
		stateDir:   cfg.StateDir,
	}
}

// ProcessQuery executes the full multi-agent pipeline
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (string, *AgentState, error) {
	state := NewAgentState(query)
	defer state.Save(o.stateDir) // Always save state for debugging

	// Iterative retry loop (max maxRetries + 1 attempts)
	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		state.RetryCount = attempt

		// Phase 1: Intent Parsing
		state.SetPhase("intent")
		intentOutput, err := o.extractIntent(ctx, query, state)
		if err != nil {
			state.AddError(err)
			state.AddWarning(fmt.Sprintf("Intent parsing failed, using default: %s", err.Error()))
		}
		state.Intent = intentOutput

		// Phase 2: Task Planning
		state.SetPhase("planning")
		plan, err := o.createPlan(ctx, intentOutput, state)
		if err != nil {
			state.AddError(err)
			return "", state, fmt.Errorf("planning failed: %w", err)
		}
		state.Plan = plan

		// Phase 3: Execution
		state.SetPhase("execution")
		var results []StepResult

		if o.EnableParallel {
			results, err = o.executePlanParallel(ctx, plan)
		} else {
			results, err = o.executePlan(ctx, plan)
		}

		if err != nil {
			state.AddError(err)
			// Check if context was cancelled
			if ctx.Err() != nil {
				return "", state, fmt.Errorf("execution cancelled: %w", ctx.Err())
			}
			return "", state, fmt.Errorf("execution failed: %w", err)
		}
		state.StepResults = results

		// Phase 4: Validation
		state.SetPhase("validation")
		validation, err := o.validateResults(ctx, intentOutput, plan, results, state)
		if err != nil {
			state.AddError(err)
			state.AddWarning(fmt.Sprintf("Validation failed, using default: %s", err.Error()))
		}
		state.Validation = validation

		// Check if we should proceed, retry, or replan
		if validation.Recommendation == "proceed" {
			break
		}

		if validation.Recommendation == "retry" && attempt < o.maxRetries {
			state.AddWarning(fmt.Sprintf("Retrying execution (attempt %d/%d)", attempt+1, o.maxRetries))
			continue
		}

		if validation.Recommendation == "replan" && attempt < o.maxRetries {
			state.AddWarning(fmt.Sprintf("Replanning (attempt %d/%d)", attempt+1, o.maxRetries))
			continue
		}

		// If we've exhausted retries, proceed anyway
		if attempt == o.maxRetries {
			state.AddWarning(fmt.Sprintf("Max retries reached (%d), proceeding with current results", o.maxRetries))
			break
		}
	}

	// Phase 5: Synthesis
	state.SetPhase("synthesis")
	response := o.synthesizeResponse(ctx, query, state.StepResults, state)
	state.FinalResponse = response

	return response, state, nil
}

func (o *Orchestrator) callModel(ctx context.Context, prompt string) (string, int) {
	// Use Chat which returns a channel
	ch, err := o.provider.Chat(ctx, []provider.Message{
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil {
		return "", 0
	}

	// Collect all chunks efficiently
	var builder strings.Builder
	var tokens int
	for chunk := range ch {
		if chunk.Error != nil {
			return "", 0
		}
		builder.WriteString(chunk.Delta)
		if chunk.Done && chunk.Usage != nil {
			tokens = chunk.Usage.TotalTokens
		}
	}
	return builder.String(), tokens
}

func (o *Orchestrator) extractIntent(ctx context.Context, query string, state *AgentState) (*IntentOutput, error) {
	callModel := func(prompt string) string {
		result, tokens := o.callModel(ctx, prompt)
		state.AddTokens(tokens)
		return result
	}
	return ParseIntent(query, callModel, o.maxRetries)
}

func (o *Orchestrator) createPlan(ctx context.Context, intentOutput *IntentOutput, state *AgentState) (*Plan, error) {
	// Build tools section
	toolsSection := ""
	for _, toolDef := range o.registry.ToolDefs() {
		toolsSection += fmt.Sprintf("- %s: %s\n", toolDef.Name, toolDef.Description)
	}

	callModel := func(prompt string) string {
		result, tokens := o.callModel(ctx, prompt)
		state.AddTokens(tokens)
		return result
	}

	return CreatePlan(intentOutput, toolsSection, callModel, o.maxRetries, o.maxSteps)
}

func (o *Orchestrator) executePlan(ctx context.Context, plan *Plan) ([]StepResult, error) {
	results := make([]StepResult, len(plan.Steps))

	for i, step := range plan.Steps {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		start := time.Now()

		// Execute the tool with previous results for parameter extraction
		result, err := o.executeStep(ctx, step, results[:i])

		duration := time.Since(start).Milliseconds()

		if err != nil {
			results[i] = StepResult{
				StepNumber:  step.StepNumber,
				Status:      "failure",
				Result:      "",
				Observation: fmt.Sprintf("Step failed: %s", err.Error()),
				Error:       err.Error(),
				Duration:    duration,
			}
		} else {
			results[i] = StepResult{
				StepNumber:  step.StepNumber,
				Status:      "success",
				Result:      result,
				Observation: fmt.Sprintf("Executed %s successfully", step.Action),
				Duration:    duration,
			}
		}
	}

	return results, nil
}

func (o *Orchestrator) executeStep(ctx context.Context, step PlanStep, previousResults []StepResult) (string, error) {
	// Extract dynamic parameters from previous results
	params := extractDynamicParams(step.Parameters, previousResults)

	// Convert parameters to JSON string
	paramsJSON, err := marshalParams(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Execute the tool
	result, err := o.registry.Execute(ctx, step.Action, paramsJSON, nil)
	if err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("%s", result.Error)
	}

	return result.Output, nil
}

func marshalParams(params map[string]interface{}) (string, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (o *Orchestrator) validateResults(ctx context.Context, intentOutput *IntentOutput, plan *Plan, results []StepResult, state *AgentState) (*ValidationResult, error) {
	callModel := func(prompt string) string {
		result, tokens := o.callModel(ctx, prompt)
		state.AddTokens(tokens)
		return result
	}
	validation, err := ValidateResults(intentOutput, plan, results, callModel)
	return validation, err
}

func (o *Orchestrator) synthesizeResponse(ctx context.Context, query string, results []StepResult, state *AgentState) string {
	callModel := func(prompt string) string {
		result, tokens := o.callModel(ctx, prompt)
		state.AddTokens(tokens)
		return result
	}
	return SynthesizeResponse(query, results, callModel)
}
