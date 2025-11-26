// data-harvest is a CLI tool for discovering miners and collecting their data
// into an SQLite database.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/database"
	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/stock"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

const usage = `data-harvest - Miner data collection tool

Usage:
  data-harvest <command> [arguments]

Commands:
  scan <network>       Scan a network (CIDR) and harvest all discovered miners
                       Example: data-harvest scan 192.168.1.0/24

  harvest <ip> [ip...] Harvest data from specific miners
                       Example: data-harvest harvest 192.168.1.27 192.168.1.28

  daemon [networks...] Run continuous harvesting (use Ctrl+C to stop)
                       Example: data-harvest daemon 192.168.1.0/24

  list                 List all known miners in the database

  show <ip>            Show detailed information for a specific miner
                       Example: data-harvest show 192.168.1.27

  debug-api <ip>       Fetch and print raw API response (for debugging)
                       Example: data-harvest debug-api 192.168.1.21

Environment Variables:
  POWERHIVE_DB         SQLite database path (default: powerhive.db)
  VNISH_PASSWORD       VNish firmware password (default: admin)
  STOCK_USERNAME       Stock firmware username (default: root)
  STOCK_PASSWORD       Stock firmware password (default: root)
  HARVEST_INTERVAL     Daemon polling interval (default: 60s)
  HARVEST_CONCURRENCY  Parallel harvest workers (default: 10)
  HARVEST_TIMEOUT      Per-miner timeout (default: 10s)
  NETWORK_CIDR         Comma-separated CIDRs for daemon mode (e.g., 10.40.36.0/24,10.40.37.0/24)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	// Load configuration
	cfg := LoadConfig()

	// Initialize database
	repo, err := database.NewSQLiteRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer repo.Close()

	// Create probers
	probers := createProbers(cfg)

	// Create harvester
	harvester := NewHarvester(repo, probers, cfg)

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
	cmd := os.Args[1]
	switch cmd {
	case "scan":
		runScan(ctx, harvester, cfg)
	case "harvest":
		runHarvest(ctx, harvester)
	case "daemon":
		runDaemon(ctx, harvester, cfg)
	case "list":
		runList(ctx, harvester)
	case "show":
		runShow(ctx, harvester)
	case "debug-api":
		runDebugAPI(ctx, cfg)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func createProbers(cfg *Config) []miner.FirmwareProber {
	// VNish prober
	vnishAuth := vnish.NewAuthManager(cfg.VNishPassword)
	vnishProber := vnish.NewProber(vnishAuth, vnish.WithProberTimeout(cfg.Timeout))

	// Stock prober
	stockAuth := stock.NewDigestAuth()
	stockAuth.Username = cfg.StockUsername
	stockAuth.Password = cfg.StockPassword
	stockProber := stock.NewProber(stockAuth, stock.WithProberTimeout(cfg.Timeout))

	return []miner.FirmwareProber{vnishProber, stockProber}
}

func runScan(ctx context.Context, h *Harvester, cfg *Config) {
	var network string
	if len(os.Args) >= 3 {
		network = os.Args[2]
	} else if len(cfg.NetworkCIDRs) > 0 {
		network = cfg.NetworkCIDRs[0] // Use first network from config
	} else {
		fmt.Fprintln(os.Stderr, "Error: network CIDR required")
		fmt.Fprintln(os.Stderr, "Usage: data-harvest scan <network>")
		fmt.Fprintln(os.Stderr, "Example: data-harvest scan 192.168.1.0/24")
		os.Exit(1)
	}

	start := time.Now()
	if _, err := h.HarvestNetwork(ctx, network); err != nil {
		log.Fatalf("Scan failed: %v", err)
	}
	log.Printf("Scan completed in %s", time.Since(start).Round(time.Millisecond))
}

func runHarvest(ctx context.Context, h *Harvester) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: at least one IP address required")
		fmt.Fprintln(os.Stderr, "Usage: data-harvest harvest <ip> [ip...]")
		os.Exit(1)
	}

	ips := os.Args[2:]
	start := time.Now()
	if _, err := h.HarvestMiners(ctx, ips); err != nil {
		log.Fatalf("Harvest failed: %v", err)
	}
	log.Printf("Harvest completed in %s", time.Since(start).Round(time.Millisecond))
}

func runDaemon(ctx context.Context, h *Harvester, cfg *Config) {
	var networks []string

	if len(os.Args) >= 3 {
		networks = os.Args[2:]
	} else if len(cfg.NetworkCIDRs) > 0 {
		networks = cfg.NetworkCIDRs
	} else {
		// No networks specified, will only harvest known miners
		log.Println("Warning: No networks specified, will only harvest known miners from database")
	}

	log.Printf("Database: %s", cfg.DBPath)
	log.Printf("Networks: %v", networks)
	log.Printf("Interval: %s", cfg.HarvestInterval)
	log.Printf("Concurrency: %d", cfg.Concurrency)

	if err := h.RunDaemon(ctx, networks); err != nil && err != context.Canceled {
		log.Fatalf("Daemon error: %v", err)
	}
}

func runList(ctx context.Context, h *Harvester) {
	if err := h.ListMiners(ctx); err != nil {
		log.Fatalf("List failed: %v", err)
	}
}

func runShow(ctx context.Context, h *Harvester) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: IP address required")
		fmt.Fprintln(os.Stderr, "Usage: data-harvest show <ip>")
		os.Exit(1)
	}

	ip := os.Args[2]
	if err := h.ShowMiner(ctx, ip); err != nil {
		log.Fatalf("Show failed: %v", err)
	}
}

func runDebugAPI(ctx context.Context, cfg *Config) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: IP address required")
		fmt.Fprintln(os.Stderr, "Usage: data-harvest debug-api <ip>")
		os.Exit(1)
	}

	ip := os.Args[2]

	// Create VNish client
	auth := vnish.NewAuthManager(cfg.VNishPassword)
	client := vnish.NewClient(ip, auth, vnish.WithTimeout(cfg.Timeout))

	// Fetch raw summary
	fmt.Printf("Fetching raw /api/v1/summary from %s...\n\n", ip)
	raw, err := client.GetSummaryRaw(ctx)
	if err != nil {
		log.Fatalf("Failed to get summary: %v", err)
	}

	fmt.Println(string(raw))
}
