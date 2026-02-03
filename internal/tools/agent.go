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
	Task         string   `json:"task"`
	ContextFiles []string `json:"context_files,omitempty"`
}

func (s *SpawnAgentTool) Name() string            { return "spawn_agent" }
func (s *SpawnAgentTool) NeedsConfirmation() bool { return true }
func (s *SpawnAgentTool) Description() string {
	return "Spawn a sub-agent to handle a complex task autonomously. The sub-agent has access to all tools and will return its output when done. Use for parallel or delegated work."
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

	id, err := s.spawner.Spawn(ctx, args.Task, args.ContextFiles)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}

	// Poll for completion
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			s.spawner.Cancel(id)
			return Result{Output: fmt.Sprintf("Agent #%d timed out after 5 minutes", id)}, nil
		case <-ctx.Done():
			s.spawner.Cancel(id)
			return Result{Output: fmt.Sprintf("Agent #%d cancelled", id)}, nil
		case <-ticker.C:
			info, ok := s.spawner.Get(id)
			if !ok {
				continue
			}
			if info.Status != "running" {
				if info.Status == "failed" {
					return Result{Error: fmt.Sprintf("Agent #%d failed: %s", id, info.Output)}, nil
				}
				output := info.Output
				if len(output) > 4000 {
					output = output[:4000] + "\n... (truncated)"
				}
				return Result{Output: fmt.Sprintf("Agent #%d completed:\n%s", id, output)}, nil
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
