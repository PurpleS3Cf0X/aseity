package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeanpaul/aseity/internal/config"
	"github.com/jeanpaul/aseity/internal/orchestrator"
	"github.com/jeanpaul/aseity/internal/tools"
)

// launchOrchestrator runs the orchestrator mode
func launchOrchestrator(cfg *config.Config, provName, modelName, query string, debug bool, maxRetries, maxSteps int, parallel bool, deepResearch bool) {
	if query == "" {
		fatal("orchestrator mode requires a query (provide as argument)")
	}

	// Create provider
	prov, err := makeProvider(cfg, provName, modelName)
	if err != nil {
		fatal("failed to create provider: %s", err)
	}

	// Create tool registry
	toolReg := tools.NewRegistry(nil, false) // No auto-approve, not allow-all

	// Configure orchestrator
	orchConfig := &orchestrator.Config{
		MaxRetries: maxRetries,
		MaxSteps:   maxSteps,
		StateDir:   filepath.Join(os.TempDir(), "aseity-orchestrator-state"),
	}

	if deepResearch {
		orchConfig.ForceIntent = &orchestrator.IntentOutput{
			IntentType:    "deep_research",
			RequiresTools: true,
			Complexity:    "complex",
			Reasoning:     "User explicitly requested Deep Research mode.",
			Entities:      []string{query},
		}
		fmt.Println("🚀 Deep Research Mode Enabled")
	}

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(prov, toolReg, orchConfig)

	// Enable parallel execution if requested
	orch.EnableParallel = parallel

	// Execute query
	ctx := context.Background()
	fmt.Printf("🤖 Orchestrator Mode\n")
	fmt.Printf("Provider: %s | Model: %s\n", provName, modelName)
	fmt.Printf("Max Retries: %d | Max Steps: %d | Parallel: %v\n\n", maxRetries, maxSteps, parallel)
	fmt.Printf("Query: %s\n\n", query)
	fmt.Println("Processing...")
	fmt.Println(strings.Repeat("─", 80))

	response, state, err := orch.ProcessQuery(ctx, query)
	if err != nil {
		fatal("orchestrator failed: %s", err)
	}

	// Display results
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println("✅ RESPONSE")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println(response)
	fmt.Println()

	// Debug mode: show detailed state
	if debug {
		fmt.Println(strings.Repeat("═", 80))
		fmt.Println("🔍 ORCHESTRATOR STATE (DEBUG)")
		fmt.Println(strings.Repeat("═", 80))

		fmt.Printf("\n📊 Metadata:\n")
		fmt.Printf("  Session ID: %s\n", state.SessionID)
		fmt.Printf("  Total Tokens: %d\n", state.TotalTokens)
		fmt.Printf("  Retry Count: %d\n", state.RetryCount)
		fmt.Printf("  Current Phase: %s\n", state.CurrentPhase)

		if state.Intent != nil {
			fmt.Printf("\n🎯 Intent:\n")
			fmt.Printf("  Type: %s\n", state.Intent.IntentType)
			fmt.Printf("  Complexity: %s\n", state.Intent.Complexity)
			fmt.Printf("  Requires Tools: %v\n", state.Intent.RequiresTools)
			fmt.Printf("  Reasoning: %s\n", state.Intent.Reasoning)
		}

		if state.Plan != nil {
			fmt.Printf("\n📋 Plan (%d steps):\n", len(state.Plan.Steps))
			for _, step := range state.Plan.Steps {
				fmt.Printf("  %d. %s\n", step.StepNumber, step.Action)
				fmt.Printf("     Reasoning: %s\n", step.Reasoning)
				if len(step.DependsOn) > 0 {
					fmt.Printf("     Depends on: %v\n", step.DependsOn)
				}
			}
			fmt.Printf("  Expected Outcome: %s\n", state.Plan.ExpectedOutcome)
		}

		if len(state.StepResults) > 0 {
			fmt.Printf("\n⚙️  Execution Results:\n")
			for _, result := range state.StepResults {
				status := "✅"
				if result.Status == "failure" {
					status = "❌"
				}
				fmt.Printf("  %s Step %d: %s (%dms)\n", status, result.StepNumber, result.Status, result.Duration)
				fmt.Printf("     Observation: %s\n", result.Observation)
				if result.Error != "" {
					fmt.Printf("     Error: %s\n", result.Error)
				}
			}
		}

		if state.Validation != nil {
			fmt.Printf("\n✓ Validation:\n")
			fmt.Printf("  Intent Fulfilled: %v\n", state.Validation.IntentFulfilled)
			fmt.Printf("  Confidence: %d%%\n", state.Validation.Confidence)
			fmt.Printf("  Recommendation: %s\n", state.Validation.Recommendation)
			if len(state.Validation.MissingInformation) > 0 {
				fmt.Printf("  Missing: %v\n", state.Validation.MissingInformation)
			}
		}

		if len(state.Warnings) > 0 {
			fmt.Printf("\n⚠️  Warnings:\n")
			for _, warning := range state.Warnings {
				fmt.Printf("  • %s\n", warning)
			}
		}

		if len(state.Errors) > 0 {
			fmt.Printf("\n❌ Errors:\n")
			for _, errMsg := range state.Errors {
				fmt.Printf("  • %s\n", errMsg)
			}
		}

		fmt.Println(strings.Repeat("═", 80))
	} else {
		// Non-debug: just show summary
		fmt.Printf("📊 Tokens: %d | Retries: %d", state.TotalTokens, state.RetryCount)
		if len(state.Warnings) > 0 {
			fmt.Printf(" | ⚠️  %d warnings", len(state.Warnings))
		}
		fmt.Println()
	}
}
