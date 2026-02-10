package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/jeanpaul/aseity/internal/types"
)

// WaitAllAgentsTool waits for multiple background agents to complete.
type WaitAllAgentsTool struct {
	spawner types.AgentSpawner
}

func NewWaitAllAgentsTool(spawner types.AgentSpawner) *WaitAllAgentsTool {
	return &WaitAllAgentsTool{spawner: spawner}
}

func (w *WaitAllAgentsTool) Name() string            { return "wait_all_agents" }
func (w *WaitAllAgentsTool) NeedsConfirmation() bool { return false }
func (w *WaitAllAgentsTool) Description() string {
	return "Wait for multiple background sub-agents to complete and get their aggregated outputs."
}

func (w *WaitAllAgentsTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "integer"},
				"description": "List of Agent IDs to wait for.",
			},
		},
		"required": []string{"agent_ids"},
	}
}

type waitAllResult struct {
	AgentID int    `json:"agent_id"`
	Status  string `json:"status"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

func (w *WaitAllAgentsTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args struct {
		AgentIDs []int `json:"agent_ids"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if len(args.AgentIDs) == 0 {
		return Result{Output: "No agents to wait for."}, nil
	}

	results := make([]waitAllResult, len(args.AgentIDs))
	var wg sync.WaitGroup

	// We use the helper logic from SpawnAgentTool/WaitForAgentTool but repeated
	helper := &SpawnAgentTool{spawner: w.spawner}

	for i, id := range args.AgentIDs {
		wg.Add(1)
		go func(index, agentID int) {
			defer wg.Done()

			// Use the existing waitForAgent logic (which handles timeout/cancellation)
			// But we need to make sure we don't block forever if one agent hangs,
			// though waitForAgent has a timeout.
			res := helper.waitForAgent(ctx, agentID)

			status := "success"
			errMsg := ""
			output := res.Output

			if res.Error != "" {
				status = "failed"
				errMsg = res.Error
				output = "" // Or preserve partial output if available? Result struct doesn't have it distinct from Error.
			}

			results[index] = waitAllResult{
				AgentID: agentID,
				Status:  status,
				Output:  output,
				Error:   errMsg,
			}
		}(i, id)
	}

	wg.Wait()

	// Format output
	var sb strings.Builder
	sb.WriteString("All agents completed:\n\n")

	for _, res := range results {
		if res.Status == "success" {
			sb.WriteString(fmt.Sprintf("✅ Agent #%d: SUCCESS\n", res.AgentID))
			// Truncate output for summary if too long?
			// For recursive planner, we want the full structured output usually.
			sb.WriteString(res.Output)
			sb.WriteString("\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("❌ Agent #%d: FAILED\nError: %s\n\n", res.AgentID, res.Error))
		}
		sb.WriteString("---\n")
	}

	return Result{Output: sb.String()}, nil
}
