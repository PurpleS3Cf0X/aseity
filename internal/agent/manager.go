package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/types"
)

const MaxAgentDepth = 3 // prevent circular/recursive agent spawning

type subAgentStatus int

const (
	subAgentRunning subAgentStatus = iota
	subAgentDone
	subAgentFailed
	subAgentCancelled
)

type subAgentInfo struct {
	id        int
	task      string
	status    subAgentStatus
	output    string
	cancel    context.CancelFunc
	createdAt time.Time
	depth     int
}

// AgentManager implements types.AgentSpawner.
type AgentManager struct {
	mu      sync.Mutex
	agents  map[int]*subAgentInfo
	nextID  atomic.Int32
	maxConc int
	prov    provider.Provider
	toolReg *tools.Registry
	depth   int // current nesting depth
}

func NewAgentManager(prov provider.Provider, toolReg *tools.Registry, maxConcurrent int) *AgentManager {
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	return &AgentManager{
		agents:  make(map[int]*subAgentInfo),
		maxConc: maxConcurrent,
		prov:    prov,
		toolReg: toolReg,
	}
}

// NewAgentManagerWithDepth creates a manager that tracks nesting depth.
func NewAgentManagerWithDepth(prov provider.Provider, toolReg *tools.Registry, maxConcurrent, depth int) *AgentManager {
	am := NewAgentManager(prov, toolReg, maxConcurrent)
	am.depth = depth
	return am
}

func (am *AgentManager) Spawn(ctx context.Context, task string, contextFiles []string) (int, error) {
	// Check nesting depth
	if am.depth >= MaxAgentDepth {
		return 0, fmt.Errorf("maximum agent nesting depth (%d) reached — cannot spawn sub-agent", MaxAgentDepth)
	}

	am.mu.Lock()
	running := 0
	for _, a := range am.agents {
		if a.status == subAgentRunning {
			running++
		}
	}
	if running >= am.maxConc {
		am.mu.Unlock()
		return 0, fmt.Errorf("max concurrent agents (%d) reached", am.maxConc)
	}

	id := int(am.nextID.Add(1))
	subCtx, cancel := context.WithCancel(ctx)

	info := &subAgentInfo{
		id:        id,
		task:      task,
		status:    subAgentRunning,
		cancel:    cancel,
		createdAt: time.Now(),
		depth:     am.depth + 1,
	}
	am.agents[id] = info
	am.mu.Unlock()

	go func() {
		ag := NewWithDepth(am.prov, am.toolReg, am.depth+1)

		// Pre-load context files if provided
		if len(contextFiles) > 0 {
			var contextBuilder strings.Builder
			contextBuilder.WriteString("I have loaded the following context files for you:\n\n")
			for _, path := range contextFiles {
				// We reuse the FileReadTool logic or just simple os.ReadFile since agent is internal
				// But to be consistent with tools, let's just read it directly here.
				// For safety, we should probably check if file exists.
				// We'll limit the size to avoid blowing up context.
				// Read file directly
				content, err := os.ReadFile(path)
				if err != nil {
					contextBuilder.WriteString(fmt.Sprintf("Error reading %s: %s\n", path, err))
				} else {
					str := string(content)
					if len(str) > 5000 {
						str = str[:5000] + "\n... (truncated)"
					}
					contextBuilder.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", path, str))
				}
			}
			ag.Conversation().AddUser(contextBuilder.String())
		}

		// Sub-agents auto-approve all tools
		go func() {
			for {
				select {
				case ag.ConfirmCh <- true:
				case <-subCtx.Done():
					return
				}
			}
		}()

		events := make(chan Event, 64)
		var output strings.Builder

		go ag.Send(subCtx, task, events)

		for evt := range events {
			if evt.Type == EventDelta {
				output.WriteString(evt.Text)
			}
			if evt.Type == EventError {
				am.mu.Lock()
				info.status = subAgentFailed
				info.output = evt.Error
				am.mu.Unlock()
				return
			}
		}

		am.mu.Lock()
		info.status = subAgentDone
		info.output = output.String()
		am.mu.Unlock()
	}()

	return id, nil
}

func (am *AgentManager) Cancel(id int) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	info, ok := am.agents[id]
	if !ok {
		return fmt.Errorf("agent #%d not found", id)
	}
	if info.status == subAgentRunning {
		info.cancel()
		info.status = subAgentCancelled
	}
	return nil
}

func (am *AgentManager) List() []types.AgentInfo {
	am.mu.Lock()
	defer am.mu.Unlock()
	result := make([]types.AgentInfo, 0, len(am.agents))
	for _, a := range am.agents {
		result = append(result, types.AgentInfo{
			ID:     a.id,
			Task:   a.task,
			Status: am.statusString(a.status),
			Output: a.output,
		})
	}
	return result
}

func (am *AgentManager) Get(id int) (types.AgentInfo, bool) {
	am.mu.Lock()
	defer am.mu.Unlock()
	info, ok := am.agents[id]
	if !ok {
		return types.AgentInfo{}, false
	}
	return types.AgentInfo{
		ID:     info.id,
		Task:   info.task,
		Status: am.statusString(info.status),
		Output: info.output,
	}, true
}

// Cleanup removes completed/failed/cancelled agents older than the given duration.
func (am *AgentManager) Cleanup(maxAge time.Duration) int {
	am.mu.Lock()
	defer am.mu.Unlock()
	removed := 0
	cutoff := time.Now().Add(-maxAge)
	for id, a := range am.agents {
		if a.status != subAgentRunning && a.createdAt.Before(cutoff) {
			delete(am.agents, id)
			removed++
		}
	}
	return removed
}

func (am *AgentManager) statusString(s subAgentStatus) string {
	switch s {
	case subAgentRunning:
		return "running"
	case subAgentDone:
		return "done"
	case subAgentFailed:
		return "failed"
	case subAgentCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}
