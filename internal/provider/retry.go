package provider

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// RetryProvider wraps a Provider with exponential backoff retry logic.
type RetryProvider struct {
	inner      Provider
	maxRetries int
	baseDelay  time.Duration
}

func WithRetry(p Provider, maxRetries int) *RetryProvider {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &RetryProvider{inner: p, maxRetries: maxRetries, baseDelay: 500 * time.Millisecond}
}

func (r *RetryProvider) Name() string { return r.inner.Name() }

func (r *RetryProvider) ModelName() string { return r.inner.ModelName() }

func (r *RetryProvider) Models(ctx context.Context) ([]string, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		models, err := r.inner.Models(ctx)
		if err == nil {
			return models, nil
		}
		lastErr = err
		if !r.isRetryable(err) || attempt == r.maxRetries {
			break
		}
		if err := r.backoff(ctx, attempt); err != nil {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func (r *RetryProvider) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamChunk, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		ch, err := r.inner.Chat(ctx, msgs, tools)
		if err == nil {
			return ch, nil
		}
		lastErr = err
		if !r.isRetryable(err) || attempt == r.maxRetries {
			break
		}
		if err := r.backoff(ctx, attempt); err != nil {
			return nil, lastErr
		}
	}
	return nil, fmt.Errorf("after %d retries: %w", r.maxRetries, lastErr)
}

func (r *RetryProvider) isRetryable(err error) bool {
	msg := err.Error()
	// Retry on rate limits, server errors, connection issues
	for _, s := range []string{"429", "500", "502", "503", "529", "connection refused", "timeout", "deadline exceeded", "EOF", "reset by peer"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

func (r *RetryProvider) backoff(ctx context.Context, attempt int) error {
	delay := time.Duration(float64(r.baseDelay) * math.Pow(2, float64(attempt)))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
