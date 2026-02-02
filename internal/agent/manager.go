package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/types"
)

type subAgentStatus int

const (
	subAgentRunning subAgentStatus = iota
	subAgentDone
	subAgentFailed
	subAgentCancelled
)

type subAgentInfo struct {
	id     int
	task   string
	status subAgentStatus
	output string
	cancel context.CancelFunc
}

// AgentManager implements types.AgentSpawner.
type AgentManager struct {
	mu      sync.Mutex
	agents  map[int]*subAgentInfo
	nextID  atomic.Int32
	maxConc int
	prov    provider.Provider
	toolReg *tools.Registry
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

func (am *AgentManager) Spawn(ctx context.Context, task string) (int, error) {
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
		id:     id,
		task:   task,
		status: subAgentRunning,
		cancel: cancel,
	}
	am.agents[id] = info
	am.mu.Unlock()

	go func() {
		ag := New(am.prov, am.toolReg)
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
