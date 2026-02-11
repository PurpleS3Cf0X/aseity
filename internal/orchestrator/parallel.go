package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// executePlanParallel executes plan steps with dependency-aware parallelization
func (o *Orchestrator) executePlanParallel(ctx context.Context, plan *Plan) ([]StepResult, error) {
	results := make([]StepResult, len(plan.Steps))

	// Group steps into execution waves based on dependencies
	waves := groupStepsByDependencies(plan.Steps)

	// Execute each wave
	for waveIdx, wave := range waves {
		if err := o.executeWave(ctx, wave, results); err != nil {
			return results, fmt.Errorf("wave %d failed: %w", waveIdx+1, err)
		}
	}

	return results, nil
}

// executeWave executes a group of independent steps concurrently
func (o *Orchestrator) executeWave(ctx context.Context, steps []PlanStep, results []StepResult) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(steps))

	// Semaphore to limit concurrent executions (prevent resource exhaustion)
	sem := make(chan struct{}, 3) // Max 3 concurrent steps

	for _, step := range steps {
		wg.Add(1)

		go func(s PlanStep) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute with panic recovery
			if err := o.executeStepSafe(ctx, s, results); err != nil {
				errChan <- err
			}
		}(step)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("wave execution failed: %d errors: %v", len(errs), errs[0])
	}

	return nil
}

// executeStepSafe executes a step with full error handling and panic recovery
func (o *Orchestrator) executeStepSafe(ctx context.Context, step PlanStep, results []StepResult) (err error) {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in step %d: %v", step.StepNumber, r)
			results[step.StepNumber-1] = StepResult{
				StepNumber:  step.StepNumber,
				Status:      "failure",
				Result:      "",
				Observation: fmt.Sprintf("Panic: %v", r),
				Error:       fmt.Sprintf("panic: %v", r),
				Duration:    0,
			}
		}
	}()

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create step-specific timeout
	stepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()

	// Get previous results for parameter extraction
	previousResults := make([]StepResult, step.StepNumber-1)
	for i := 0; i < step.StepNumber-1; i++ {
		previousResults[i] = results[i]
	}

	// Execute the tool with previous results for parameter extraction
	result, execErr := o.executeStep(stepCtx, step, previousResults)

	duration := time.Since(start).Milliseconds()

	if execErr != nil {
		results[step.StepNumber-1] = StepResult{
			StepNumber:  step.StepNumber,
			Status:      "failure",
			Result:      "",
			Observation: fmt.Sprintf("Step failed: %s", execErr.Error()),
			Error:       execErr.Error(),
			Duration:    duration,
		}
		return execErr
	}

	results[step.StepNumber-1] = StepResult{
		StepNumber:  step.StepNumber,
		Status:      "success",
		Result:      result,
		Observation: fmt.Sprintf("Executed %s successfully", step.Action),
		Duration:    duration,
	}

	return nil
}

// groupStepsByDependencies groups steps into waves based on their dependencies
func groupStepsByDependencies(steps []PlanStep) [][]PlanStep {
	var waves [][]PlanStep
	executed := make(map[int]bool)

	for len(executed) < len(steps) {
		var wave []PlanStep

		for _, step := range steps {
			// Skip if already executed
			if executed[step.StepNumber] {
				continue
			}

			// If no dependencies specified, treat as sequential (depends on all previous)
			deps := step.DependsOn
			if len(deps) == 0 && step.StepNumber > 1 {
				// Sequential: depends on immediately previous step
				deps = []int{step.StepNumber - 1}
			}

			// Check if all dependencies are satisfied
			canExecute := true
			for _, dep := range deps {
				if !executed[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				wave = append(wave, step)
				executed[step.StepNumber] = true
			}
		}

		if len(wave) == 0 {
			// Circular dependency or invalid plan - execute remaining sequentially
			for _, step := range steps {
				if !executed[step.StepNumber] {
					wave = append(wave, step)
					executed[step.StepNumber] = true
					break
				}
			}
		}

		if len(wave) > 0 {
			waves = append(waves, wave)
		}
	}

	return waves
}
