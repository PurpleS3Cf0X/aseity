package tools

import (
	"context"
	"fmt"

	"github.com/jeanpaul/aseity/internal/provider"
)

type Registry struct {
	tools       map[string]Tool
	autoApprove map[string]bool
	allowAll    bool
}

func NewRegistry(autoApprove []string, allowAll bool) *Registry {
	aa := make(map[string]bool)
	for _, name := range autoApprove {
		aa[name] = true
	}
	return &Registry{tools: make(map[string]Tool), autoApprove: aa, allowAll: allowAll}
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

func (r *Registry) Execute(ctx context.Context, name, args string, callback func(string)) (Result, error) {
	t, ok := r.tools[name]
	if !ok {
		return Result{Error: fmt.Sprintf("unknown tool: %s", name)}, nil
	}

	if callback != nil {
		if s, ok := t.(Streamer); ok {
			return s.ExecuteStream(ctx, args, callback)
		}
	}
	return t.Execute(ctx, args)
}

func (r *Registry) NeedsConfirmation(name string) bool {
	if r.allowAll {
		return false
	}
	if r.autoApprove[name] {
		return false
	}
	t, ok := r.tools[name]
	if !ok {
		return true
	}
	return t.NeedsConfirmation()
}

// RegisterDefaults registers all built-in tools. Pass command lists from config.
func RegisterDefaults(r *Registry, allowedCmds, disallowedCmds []string) {
	r.Register(&BashTool{
		AllowedCommands:    allowedCmds,
		DisallowedCommands: disallowedCmds,
	})
	r.Register(&FileReadTool{})
	r.Register(&FileWriteTool{})
	r.Register(&FileSearchTool{})
	r.Register(&WebSearchTool{})
	r.Register(&WebFetchTool{})
}
