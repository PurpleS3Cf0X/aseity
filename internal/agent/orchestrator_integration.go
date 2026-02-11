package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeanpaul/aseity/internal/orchestrator"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// SetOrchestrator attaches an orchestrator instance to the agent
func (a *Agent) SetOrchestrator(orch *orchestrator.Orchestrator, config OrchestratorConfig) {
	a.orchestrator = orch
	a.orchestratorConfig = config
	if config.ShowProgress {
		a.ProgressCh = make(chan OrchestratorProgress, 10)
	}
}

// ShouldUseOrchestrator determines if a query should use the orchestrator
func (a *Agent) ShouldUseOrchestrator(msg string) bool {
	if !a.orchestratorConfig.Enabled || !a.orchestratorConfig.AutoDetect {
		return false
	}

	msgLower := strings.ToLower(msg)

	// Keywords that indicate complex queries
	keywords := []string{
		"research", "compare", "analyze", "weather",
		"find and", "search for", "look up", "what's the",
		"tell me about", "information about", "details on",
		"summarize", "explain", "investigate",
	}

	for _, kw := range keywords {
		if strings.Contains(msgLower, kw) {
			return true
		}
	}

	// Multi-step indicators
	if strings.Contains(msgLower, " and ") || strings.Contains(msgLower, " then ") {
		return true
	}

	// URL or web-related queries
	if strings.Contains(msgLower, "http") || strings.Contains(msgLower, "www.") {
		return true
	}

	return false
}

// ProcessWithOrchestrator handles query using the orchestrator
func (a *Agent) ProcessWithOrchestrator(ctx context.Context, msg string) (string, error) {
	if a.orchestrator == nil {
		return "", fmt.Errorf("orchestrator not initialized")
	}

	orch, ok := a.orchestrator.(*orchestrator.Orchestrator)
	if !ok {
		return "", fmt.Errorf("invalid orchestrator type")
	}

	// Send progress update
	if a.ProgressCh != nil {
		a.ProgressCh <- OrchestratorProgress{
			Mode:    "orchestrator",
			Message: "🤖 Using orchestrator mode...",
		}
	}

	// Execute query
	response, state, err := orch.ProcessQuery(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("orchestrator failed: %w", err)
	}

	// Send plan update
	if a.ProgressCh != nil && state.Plan != nil {
		a.ProgressCh <- OrchestratorProgress{
			Mode:    "plan",
			Plan:    state.Plan,
			Message: fmt.Sprintf("📋 Plan: %d steps", len(state.Plan.Steps)),
		}
	}

	// Send step results
	if a.ProgressCh != nil && len(state.StepResults) > 0 {
		a.ProgressCh <- OrchestratorProgress{
			Mode:        "execution",
			StepResults: state.StepResults,
			CurrentStep: len(state.StepResults),
			Message:     fmt.Sprintf("⚙️ Executed %d steps", len(state.StepResults)),
		}
	}

	// Send completion
	if a.ProgressCh != nil {
		a.ProgressCh <- OrchestratorProgress{
			Mode:    "complete",
			Message: fmt.Sprintf("✅ Complete (tokens: %d)", state.TotalTokens),
		}
	}

	return response, nil
}

// CreateOrchestrator creates a new orchestrator instance for the agent
func CreateOrchestrator(prov provider.Provider, toolsReg *tools.Registry, config OrchestratorConfig) *orchestrator.Orchestrator {
	orch := orchestrator.NewOrchestrator(
		prov,
		toolsReg,
		&orchestrator.Config{
			MaxRetries: config.MaxRetries,
			MaxSteps:   config.MaxSteps,
			StateDir:   filepath.Join(os.TempDir(), "aseity-orchestrator-state"),
		},
	)

	orch.EnableParallel = config.Parallel

	return orch
}
