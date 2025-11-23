// power-balancer is a CLI tool for dynamically adjusting miner power consumption
// based on farm generation/consumption data.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

const usage = `power-balancer - Dynamic miner power management

Usage:
  power-balancer [command]

Commands:
  start                Start the power balancer daemon and dashboard
  discover <network>   Discover miners on a network (one-time scan)
  status               Show current system status
  help                 Show this help message

Environment Variables (or set in .env file):
  POWER_BALANCER_DB       SQLite database path (default: power-balancer.db)
  AGGREGATOR_URL          Energy aggregator API URL
  AGGREGATOR_API_KEY      API key for energy aggregator
  NETWORK_CIDR            Network to scan for miners (e.g., 192.168.1.0/24)
  DISCOVERY_INTERVAL      How often to scan for new miners (default: 5m)
  VNISH_PASSWORD          VNish firmware password (default: admin)
  EMERGENCY_MARGIN        Emergency threshold percent (default: 5)
  CRITICAL_MARGIN         Critical threshold percent (default: 10)
  SAFE_MARGIN             Target margin percent (default: 15)
  RECOVERY_MARGIN         Recovery threshold percent (default: 20)
  POLL_INTERVAL           Energy polling interval (default: 5s)
  CHANGE_SPACING          Time between preset changes (default: 10s)
  COOLDOWN_DURATION       Per-miner cooldown (default: 10m)
  SETTLE_TIME             Time for changes to take effect (default: 5m)
  DASHBOARD_PORT          Dashboard HTTP port (default: 8081)
`

func main() {
	// Default to "start" if no command given
	cmd := "start"
	if len(os.Args) >= 2 {
		cmd = os.Args[1]
	}

	// Load configuration
	cfg := LoadConfig()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal...")
		cancel()
	}()

	// Dispatch command
	switch cmd {
	case "start":
		runStart(ctx, cfg)
	case "discover":
		runDiscover(ctx, cfg)
	case "status":
		runStatus(ctx, cfg)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func runStart(ctx context.Context, cfg *Config) {
	// Validate required config
	if cfg.AggregatorAPIKey == "" {
		log.Fatal("AGGREGATOR_API_KEY is required")
	}
	if cfg.NetworkCIDR == "" {
		log.Fatal("NETWORK_CIDR is required")
	}

	// Initialize database
	repo, err := NewRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer repo.Close()

	// Create VNish prober
	vnishAuth := vnish.NewAuthManager(cfg.VNishPassword)
	probers := []miner.FirmwareProber{
		vnish.NewProber(vnishAuth, vnish.WithProberTimeout(cfg.ChangeSpacing)),
	}

	// Create aggregator client
	aggregator := NewAggregator(cfg.AggregatorURL, cfg.AggregatorAPIKey)

	// Create controller
	controller := NewController(vnishAuth, cfg)

	// Create strategy
	strategy := NewStrategy(repo, cfg)

	// Create balancer
	balancer := NewBalancer(repo, aggregator, controller, strategy, probers, cfg)

	// Create HTTP server
	server := NewServer(repo, balancer, cfg)

	log.Printf("Power Balancer starting...")
	log.Printf("Database: %s", cfg.DBPath)
	log.Printf("Network: %s", cfg.NetworkCIDR)
	log.Printf("Dashboard: http://localhost:%d", cfg.DashboardPort)
	log.Printf("Thresholds: Emergency=%.0f%%, Critical=%.0f%%, Safe=%.0f%%, Recovery=%.0f%%",
		cfg.EmergencyMargin, cfg.CriticalMargin, cfg.SafeMargin, cfg.RecoveryMargin)

	// Start components
	errCh := make(chan error, 2)

	// Start balancer daemon
	go func() {
		errCh <- balancer.Run(ctx)
	}()

	// Start HTTP server
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Wait for first error or context cancellation
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			log.Fatalf("Error: %v", err)
		}
	case <-ctx.Done():
		log.Println("Shutting down...")
	}
}

func runDiscover(ctx context.Context, cfg *Config) {
	var network string
	if len(os.Args) >= 3 {
		network = os.Args[2]
	} else if cfg.NetworkCIDR != "" {
		network = cfg.NetworkCIDR
	} else {
		fmt.Fprintln(os.Stderr, "Error: network CIDR required")
		fmt.Fprintln(os.Stderr, "Usage: power-balancer discover <network>")
		os.Exit(1)
	}

	// Initialize database
	repo, err := NewRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer repo.Close()

	// Create VNish prober
	vnishAuth := vnish.NewAuthManager(cfg.VNishPassword)
	probers := []miner.FirmwareProber{
		vnish.NewProber(vnishAuth, vnish.WithProberTimeout(cfg.ChangeSpacing)),
	}

	// Create balancer just for discovery
	balancer := NewBalancer(repo, nil, nil, nil, probers, cfg)

	log.Printf("Discovering miners on %s...", network)
	count, err := balancer.DiscoverMiners(ctx, network)
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}
	log.Printf("Discovered %d miners", count)
}

func runStatus(ctx context.Context, cfg *Config) {
	// Initialize database
	repo, err := NewRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer repo.Close()

	// Get latest energy reading
	reading, err := repo.GetLatestEnergyReading(ctx)
	if err != nil {
		log.Printf("No energy readings yet")
	} else {
		fmt.Printf("Latest Energy Reading (%s):\n", reading.Timestamp.Format("15:04:05"))
		fmt.Printf("  Generation:  %.2f MW\n", reading.GenerationMW)
		fmt.Printf("  Consumption: %.2f MW\n", reading.ConsumptionMW)
		fmt.Printf("  Margin:      %.2f MW (%.1f%%)\n", reading.MarginMW, reading.MarginPercent)
		fmt.Printf("  Generoso:    %.2f MW (%s)\n", reading.GenerosoMW, reading.GenerosoStatus)
		fmt.Printf("  Nogueira:    %.2f MW (%s)\n", reading.NogueiraMW, reading.NogueiraStatus)
	}

	// Get managed miners count
	count, _ := repo.CountManagedMiners(ctx)
	fmt.Printf("\nManaged Miners: %d\n", count)

	// Get cooldown count
	cooldownCount, _ := repo.CountMinersOnCooldown(ctx)
	fmt.Printf("On Cooldown: %d\n", cooldownCount)

	// Get pending changes
	pendingDelta, _ := repo.SumPendingDelta(ctx)
	fmt.Printf("Pending Delta: %d W\n", pendingDelta)

	// Get recent changes
	logs, _ := repo.GetRecentChangeLogs(ctx, 5)
	if len(logs) > 0 {
		fmt.Printf("\nRecent Changes:\n")
		for _, l := range logs {
			status := "OK"
			if !l.Success {
				status = "FAIL"
			}
			fmt.Printf("  [%s] %s: %s -> %s (%s) %s\n",
				l.IssuedAt.Format("15:04:05"), l.MinerIP, l.FromPreset, l.ToPreset, l.Reason, status)
		}
	}
}
