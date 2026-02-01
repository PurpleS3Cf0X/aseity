package types

import "context"

// AgentSpawner is the interface both agent.AgentManager implements
// and tools.SpawnAgentTool consumes, breaking the import cycle.
type AgentSpawner interface {
	Spawn(ctx context.Context, task string) (int, error)
	Cancel(id int) error
	Get(id int) (AgentInfo, bool)
	List() []AgentInfo
}

type AgentInfo struct {
	ID     int
	Task   string
	Status string // "running", "done", "failed", "cancelled"
	Output string
}
