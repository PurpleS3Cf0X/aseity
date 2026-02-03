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

// Streamer is an optional interface for tools that can stream output
type Streamer interface {
	ExecuteStream(ctx context.Context, args string, callback func(string)) (Result, error)
}
