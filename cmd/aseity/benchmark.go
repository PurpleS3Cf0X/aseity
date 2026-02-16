package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeanpaul/aseity/internal/config"
	"github.com/jeanpaul/aseity/internal/orchestrator"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/tui"
)

type BenchmarkResult struct {
	Query          string
	Success        bool
	Steps          int
	Duration       time.Duration
	Grade          string // "A", "B", "C", "F"
	Reasoning      string
	Hallucinations int
}

type GoldenQuery struct {
	ID         string `json:"id"`
	Query      string `json:"query"`
	Difficulty string `json:"difficulty"` // "simple", "medium", "complex"
}

// cmdBenchmark runs the benchmark suite
func cmdBenchmark(datasetPath string, judgeModel string) {
	fmt.Println(tui.BannerStyle.Render("  Aseity Industrial Benchmark"))
	fmt.Println()

	// 1. Load Dataset
	queries := loadGoldenDataset(datasetPath)
	fmt.Printf("  Loaded %d queries from %s\n", len(queries), datasetPath)

	// 2. Setup Environment (Headless Agent)
	cfg, err := config.Load()
	if err != nil {
		fatal("config error: %s", err)
	}

	// Use default provider/model for the AGENT
	// Use judgeModel for the JUDGE
	agentProvName := cfg.DefaultProvider
	agentModelName := cfg.DefaultModel
	judgeProvName := agentProvName // Assume same provider for now, or configurable

	fmt.Printf("  Agent: %s/%s\n", agentProvName, agentModelName)
	fmt.Printf("  Judge: %s/%s\n\n", judgeProvName, judgeModel)

	// Initialize Provider for Agent
	agentProv, err := makeProvider(cfg, agentProvName, agentModelName)
	if err != nil {
		fatal("failed to create agent provider: %s", err)
	}
	agentProv = provider.WithRetry(agentProv, 3)

	// Initialize Provider for Judge
	judgeProv, err := makeProvider(cfg, judgeProvName, judgeModel)
	if err != nil {
		// Fallback to agent provider if judge model not found specifically?
		// Or try to use same provider with different model name?
		// Provider impls usually take model in constructor.
		// So we need a new provider instance for the judge model.
		// But makeProvider typically uses config.
		// We'll trust the user has the judge model available on the default provider for now.
		// If using OpenAI, we might need a separate config entry if the judge model implies a different provider.
		// For simplicity, assume judge uses same provider type/credentials as agent default, just different model name.
		// Refactoring makeProvider to override model name would be cleaner, but we can do it manually here.
		pcfg, _ := cfg.ProviderFor(judgeProvName)
		if pcfg.Type == "ollama" || pcfg.Type == "openai" {
			// Ollama uses OpenAI-compatible API
			baseURL := pcfg.BaseURL
			if baseURL == "" && pcfg.Type == "ollama" {
				baseURL = "http://localhost:11434/v1"
			}
			judgeProv = provider.NewOpenAI(judgeProvName, baseURL, pcfg.APIKey, judgeModel)
		} else if pcfg.Type == "anthropic" {
			judgeProv = provider.NewAnthropic(pcfg.APIKey, judgeModel)
		} else if pcfg.Type == "google" {
			judgeProv = provider.NewGoogle(pcfg.APIKey, judgeModel)
		} else {
			fatal("Unsupported provider for judge fallback")
		}
	}

	// Setup Tools
	toolReg := tools.NewRegistry(nil, true) // Auto-approve all for benchmark
	tools.RegisterDefaults(toolReg, cfg.Tools.AllowedCommands, cfg.Tools.DisallowedCommands)

	// Create Orchestrator
	orch := orchestrator.NewOrchestrator(agentProv, toolReg, &orchestrator.Config{
		MaxRetries: 3,
		MaxSteps:   10,
		StateDir:   "", // No debug save for benchmark to save disk? Or maybe useful?
	})

	var results []BenchmarkResult

	// 3. Execution Loop
	for i, q := range queries {
		fmt.Printf("  [%d/%d] Running: %s... ", i+1, len(queries), tui.UserLabelStyle.Render(q.ID))

		start := time.Now()

		// Run Agent
		// We use a fresh context for each query
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		_, state, err := orch.ProcessQuery(ctx, q.Query)
		cancel()

		duration := time.Since(start)

		if err != nil {
			fmt.Printf("%s\n", tui.ErrorStyle.Render("ERROR: "+err.Error()))
			results = append(results, BenchmarkResult{
				Query:     q.Query,
				Success:   false,
				Duration:  duration,
				Grade:     "F",
				Reasoning: fmt.Sprintf("System Error: %v", err),
			})
			continue
		}

		// 4. Judge Result
		grade, reasoning := judgeResult(context.Background(), judgeProv, q.Query, state)

		hallucinations := countHallucinations(state)

		fmt.Printf("%s (%s)\n", tui.BannerStyle.Render(grade), duration.Round(time.Second))

		results = append(results, BenchmarkResult{
			Query:          q.Query,
			Success:        grade == "A" || grade == "B", // Roughly
			Steps:          len(state.StepResults),
			Duration:       duration,
			Grade:          grade,
			Reasoning:      reasoning,
			Hallucinations: hallucinations,
		})
	}

	// 5. Report
	printReport(results)
}

func loadGoldenDataset(path string) []GoldenQuery {
	if path == "" {
		// Return hardcoded default if no file
		return []GoldenQuery{
			{ID: "F001", Difficulty: "Simple", Query: "What is the capital of France?"},
			{ID: "C001", Difficulty: "Complex", Query: "Fetch the latest version of Go from their website and summarize its 3 main release notes."},
			{ID: "M001", Difficulty: "Medium", Query: "Find out who the CEO of Anthropic is and what company they worked for previously."},
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fatal("Failed to read dataset: %v", err)
	}

	var queries []GoldenQuery
	if err := json.Unmarshal(data, &queries); err != nil {
		fatal("Failed to parse dataset: %v", err)
	}
	return queries
}

func judgeResult(ctx context.Context, prov provider.Provider, query string, state *orchestrator.AgentState) (string, string) {
	// Serialize State Trajectory
	// We only need steps and validation
	trajectory := ""
	for i, s := range state.StepResults {
		status := "SUCCESS"
		if s.Status != "success" {
			status = "FAILURE"
		}

		action := "unknown"
		if state.Plan != nil && i < len(state.Plan.Steps) {
			action = state.Plan.Steps[i].Action
		}

		trajectory += fmt.Sprintf("Step %d [%s] %s: %s\n", s.StepNumber, status, action, s.Observation)
	}

	finalResponse := state.FinalResponse

	prompt := fmt.Sprintf(`You are an AI Judge evaluating an autonomous agent's performance.

Query: "%s"

Agent Trajectory:
%s

Final Response:
%s

Task: Grade the agent's performance on a scale of A, B, C, F.
- A: Perfect execution. Correct answer, efficient steps, no errors.
- B: Good. Correct answer, but maybe inefficient or 1 minor retriable error.
- C: Acceptable. Partially correct or very inefficient.
- F: Failed. Wrong answer, hallucinated tools, or crashed.

Output JSON only:
{
  "grade": "A|B|C|F",
  "reasoning": "Short explanation"
}`, query, trajectory, finalResponse)

	ch, err := prov.Chat(ctx, []provider.Message{{Role: "user", Content: prompt}}, nil)
	if err != nil {
		return "N/A", "Judge Error: " + err.Error()
	}

	var builder strings.Builder
	for chunk := range ch {
		builder.WriteString(chunk.Delta)
	}

	output := builder.String()
	// Extract JSON
	output = strings.TrimPrefix(output, "```json")
	output = strings.TrimSuffix(output, "```") // Rough stripping
	// Better json extraction needed if model is chatty
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start != -1 && end != -1 {
		output = output[start : end+1]
	}

	var result struct {
		Grade     string `json:"grade"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return "N/A", "Judge Parse Error: " + err.Error() + " | Raw: " + output
	}
	return result.Grade, result.Reasoning
}

func countHallucinations(state *orchestrator.AgentState) int {
	count := 0
	for _, s := range state.StepResults {
		if strings.Contains(s.Observation, "unknown tool") || strings.Contains(s.Observation, "invalid arguments") {
			count++
		}
	}
	return count
}

func printReport(results []BenchmarkResult) {
	fmt.Println()
	fmt.Println(tui.BannerStyle.Render("  Benchmark Report"))
	fmt.Println("  ------------------------------------------------")

	total := len(results)
	passed := 0
	totalDuration := time.Duration(0)

	for _, r := range results {
		if r.Success {
			passed++
		}
		totalDuration += r.Duration

		icon := "✓"
		color := tui.Green
		if !r.Success {
			icon = "✗"
			color = tui.Red
		}

		fmt.Printf("  %s [%s] %s\n",
			tui.HelpStyle.Foreground(color).Render(icon),
			tui.ToolCallStyle.Render(r.Grade),
			tui.UserLabelStyle.Render(r.Query),
		)
		fmt.Printf("      Steps: %d | Time: %s | Hallucinations: %d\n", r.Steps, r.Duration.Round(time.Millisecond), r.Hallucinations)
		fmt.Printf("      Reason: %s\n\n", tui.HelpStyle.Render(r.Reasoning))
	}

	successRate := float64(passed) / float64(total) * 100
	fmt.Println("  ------------------------------------------------")
	fmt.Printf("  Success Rate: %.1f%%\n", successRate)
	fmt.Printf("  Total Time:   %s\n", totalDuration.Round(time.Second))
}
