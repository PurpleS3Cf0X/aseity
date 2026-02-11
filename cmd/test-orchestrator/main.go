package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jeanpaul/aseity/internal/config"
	"github.com/jeanpaul/aseity/internal/model"
	"github.com/jeanpaul/aseity/internal/orchestrator"
	"github.com/jeanpaul/aseity/internal/provider/ollama"
	"github.com/jeanpaul/aseity/internal/tools"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Ollama provider with qwen2.5:32b
	ollamaURL := "http://localhost:11434"
	if p, ok := cfg.Providers["ollama"]; ok {
		ollamaURL = p.BaseURL
	}

	mgr := model.NewManager(ollamaURL, "")
	prov := ollama.New(ollamaURL, "qwen2.5:32b", mgr)

	// Initialize tools registry
	registry := tools.NewRegistry(nil, false)
	tools.RegisterDefaults(registry, nil, nil)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(prov, registry, &orchestrator.Config{
		MaxRetries: 3,
		MaxSteps:   10,
		StateDir:   "/tmp/aseity-states",
	})

	// Test queries
	queries := []string{
		"Find the latest threat intelligence news",
		"What is the weather like?",
		"Search for CVE-2024 vulnerabilities",
	}

	for i, query := range queries {
		fmt.Printf("\n=== Test %d: %s ===\n", i+1, query)

		response, state, err := orch.ProcessQuery(context.Background(), query)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		fmt.Printf("\n📊 STATE:\n")
		fmt.Printf("  Phase: %s\n", state.CurrentPhase)
		fmt.Printf("  Intent Type: %s\n", state.Intent.IntentType)
		fmt.Printf("  Complexity: %s\n", state.Intent.Complexity)
		fmt.Printf("  Steps: %d\n", len(state.Plan.Steps))
		fmt.Printf("  Retries: %d\n", state.RetryCount)
		fmt.Printf("  Tokens: %d\n", state.TotalTokens)

		fmt.Printf("\n📝 PLAN:\n")
		for _, step := range state.Plan.Steps {
			fmt.Printf("  %d. %s - %s\n", step.StepNumber, step.Action, step.Reasoning)
		}

		fmt.Printf("\n✅ RESULTS:\n")
		for _, result := range state.StepResults {
			status := "✓"
			if result.Status != "success" {
				status = "✗"
			}
			fmt.Printf("  %s Step %d: %s (%dms)\n", status, result.StepNumber, result.Observation, result.Duration)
		}

		fmt.Printf("\n🎯 VALIDATION:\n")
		fmt.Printf("  Fulfilled: %v\n", state.Validation.IntentFulfilled)
		fmt.Printf("  Confidence: %d%%\n", state.Validation.Confidence)
		fmt.Printf("  Recommendation: %s\n", state.Validation.Recommendation)

		fmt.Printf("\n💬 RESPONSE:\n%s\n", response)
		fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	}
}
