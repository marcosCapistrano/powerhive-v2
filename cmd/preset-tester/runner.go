package main

import (
	"context"
	"fmt"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

// Runner executes the test suite.
type Runner struct {
	client       *vnish.HTTPClient
	pollInterval time.Duration
	timeout      time.Duration
	results      []TestResult
}

// NewRunner creates a new test runner.
func NewRunner(client *vnish.HTTPClient, pollInterval, timeout time.Duration) *Runner {
	return &Runner{
		client:       client,
		pollInterval: pollInterval,
		timeout:      timeout,
		results:      make([]TestResult, 0),
	}
}

// Run executes all test cases in the suite.
func (r *Runner) Run(ctx context.Context, suite []TestCase) []TestResult {
	r.results = make([]TestResult, 0, len(suite))

	for i, tc := range suite {
		select {
		case <-ctx.Done():
			fmt.Printf("\n[ABORT] Test suite cancelled\n")
			return r.results
		default:
		}

		fmt.Printf("\n[%d/%d] TEST: %s\n", i+1, len(suite), tc.Name)
		fmt.Printf("        %s\n", tc.Description)

		result := r.runTest(ctx, &tc)
		r.results = append(r.results, result)

		if result.Error != nil {
			fmt.Printf("        ERROR: %v\n", result.Error)
		} else if result.RebootTriggered {
			fmt.Printf("        REBOOT REQUIRED - Aborting test suite\n")
			return r.results
		} else if result.RestartTriggered {
			fmt.Printf("        RESTART TRIGGERED - Recovering...\n")
			if err := r.recover(ctx); err != nil {
				fmt.Printf("        RECOVERY FAILED: %v\n", err)
				return r.results
			}
			fmt.Printf("        Recovered, continuing...\n")
		} else {
			fmt.Printf("        PASS (no restart triggered)\n")
		}
	}

	return r.results
}

// runTest executes a single test case.
func (r *Runner) runTest(ctx context.Context, tc *TestCase) TestResult {
	result := TestResult{
		TestCase:  tc,
		StartTime: time.Now(),
	}

	for _, step := range tc.Steps {
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			result.EndTime = time.Now()
			return result
		default:
		}

		stepResult := r.executeStep(ctx, step)
		result.StepResults = append(result.StepResults, stepResult)

		if stepResult.Error != nil {
			result.Error = stepResult.Error
			break
		}

		if stepResult.RebootRequired {
			result.RebootTriggered = true
			break
		}

		// If restart is required at ANY step, fail the test and trigger recovery
		if stepResult.RestartRequired {
			result.RestartTriggered = true
			break
		}
	}

	result.EndTime = time.Now()
	return result
}

// executeStep executes a single test step.
func (r *Runner) executeStep(ctx context.Context, step TestStep) StepResult {
	result := StepResult{
		Step:      step,
		Timestamp: time.Now(),
	}

	switch step.Action {
	case ActionSetPreset:
		fmt.Printf("        ... setting preset to %s\n", step.Preset)
		if err := r.client.SetPreset(ctx, step.Preset); err != nil {
			result.Error = fmt.Errorf("SetPreset(%s): %w", step.Preset, err)
			return result
		}
		// Small delay to let the API process
		time.Sleep(100 * time.Millisecond)
		// Check immediate status
		status, err := r.getStatus(ctx)
		if err != nil {
			result.Error = err
			return result
		}
		result.RestartRequired = status.RestartRequired
		result.RebootRequired = status.RebootRequired
		result.MinerState = status.MinerState
		result.CurrentPreset = status.CurrentPreset

	case ActionWait:
		if step.WaitTime > 0 {
			fmt.Printf("        ... waiting %s\n", step.WaitTime)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
			case <-time.After(step.WaitTime):
			}
		}

	case ActionWaitStable:
		fmt.Printf("        ... waiting for stability\n")
		status, err := r.waitForStable(ctx)
		if err != nil {
			result.Error = err
			return result
		}
		result.RestartRequired = status.RestartRequired
		result.RebootRequired = status.RebootRequired
		result.MinerState = status.MinerState
		result.CurrentPreset = status.CurrentPreset

		if status.RestartRequired {
			fmt.Printf("        ... restart_required=true detected!\n")
		} else if status.RebootRequired {
			fmt.Printf("        ... reboot_required=true detected!\n")
		} else {
			fmt.Printf("        ... stable (preset=%s, state=%s)\n", status.CurrentPreset, status.MinerState)
		}

	case ActionCheckRestart:
		status, err := r.getStatus(ctx)
		if err != nil {
			result.Error = err
			return result
		}
		result.RestartRequired = status.RestartRequired
		result.RebootRequired = status.RebootRequired
		result.MinerState = status.MinerState
		result.CurrentPreset = status.CurrentPreset

		if status.RestartRequired {
			fmt.Printf("        ... restart_required=true\n")
		}
		if status.RebootRequired {
			fmt.Printf("        ... reboot_required=true\n")
		}
	}

	return result
}

// MinerState holds current miner state for simplified access.
type MinerState struct {
	MinerState      string
	RestartRequired bool
	RebootRequired  bool
	CurrentPreset   string
	PresetStatus    string
}

// getStatus fetches current miner status.
func (r *Runner) getStatus(ctx context.Context) (*MinerState, error) {
	status, err := r.client.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetStatus: %w", err)
	}

	perf, err := r.client.GetPerfSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetPerfSummary: %w", err)
	}

	return &MinerState{
		MinerState:      status.MinerState,
		RestartRequired: status.RestartRequired,
		RebootRequired:  status.RebootRequired,
		CurrentPreset:   perf.CurrentPreset.Name,
		PresetStatus:    perf.CurrentPreset.Status,
	}, nil
}

// waitForStable polls until miner is stable or timeout.
// If restart_required or reboot_required is detected, returns immediately with that status
// (doesn't wait for timeout since those flags won't clear without action).
func (r *Runner) waitForStable(ctx context.Context) (*MinerState, error) {
	deadline := time.Now().Add(r.timeout)
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for stability after %s", r.timeout)
			}

			status, err := r.getStatus(ctx)
			if err != nil {
				continue // retry on error
			}

			// If restart/reboot is required, return immediately - waiting won't help
			if status.RestartRequired || status.RebootRequired {
				return status, nil
			}

			// Check if miner is in valid operational state
			// Keep waiting if miner is starting, initializing, etc.
			validStates := map[string]bool{
				"running":     true,
				"auto-tuning": true,
				"mining":      true,
			}
			if !validStates[status.MinerState] {
				continue
			}

			return status, nil
		}
	}
}

// recover attempts to recover after a restart was triggered.
func (r *Runner) recover(ctx context.Context) error {
	fmt.Printf("        ... calling RestartMining()\n")
	if err := r.client.RestartMining(ctx); err != nil {
		return fmt.Errorf("RestartMining: %w", err)
	}

	// Wait a bit for the restart to take effect
	time.Sleep(2 * time.Second)

	// Wait for stable
	_, err := r.waitForStable(ctx)
	return err
}

// Results returns all test results.
func (r *Runner) Results() []TestResult {
	return r.results
}
