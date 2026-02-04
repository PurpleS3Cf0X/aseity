package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jeanpaul/aseity/internal/types"
)

// SpawnAgentTool creates a sub-agent that runs a task autonomously.
type SpawnAgentTool struct {
	mu      sync.Mutex
	spawner types.AgentSpawner
}

func NewSpawnAgentTool(spawner types.AgentSpawner) *SpawnAgentTool {
	return &SpawnAgentTool{spawner: spawner}
}

type spawnAgentArgs struct {
	Task          string   `json:"task"`
	ContextFiles  []string `json:"context_files,omitempty"`
	AgentName     string   `json:"agent_name,omitempty"`
	RequireReview bool     `json:"require_review,omitempty"`
}

func (s *SpawnAgentTool) Name() string            { return "spawn_agent" }
func (s *SpawnAgentTool) NeedsConfirmation() bool { return true }
func (s *SpawnAgentTool) Description() string {
	return "Spawn a sub-agent to handle a complex task autonomously. Optionally specify 'agent_name' to use a custom persona."
}

func (s *SpawnAgentTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "The task description for the sub-agent to accomplish",
			},
			"context_files": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of absolute file paths to load into the sub-agent's context immediately.",
			},
			"agent_name": map[string]any{
				"type":        "string",
				"description": "Optional name of a custom agent persona to use (e.g. 'researcher', 'coder').",
			},
			"require_review": map[string]any{
				"type":        "boolean",
				"description": "If true, a 'Critic' agent will verify the output and request corrections if needed (auto-loop).",
			},
		},
		"required": []string{"task"},
	}
}

func (s *SpawnAgentTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args spawnAgentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if s.spawner == nil {
		return Result{Error: "agent manager not initialized"}, nil
	}

	maxAttempts := 1
	if args.RequireReview {
		maxAttempts = 3
	}

	currentTask := args.Task
	var lastOutput string

	judge := NewJudgeTool(s.spawner)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 1. Prepare Task Wrapper
		finalTask := currentTask
		if args.AgentName != "" {
			finalTask = fmt.Sprintf("You are acting as the '%s' agent. Your specific objective is:\n%s\n\nINSTRUCTIONS:\n1. Analyze the request.\n2. Create a plan in a <thought> block.\n3. Execute the plan effectively.", args.AgentName, currentTask)
		}

		if attempt > 1 {
			// Add retry context
			finalTask = fmt.Sprintf("%s\n\n[SYSTEM]: This is attempt #%d. Your previous output was REJECTED. Please fix the following issues:\n%s", finalTask, attempt, lastOutput)
		}

		// 2. Spawn Worker
		id, err := s.spawner.Spawn(ctx, finalTask, args.ContextFiles, args.AgentName)
		if err != nil {
			return Result{Error: err.Error()}, nil
		}

		// 3. Wait for Worker
		workerRes := s.waitForAgent(ctx, id)
		if workerRes.Error != "" {
			return workerRes, nil // Runtime error in agent
		}
		lastOutput = workerRes.Output

		// If review not required, done.
		if !args.RequireReview {
			return workerRes, nil
		}

		// 4. Spawn Judge
		judgeArgs := map[string]string{
			"original_goal": args.Task,
			"content":       workerRes.Output,
		}
		judgeArgsBytes, _ := json.Marshal(judgeArgs)

		judgeRes, err := judge.Execute(ctx, string(judgeArgsBytes))
		if err != nil {
			// Judge failed system-wise, warn but return worker result?
			// Or fail? Let's return worker result with a warning.
			return Result{Output: fmt.Sprintf("%s\n\n(Warning: Verification failed: %v)", workerRes.Output, err)}, nil
		}
		if judgeRes.Error != "" {
			return Result{Output: fmt.Sprintf("%s\n\n(Warning: Verification error: %s)", workerRes.Output, judgeRes.Error)}, nil
		}

		// 5. Parse Judge Verdict
		var verdict JudgeResult
		if err := json.Unmarshal([]byte(judgeRes.Output), &verdict); err != nil {
			// Malformed verdict, assume pass or manual review needed
			return Result{Output: fmt.Sprintf("%s\n\n(Warning: Malformed verification: %s)", workerRes.Output, judgeRes.Output)}, nil
		}

		if verdict.Status == "pass" {
			return Result{Output: fmt.Sprintf("%s\n\n[Verified PASS by Critic]", workerRes.Output)}, nil
		}

		// 6. Handle Fail -> Loop
		// Update for next iteration
		lastOutput = verdict.Feedback // This becomes the "issues" passed to next attempt
		// Continue loop
	}

	return Result{Error: fmt.Sprintf("Auto-Verification failed after %d attempts. Last feedback: %s", maxAttempts, lastOutput)}, nil
}

func (s *SpawnAgentTool) waitForAgent(ctx context.Context, id int) Result {
	timeout := time.After(10 * time.Minute) // Increased for loop
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			s.spawner.Cancel(id)
			return Result{Error: fmt.Sprintf("Agent #%d timed out", id)}
		case <-ctx.Done():
			s.spawner.Cancel(id)
			return Result{Error: fmt.Sprintf("Agent #%d cancelled", id)}
		case <-ticker.C:
			info, ok := s.spawner.Get(id)
			if !ok {
				continue
			}
			if info.Status != "running" {
				if info.Status == "failed" {
					return Result{Error: fmt.Sprintf("Agent #%d failed: %s", id, info.Output)}
				}
				// Success
				return Result{Output: info.Output}
			}
		}
	}
}

// ListAgentsTool lists all sub-agents and their status.
type ListAgentsTool struct {
	spawner types.AgentSpawner
}

func NewListAgentsTool(spawner types.AgentSpawner) *ListAgentsTool {
	return &ListAgentsTool{spawner: spawner}
}

func (l *ListAgentsTool) Name() string            { return "list_agents" }
func (l *ListAgentsTool) NeedsConfirmation() bool { return false }
func (l *ListAgentsTool) Description() string {
	return "List all sub-agents and their current status (running, done, failed, cancelled)."
}

func (l *ListAgentsTool) Parameters() any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (l *ListAgentsTool) Execute(_ context.Context, _ string) (Result, error) {
	agents := l.spawner.List()
	if len(agents) == 0 {
		return Result{Output: "No sub-agents have been spawned."}, nil
	}

	var sb strings.Builder
	for _, a := range agents {
		fmt.Fprintf(&sb, "Agent #%d [%s]: %s\n", a.ID, a.Status, truncateStr(a.Task, 80))
	}
	return Result{Output: sb.String()}, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
