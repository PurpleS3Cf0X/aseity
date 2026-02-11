package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// AgentState represents the complete state of a multi-agent execution
type AgentState struct {
	// Input
	OriginalQuery string    `json:"original_query"`
	Timestamp     time.Time `json:"timestamp"`
	SessionID     string    `json:"session_id"`

	// Phase 1: Intent
	Intent *IntentOutput `json:"intent,omitempty"`

	// Phase 2: Planning
	Plan *Plan `json:"plan,omitempty"`

	// Phase 3: Execution
	StepResults []StepResult `json:"step_results,omitempty"`

	// Phase 4: Validation
	Validation *ValidationResult `json:"validation,omitempty"`

	// Phase 5: Synthesis
	FinalResponse string `json:"final_response,omitempty"`

	// Metadata
	CurrentPhase string   `json:"current_phase"`
	RetryCount   int      `json:"retry_count"`
	TotalTokens  int      `json:"total_tokens"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// IntentOutput represents the structured output from Phase 1
type IntentOutput struct {
	Reasoning     string   `json:"reasoning"`
	IntentType    string   `json:"intent_type"`
	RequiresTools bool     `json:"requires_tools"`
	Entities      []string `json:"entities"`
	Complexity    string   `json:"complexity"`
}

// Plan represents the structured output from Phase 2
type Plan struct {
	Steps           []PlanStep `json:"steps"`
	ExpectedOutcome string     `json:"expected_outcome"`
}

// PlanStep represents a single step in the execution plan
type PlanStep struct {
	StepNumber int                    `json:"step_number"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Reasoning  string                 `json:"reasoning"`
	DependsOn  []int                  `json:"depends_on,omitempty"`
}

// StepResult represents the output from executing a single step
type StepResult struct {
	StepNumber  int    `json:"step_number"`
	Status      string `json:"status"` // "success" | "failure"
	Result      string `json:"result"`
	Observation string `json:"observation"`
	Error       string `json:"error,omitempty"`
	Duration    int64  `json:"duration_ms"`
}

// ValidationResult represents the output from Phase 4
type ValidationResult struct {
	IntentFulfilled    bool     `json:"intent_fulfilled"`
	MissingInformation []string `json:"missing_information"`
	Confidence         int      `json:"confidence"`
	Recommendation     string   `json:"recommendation"` // "proceed" | "retry" | "replan"
}

// NewAgentState creates a new agent state with initialized values
func NewAgentState(query string) *AgentState {
	return &AgentState{
		OriginalQuery: query,
		Timestamp:     time.Now(),
		SessionID:     uuid.New().String(),
		CurrentPhase:  "init",
		RetryCount:    0,
		TotalTokens:   0,
		Errors:        []string{},
		Warnings:      []string{},
		StepResults:   []StepResult{},
	}
}

// Save persists the state to disk for debugging and replay
func (s *AgentState) Save(dir string) error {
	if dir == "" {
		dir = os.TempDir()
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Sanitize SessionID to prevent path traversal
	safeID := filepath.Base(s.SessionID)
	filename := filepath.Join(dir, fmt.Sprintf("%s.json", safeID))
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Load reads a state from disk
func LoadState(filename string) (*AgentState, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// AddError records an error in the state
func (s *AgentState) AddError(err error) {
	if err != nil {
		s.Errors = append(s.Errors, fmt.Sprintf("[%s] %s", s.CurrentPhase, err.Error()))
	}
}

// AddWarning records a warning in the state
func (s *AgentState) AddWarning(warning string) {
	s.Warnings = append(s.Warnings, fmt.Sprintf("[%s] %s", s.CurrentPhase, warning))
}

// SetPhase updates the current phase
func (s *AgentState) SetPhase(phase string) {
	s.CurrentPhase = phase
}

// IncrementRetry increments the retry counter
func (s *AgentState) IncrementRetry() {
	s.RetryCount++
}

// AddTokens adds to the total token count
func (s *AgentState) AddTokens(tokens int) {
	s.TotalTokens += tokens
}
