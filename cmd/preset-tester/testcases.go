package main

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// TestCase defines a single preset change test.
type TestCase struct {
	Name        string
	Description string
	Phase       string
	Steps       []TestStep
}

// TestStep is a single action within a test case.
type TestStep struct {
	Action      StepAction
	Preset      string        // target preset (for SetPreset action)
	WaitTime    time.Duration // wait time (for Wait action)
	WaitStable  bool          // wait for stability (for WaitStable action)
}

// StepAction defines the type of test step.
type StepAction int

const (
	ActionSetPreset StepAction = iota
	ActionWait
	ActionWaitStable
	ActionCheckRestart
)

// TestResult holds the result of a single test.
type TestResult struct {
	TestCase         *TestCase
	StartTime        time.Time
	EndTime          time.Time
	RestartTriggered bool
	RebootTriggered  bool
	Error            error
	StepResults      []StepResult
}

// StepResult holds the result of a single step.
type StepResult struct {
	Step             TestStep
	Timestamp        time.Time
	RestartRequired  bool
	RebootRequired   bool
	MinerState       string
	CurrentPreset    string
	Error            error
}

// Passed returns true if the test passed (no restart/reboot triggered).
func (r *TestResult) Passed() bool {
	return !r.RestartTriggered && !r.RebootTriggered && r.Error == nil
}

// Duration returns how long the test took.
func (r *TestResult) Duration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// GenerateTestSuite creates the full test suite based on available presets.
func GenerateTestSuite(presets []string, minPreset, maxPreset string) []TestCase {
	// Filter presets to the specified range
	filtered := filterPresets(presets, minPreset, maxPreset)
	if len(filtered) < 2 {
		return nil
	}

	var suite []TestCase

	// Phase 1: Baseline tests
	suite = append(suite, generateBaselineTests(filtered)...)

	// Phase 2: Timing tests
	suite = append(suite, generateTimingTests(filtered)...)

	// Phase 3: Direction tests
	suite = append(suite, generateDirectionTests(filtered)...)

	// Phase 4: Magnitude tests
	suite = append(suite, generateMagnitudeTests(filtered)...)

	// Phase 5: Edge case tests
	suite = append(suite, generateEdgeCaseTests(filtered)...)

	return suite
}

// filterPresets returns presets within the min/max range.
// "disabled" preset is included if maxPreset is "disabled" (since disabled = highest power).
func filterPresets(presets []string, minPreset, maxPreset string) []string {
	includeDisabled := maxPreset == "disabled"
	minVal := presetToInt(minPreset)
	maxVal := presetToInt(maxPreset)

	var filtered []string
	for _, p := range presets {
		if p == "disabled" {
			if includeDisabled {
				filtered = append(filtered, p)
			}
			continue
		}
		val := presetToInt(p)
		if val >= minVal && val <= maxVal {
			filtered = append(filtered, p)
		}
	}

	// Sort by power consumption: lowest first, "disabled" (stock) last
	sort.Slice(filtered, func(i, j int) bool {
		return presetToInt(filtered[i]) < presetToInt(filtered[j])
	})

	return filtered
}

func presetToInt(preset string) int {
	if preset == "disabled" {
		return 999999 // "disabled" = stock = highest power consumption
	}
	val, _ := strconv.Atoi(preset)
	return val
}

// generateBaselineTests creates tests for the safe path (wait for stable between changes).
func generateBaselineTests(presets []string) []TestCase {
	if len(presets) < 2 {
		return nil
	}

	low := presets[0]
	high := presets[len(presets)-1]
	mid := presets[len(presets)/2]

	return []TestCase{
		{
			Name:        "baseline_low_to_high",
			Description: fmt.Sprintf("Stable path: %s -> %s (wait for stable)", low, high),
			Phase:       "baseline",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionCheckRestart},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "baseline_high_to_low",
			Description: fmt.Sprintf("Stable path: %s -> %s (wait for stable)", high, low),
			Phase:       "baseline",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionCheckRestart},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "baseline_mid_to_mid",
			Description: fmt.Sprintf("Stable path: %s -> %s -> %s", low, mid, high),
			Phase:       "baseline",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionCheckRestart},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionCheckRestart},
			},
		},
	}
}

// generateTimingTests creates tests to find minimum safe wait time.
func generateTimingTests(presets []string) []TestCase {
	if len(presets) < 2 {
		return nil
	}

	low := presets[0]
	mid := presets[len(presets)/2]
	high := presets[len(presets)-1]

	waitTimes := []time.Duration{
		0,
		5 * time.Second,
		10 * time.Second,
		15 * time.Second,
		20 * time.Second,
		30 * time.Second,
		45 * time.Second,
		60 * time.Second,
		90 * time.Second,
		120 * time.Second,
	}

	var tests []TestCase
	for _, wait := range waitTimes {
		waitStr := fmt.Sprintf("%ds", int(wait.Seconds()))
		if wait == 0 {
			waitStr = "immediate"
		}

		tests = append(tests, TestCase{
			Name:        fmt.Sprintf("timing_%s", waitStr),
			Description: fmt.Sprintf("Change %s->%s, wait %s, change %s->%s", low, mid, waitStr, mid, high),
			Phase:       "timing",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionWait, WaitTime: wait},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionCheckRestart},
			},
		})
	}

	return tests
}

// generateDirectionTests creates tests for different change directions.
func generateDirectionTests(presets []string) []TestCase {
	if len(presets) < 3 {
		return nil
	}

	low := presets[0]
	mid := presets[len(presets)/2]
	high := presets[len(presets)-1]

	return []TestCase{
		{
			Name:        "direction_up_up",
			Description: fmt.Sprintf("Up then up: %s->%s->%s (immediate)", low, mid, high),
			Phase:       "direction",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionWait, WaitTime: 0},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "direction_down_down",
			Description: fmt.Sprintf("Down then down: %s->%s->%s (immediate)", high, mid, low),
			Phase:       "direction",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionWait, WaitTime: 0},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "direction_up_down",
			Description: fmt.Sprintf("Up then down: %s->%s->%s (immediate)", low, high, mid),
			Phase:       "direction",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionWait, WaitTime: 0},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "direction_down_up",
			Description: fmt.Sprintf("Down then up: %s->%s->%s (immediate)", high, low, mid),
			Phase:       "direction",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWait, WaitTime: 0},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionCheckRestart},
			},
		},
	}
}

// generateMagnitudeTests creates tests for different jump sizes.
func generateMagnitudeTests(presets []string) []TestCase {
	if len(presets) < 4 {
		return nil
	}

	var tests []TestCase

	// Small jump (1 step)
	tests = append(tests, TestCase{
		Name:        "magnitude_small",
		Description: fmt.Sprintf("Small jump: %s->%s->%s (1 step each, immediate)", presets[0], presets[1], presets[2]),
		Phase:       "magnitude",
		Steps: []TestStep{
			{Action: ActionWaitStable, WaitStable: true},
			{Action: ActionSetPreset, Preset: presets[0]},
			{Action: ActionWaitStable, WaitStable: true},
			{Action: ActionSetPreset, Preset: presets[1]},
			{Action: ActionWait, WaitTime: 0},
			{Action: ActionSetPreset, Preset: presets[2]},
			{Action: ActionCheckRestart},
		},
	})

	// Large jump (min to max)
	tests = append(tests, TestCase{
		Name:        "magnitude_large",
		Description: fmt.Sprintf("Large jump: %s->%s->%s (min to max, immediate)", presets[0], presets[len(presets)-1], presets[0]),
		Phase:       "magnitude",
		Steps: []TestStep{
			{Action: ActionWaitStable, WaitStable: true},
			{Action: ActionSetPreset, Preset: presets[0]},
			{Action: ActionWaitStable, WaitStable: true},
			{Action: ActionSetPreset, Preset: presets[len(presets)-1]},
			{Action: ActionWait, WaitTime: 0},
			{Action: ActionSetPreset, Preset: presets[0]},
			{Action: ActionCheckRestart},
		},
	})

	return tests
}

// generateEdgeCaseTests creates tests for edge cases.
func generateEdgeCaseTests(presets []string) []TestCase {
	if len(presets) < 2 {
		return nil
	}

	low := presets[0]
	mid := presets[len(presets)/2]
	high := presets[len(presets)-1]

	// Check if "disabled" is in the preset list (it will be last since it's highest power)
	hasDisabled := high == "disabled"

	// Get a numeric preset for disabled tests (use low since disabled is now at the end)
	firstNumeric := low
	// Get second highest for tests that need a non-disabled high preset
	secondHighest := mid
	if hasDisabled && len(presets) > 2 {
		secondHighest = presets[len(presets)-2]
	}

	tests := []TestCase{
		{
			Name:        "edge_same_preset",
			Description: fmt.Sprintf("Same preset: %s->%s (no actual change)", mid, mid),
			Phase:       "edge",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "edge_rapid_fire_3",
			Description: fmt.Sprintf("Rapid fire: %s->%s->%s->%s (3 changes, no wait)", low, mid, high, low),
			Phase:       "edge",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionCheckRestart},
			},
		},
		{
			Name:        "edge_rapid_fire_5",
			Description: "Rapid fire: 5 changes in quick succession",
			Phase:       "edge",
			Steps: []TestStep{
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionWaitStable, WaitStable: true},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionSetPreset, Preset: mid},
				{Action: ActionSetPreset, Preset: low},
				{Action: ActionSetPreset, Preset: high},
				{Action: ActionCheckRestart},
			},
		},
	}

	// Add disabled preset tests if "disabled" is in range
	if hasDisabled {
		tests = append(tests,
			TestCase{
				Name:        "edge_disabled_to_preset",
				Description: fmt.Sprintf("Stock to preset: disabled->%s (wait stable)", firstNumeric),
				Phase:       "edge",
				Steps: []TestStep{
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: "disabled"},
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: firstNumeric},
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionCheckRestart},
				},
			},
			TestCase{
				Name:        "edge_preset_to_disabled",
				Description: fmt.Sprintf("Preset to stock: %s->disabled (wait stable)", firstNumeric),
				Phase:       "edge",
				Steps: []TestStep{
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: firstNumeric},
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: "disabled"},
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionCheckRestart},
				},
			},
			TestCase{
				Name:        "edge_disabled_immediate",
				Description: fmt.Sprintf("Quick disabled toggle: disabled->%s->disabled (immediate)", firstNumeric),
				Phase:       "edge",
				Steps: []TestStep{
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: "disabled"},
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: firstNumeric},
					{Action: ActionWait, WaitTime: 0},
					{Action: ActionSetPreset, Preset: "disabled"},
					{Action: ActionCheckRestart},
				},
			},
			TestCase{
				Name:        "edge_disabled_to_high_immediate",
				Description: fmt.Sprintf("Stock to high autotune immediate: disabled->%s->%s", firstNumeric, secondHighest),
				Phase:       "edge",
				Steps: []TestStep{
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: "disabled"},
					{Action: ActionWaitStable, WaitStable: true},
					{Action: ActionSetPreset, Preset: firstNumeric},
					{Action: ActionWait, WaitTime: 0},
					{Action: ActionSetPreset, Preset: secondHighest},
					{Action: ActionCheckRestart},
				},
			},
		)
	}

	return tests
}
