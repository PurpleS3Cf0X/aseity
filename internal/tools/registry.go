package tools

import (
	"context"
	"fmt"

	"github.com/jeanpaul/aseity/internal/provider"
)

type Registry struct {
	tools       map[string]Tool
	autoApprove map[string]bool
}

func NewRegistry(autoApprove []string) *Registry {
	aa := make(map[string]bool)
	for _, name := range autoApprove {
		aa[name] = true
	}
	return &Registry{tools: make(map[string]Tool), autoApprove: aa}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) ToolDefs() []provider.ToolDef {
	defs := make([]provider.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, provider.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}

func (r *Registry) Execute(ctx context.Context, name, args string) (Result, error) {
	t, ok := r.tools[name]
	if !ok {
		return Result{Error: fmt.Sprintf("unknown tool: %s", name)}, nil
	}
	return t.Execute(ctx, args)
}

func (r *Registry) NeedsConfirmation(name string) bool {
	if r.autoApprove[name] {
		return false
	}
	t, ok := r.tools[name]
	if !ok {
		return true
	}
	return t.NeedsConfirmation()
}

func RegisterDefaults(r *Registry) {
	r.Register(&BashTool{})
	r.Register(&FileReadTool{})
	r.Register(&FileWriteTool{})
	r.Register(&FileSearchTool{})
	r.Register(&WebSearchTool{})
	r.Register(&WebFetchTool{})
	r.Register(&SystemInfoTool{})
	r.Register(&ProcessListTool{})
	r.Register(&NetworkInfoTool{})
	r.Register(&ClipboardTool{})
}
