package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jeanpaul/aseity/internal/types"
)

// JudgeTool spawns a specialized "Critic" sub-agent to review content.
type JudgeTool struct {
	spawner types.AgentSpawner
}

func NewJudgeTool(spawner types.AgentSpawner) *JudgeTool {
	return &JudgeTool{spawner: spawner}
}

func (j *JudgeTool) Name() string            { return "judge_output" }
func (j *JudgeTool) NeedsConfirmation() bool { return false } // Purely analytical, safe to auto-run
func (j *JudgeTool) Description() string {
	return "Submit content (code, text, plans) to a specialized Critic Agent for review. Returns PASS or FAIL with specific feedback."
}

func (j *JudgeTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"original_goal": map[string]any{
				"type":        "string",
				"description": "The original request or goal that the content attempts to satisfy.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The actual content, code, or output to be reviewed.",
			},
		},
		"required": []string{"original_goal", "content"},
	}
}

type judgeArgs struct {
	OriginalGoal string `json:"original_goal"`
	Content      string `json:"content"`
}

type JudgeResult struct {
	Status   string `json:"status"`   // "pass" or "fail"
	Feedback string `json:"feedback"` // Detailed reasoning
}

func (j *JudgeTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args judgeArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if j.spawner == nil {
		return Result{Error: "agent system not initialized"}, nil
	}

	// Construct the Critic's Strict Persona
	prompt := fmt.Sprintf(`You are a STRICT CRITIC and CODE REVIEWER.
Your job is to evaluate if the CONTENT satisfies the ORIGINAL GOAL.

ORIGINAL GOAL:
"%s"

CONTENT TO REVIEW:
"""
%s
"""

INSTRUCTIONS:
1. Analyze the content for logic errors, security flaws, missing requirements, or hallucinations.
2. Ignore minor formatting issues unless requested.
3. Be harsh but fair.

OUTPUT FORMAT:
Return ONLY a JSON object with this format (no markdown):
{
  "status": "pass" | "fail",
  "feedback": "..."
}
If valid, feedback should be "LGTM".
If invalid, feedback must explain the specific defect.`, args.OriginalGoal, args.Content)

	// Spawn the critic
	// We use "Critic" as the agent name to trigger any specific persona logic if configured,
	// but the specific task prompt above overrides the main directive.
	id, err := j.spawner.Spawn(ctx, prompt, nil, "Critic")
	if err != nil {
		return Result{Error: "failed to spawn critic: " + err.Error()}, nil
	}

	// Poll for result (Timeboxed to 2 minutes for a review)
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			j.spawner.Cancel(id)
			return Result{Error: "critic timed out"}, nil
		case <-ctx.Done():
			j.spawner.Cancel(id)
			return Result{Error: "context cancelled"}, nil
		case <-ticker.C:
			info, ok := j.spawner.Get(id)
			if !ok {
				continue
			}
			if info.Status == "done" {
				// Parse the critic's JSON output
				output := cleanJSON(info.Output)
				var verdict JudgeResult
				if err := json.Unmarshal([]byte(output), &verdict); err != nil {
					// Fallback: If model didn't output JSON, treat as raw feedback
					return Result{Output: fmt.Sprintf("Critic finished but returned malformed JSON. Raw Output:\n%s", info.Output)}, nil
				}

				// Re-serialize strictly to ensure clean tool output
				finalJSON, _ := json.Marshal(verdict)
				return Result{Output: string(finalJSON)}, nil
			}
			if info.Status == "failed" {
				return Result{Error: "critic failed: " + info.Output}, nil
			}
		}
	}
}

// cleanJSON helper to strip potential markdown blocks from model output
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
