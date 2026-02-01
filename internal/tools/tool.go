package tools

import "context"

type Result struct {
	Output string
	Error  string
}

type Tool interface {
	Name() string
	Description() string
	Parameters() any
	Execute(ctx context.Context, args string) (Result, error)
	NeedsConfirmation() bool
}
