package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

func main() {
	host := flag.String("host", "", "Miner IP address (required)")
	password := flag.String("password", "", "VNish password (default: VNISH_PASSWORD env or 'admin')")
	minPreset := flag.String("min-preset", "disabled", "Minimum preset for testing (use 'disabled' for stock)")
	maxPreset := flag.String("max-preset", "3000", "Maximum preset for testing")
	pollInterval := flag.Duration("poll", 1*time.Second, "Polling interval for stability checks")
	timeout := flag.Duration("timeout", 5*time.Minute, "Timeout for waiting for stability")
	flag.Parse()

	if *host == "" {
		fmt.Fprintf(os.Stderr, "Error: --host is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: preset-tester --host <miner-ip> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Get password from flag, env, or default
	pass := *password
	if pass == "" {
		pass = os.Getenv("VNISH_PASSWORD")
	}
	if pass == "" {
		pass = "admin"
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping tests...")
		cancel()
	}()

	// Create VNish client
	auth := vnish.NewAuthManager(pass)
	client := vnish.NewClient(*host, auth)

	// Print banner
	fmt.Println("============================================================")
	fmt.Println("              VNISH PRESET CHANGE TESTER")
	fmt.Println("============================================================")
	fmt.Printf("Host:        %s\n", *host)
	fmt.Printf("Min Preset:  %s\n", *minPreset)
	fmt.Printf("Max Preset:  %s\n", *maxPreset)
	fmt.Printf("Poll:        %s\n", *pollInterval)
	fmt.Printf("Timeout:     %s\n", *timeout)
	fmt.Println("============================================================")

	// Authenticate
	fmt.Println("\nConnecting and authenticating...")
	if err := client.EnsureAuthenticated(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to authenticate: %v\n", err)
		os.Exit(1)
	}
	if err := client.EnsureAPIKey(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to ensure API key: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Authenticated successfully.")

	// Get available presets
	fmt.Println("\nFetching available presets...")
	presets, err := client.GetAutotunePresets(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get presets: %v\n", err)
		os.Exit(1)
	}

	// Extract preset names
	presetNames := make([]string, 0, len(presets))
	for _, p := range presets {
		presetNames = append(presetNames, p.Name)
	}
	fmt.Printf("Available presets: %v\n", presetNames)

	// Generate test suite
	fmt.Println("\nGenerating test suite...")
	suite := GenerateTestSuite(presetNames, *minPreset, *maxPreset)
	if len(suite) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no test cases generated (check min/max preset range)\n")
		os.Exit(1)
	}
	fmt.Printf("Generated %d test cases\n", len(suite))

	// Show test summary by phase
	phaseCount := make(map[string]int)
	for _, tc := range suite {
		phaseCount[tc.Phase]++
	}
	fmt.Println("\nTest breakdown:")
	for phase, count := range phaseCount {
		fmt.Printf("  - %s: %d tests\n", phase, count)
	}

	// Get current miner status
	fmt.Println("\nChecking current miner status...")
	status, err := client.GetStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get status: %v\n", err)
		os.Exit(1)
	}
	perf, err := client.GetPerfSummary(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get perf summary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current state: %s\n", status.MinerState)
	fmt.Printf("Current preset: %s (%s)\n", perf.CurrentPreset.Name, perf.CurrentPreset.Status)
	fmt.Printf("Restart required: %v\n", status.RestartRequired)
	fmt.Printf("Reboot required: %v\n", status.RebootRequired)

	// Warn if restart/reboot already required
	if status.RestartRequired || status.RebootRequired {
		fmt.Println("\nWARNING: Miner already has restart/reboot required flag set!")
		fmt.Println("Consider restarting the miner before running tests.")
	}

	// Confirm start
	fmt.Println("\n============================================================")
	fmt.Println("Ready to start automated testing.")
	fmt.Println("This will change presets on the miner multiple times.")
	fmt.Println("Press Ctrl+C at any time to abort.")
	fmt.Println("============================================================")
	fmt.Println("\nStarting in 5 seconds...")
	select {
	case <-ctx.Done():
		fmt.Println("Aborted.")
		os.Exit(0)
	case <-time.After(5 * time.Second):
	}

	// Run tests
	fmt.Println("\n============================================================")
	fmt.Println("                    RUNNING TESTS")
	fmt.Println("============================================================")

	runner := NewRunner(client, *pollInterval, *timeout)
	results := runner.Run(ctx, suite)

	// Generate and print report
	report := GenerateReport(results)
	fmt.Print(report)

	// Exit with appropriate code
	hasFailures := false
	for _, r := range results {
		if !r.Passed() {
			hasFailures = true
			break
		}
	}
	if hasFailures {
		os.Exit(1)
	}
}
