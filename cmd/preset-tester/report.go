package main

import (
	"fmt"
	"strings"
	"time"
)

// GenerateReport creates a summary report from test results.
func GenerateReport(results []TestResult) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(repeat("=", 60) + "\n")
	sb.WriteString("                    TEST RESULTS SUMMARY\n")
	sb.WriteString(repeat("=", 60) + "\n\n")

	// Group by phase
	phases := []string{"baseline", "timing", "direction", "magnitude", "edge"}
	phaseResults := make(map[string][]TestResult)

	for _, r := range results {
		phase := r.TestCase.Phase
		phaseResults[phase] = append(phaseResults[phase], r)
	}

	// Summary counts
	totalPassed := 0
	totalFailed := 0
	totalError := 0

	for _, phase := range phases {
		pResults := phaseResults[phase]
		if len(pResults) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("Phase: %s\n", strings.ToUpper(phase)))
		sb.WriteString(repeat("-", 40) + "\n")

		for _, r := range pResults {
			var status string
			var detail string

			if r.Error != nil {
				status = "ERROR"
				detail = r.Error.Error()
				totalError++
			} else if r.RebootTriggered {
				status = "REBOOT"
				detail = "reboot_required triggered"
				totalFailed++
			} else if r.RestartTriggered {
				status = "FAIL"
				detail = "restart_required triggered"
				totalFailed++
			} else {
				status = "PASS"
				detail = fmt.Sprintf("completed in %s", r.Duration().Round(time.Second))
				totalPassed++
			}

			icon := getStatusIcon(status)
			sb.WriteString(fmt.Sprintf("  %s %-25s %s\n", icon, r.TestCase.Name+":", status))
			if detail != "" && status != "PASS" {
				sb.WriteString(fmt.Sprintf("      %s\n", detail))
			}
		}
		sb.WriteString("\n")
	}

	// Overall summary
	sb.WriteString(repeat("=", 60) + "\n")
	sb.WriteString("SUMMARY\n")
	sb.WriteString(repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("  Total tests: %d\n", len(results)))
	sb.WriteString(fmt.Sprintf("  Passed:      %d\n", totalPassed))
	sb.WriteString(fmt.Sprintf("  Failed:      %d (restart triggered)\n", totalFailed))
	sb.WriteString(fmt.Sprintf("  Errors:      %d\n", totalError))
	sb.WriteString("\n")

	// Timing analysis
	sb.WriteString(analyzeTimingResults(phaseResults["timing"]))

	// Conclusions
	sb.WriteString(generateConclusions(results, phaseResults))

	return sb.String()
}

func getStatusIcon(status string) string {
	switch status {
	case "PASS":
		return "[OK]"
	case "FAIL":
		return "[X]"
	case "REBOOT":
		return "[!!]"
	case "ERROR":
		return "[?]"
	default:
		return "[ ]"
	}
}

// analyzeTimingResults finds the minimum safe wait time from timing tests.
func analyzeTimingResults(timingResults []TestResult) string {
	if len(timingResults) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("TIMING ANALYSIS\n")
	sb.WriteString(repeat("-", 40) + "\n")

	minSafeWait := time.Duration(-1)
	maxUnsafeWait := time.Duration(0)

	for _, r := range timingResults {
		// Extract wait time from test name (e.g., "timing_30s")
		name := r.TestCase.Name
		var waitTime time.Duration

		if strings.Contains(name, "immediate") || strings.HasSuffix(name, "_0s") {
			waitTime = 0
		} else {
			// Parse from name like "timing_30s"
			var seconds int
			if _, err := fmt.Sscanf(name, "timing_%ds", &seconds); err == nil {
				waitTime = time.Duration(seconds) * time.Second
			}
		}

		if r.Passed() {
			if minSafeWait < 0 || waitTime < minSafeWait {
				minSafeWait = waitTime
			}
		} else if r.RestartTriggered {
			if waitTime > maxUnsafeWait {
				maxUnsafeWait = waitTime
			}
		}
	}

	if minSafeWait >= 0 {
		sb.WriteString(fmt.Sprintf("  Minimum safe wait time: %s\n", minSafeWait))
	} else {
		sb.WriteString("  Minimum safe wait time: UNKNOWN (no passing timing tests)\n")
	}

	if maxUnsafeWait > 0 {
		sb.WriteString(fmt.Sprintf("  Maximum unsafe wait time: %s\n", maxUnsafeWait))
	}

	sb.WriteString("\n")
	return sb.String()
}

// generateConclusions provides actionable insights.
func generateConclusions(results []TestResult, phaseResults map[string][]TestResult) string {
	var sb strings.Builder
	sb.WriteString("CONCLUSIONS\n")
	sb.WriteString(repeat("-", 40) + "\n")

	// Check baseline
	baselinePassed := true
	for _, r := range phaseResults["baseline"] {
		if !r.Passed() {
			baselinePassed = false
			break
		}
	}

	if baselinePassed {
		sb.WriteString("  [OK] Baseline tests passed: waiting for stability is safe\n")
	} else {
		sb.WriteString("  [!] Baseline tests failed: even stable changes trigger restart\n")
	}

	// Check if any immediate changes worked
	anyImmediatePass := false
	for _, r := range results {
		for _, step := range r.TestCase.Steps {
			if step.Action == ActionWait && step.WaitTime == 0 {
				if r.Passed() {
					anyImmediatePass = true
					break
				}
			}
		}
	}

	if anyImmediatePass {
		sb.WriteString("  [OK] Some immediate changes succeeded\n")
	} else {
		sb.WriteString("  [!] All immediate changes failed: waiting is required\n")
	}

	// Direction analysis
	directionMatters := false
	dirResults := phaseResults["direction"]
	passedDirs := make(map[string]bool)
	for _, r := range dirResults {
		if r.Passed() {
			passedDirs[r.TestCase.Name] = true
		}
	}
	if len(passedDirs) > 0 && len(passedDirs) < len(dirResults) {
		directionMatters = true
	}

	if directionMatters {
		sb.WriteString("  [!] Direction matters: some directions succeed, others fail\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// repeat is a helper for string repetition
func repeat(s string, n int) string {
	return strings.Repeat(s, n)
}
