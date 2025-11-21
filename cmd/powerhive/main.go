package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/discovery"
	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/stock"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "scan":
		if len(os.Args) < 3 {
			fmt.Println("Usage: powerhive scan <cidr>")
			fmt.Println("Example: powerhive scan 192.168.1.0/24")
			os.Exit(1)
		}
		runScan(os.Args[2])

	case "info":
		if len(os.Args) < 3 {
			fmt.Println("Usage: powerhive info <miner-ip>")
			fmt.Println("Example: powerhive info 192.168.1.2")
			os.Exit(1)
		}
		runInfo(os.Args[2])

	case "detect":
		if len(os.Args) < 3 {
			fmt.Println("Usage: powerhive detect <miner-ip>")
			fmt.Println("Example: powerhive detect 192.168.1.2")
			os.Exit(1)
		}
		runDetect(os.Args[2])

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("PowerHive - Bitcoin Miner Management")
	fmt.Println()
	fmt.Println("Usage: powerhive <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan <cidr>      Scan network for miners (e.g., 192.168.1.0/24)")
	fmt.Println("  detect <ip>      Detect miner type at IP address")
	fmt.Println("  info <ip>        Get detailed miner information")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  VNISH_PASSWORD   VNish firmware password (default: admin)")
	fmt.Println("  STOCK_USERNAME   Stock firmware username (default: root)")
	fmt.Println("  STOCK_PASSWORD   Stock firmware password (default: root)")
}

// createProbers creates firmware probers for discovery.
// Order matters: VNish first (more features), then Stock as fallback.
func createProbers() []miner.FirmwareProber {
	// VNish prober
	vnishAuth := vnish.NewAuthManager(getVNishPassword())
	vnishProber := vnish.NewProber(vnishAuth, vnish.WithProberTimeout(5*time.Second))

	// Stock prober
	stockAuth := stock.NewDigestAuthWithCredentials(getStockUsername(), getStockPassword())
	stockProber := stock.NewProber(stockAuth, stock.WithProberTimeout(5*time.Second))

	return []miner.FirmwareProber{vnishProber, stockProber}
}

func runScan(cidr string) {
	ctx := context.Background()
	probers := createProbers()

	// Create scanner with probers
	scanner := discovery.NewScanner(probers,
		discovery.WithTimeout(3*time.Second),
		discovery.WithConcurrency(50),
	)

	fmt.Printf("Scanning %s for miners (VNish + Stock firmware)...\n", cidr)
	startTime := time.Now()

	result, err := scanner.ScanNetwork(ctx, cidr)
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	fmt.Printf("\nScan completed in %v\n", result.Duration)
	fmt.Printf("Scanned IPs: %d, Responsive: %d, Miners found: %d\n",
		result.ScannedIPs, result.ResponsiveHosts, len(result.Miners))

	if len(result.Miners) > 0 {
		fmt.Println("\nDiscovered Miners:")
		fmt.Println("------------------")
		for _, m := range result.Miners {
			fwInfo := m.Firmware
			if m.FirmwareVersion != "" {
				fwInfo += " " + m.FirmwareVersion
			}
			state := m.State
			if state == "" {
				state = "unknown"
			}
			fmt.Printf("  %-15s - %-20s (%s) [%s]\n",
				m.IP, m.Model, fwInfo, state)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nHosts with errors: %d (not miners or connection failed)\n", len(result.Errors))
	}

	fmt.Printf("\nTotal time: %v\n", time.Since(startTime))
}

func runDetect(ip string) {
	ctx := context.Background()
	probers := createProbers()

	detector := discovery.NewDetector(probers, discovery.WithDetectorTimeout(5*time.Second))

	fmt.Printf("Detecting miner at %s...\n", ip)

	discovered, err := detector.DetectMiner(ctx, ip)
	if err != nil {
		log.Fatalf("Detection failed: %v", err)
	}

	fmt.Printf("\nMiner detected:\n")
	fmt.Printf("  IP:           %s\n", discovered.IP)
	fmt.Printf("  Hostname:     %s\n", discovered.Hostname)
	fmt.Printf("  MAC:          %s\n", discovered.MAC)
	fmt.Printf("  Model:        %s\n", discovered.Model)
	fmt.Printf("  Series:       %s\n", discovered.Series)
	fmt.Printf("  Firmware:     %s %s\n", discovered.Firmware, discovered.FirmwareVersion)
	fmt.Printf("  Firmware Type: %s\n", discovered.FirmwareType)
	fmt.Printf("  State:        %s\n", discovered.State)
}

func runInfo(ip string) {
	ctx := context.Background()
	probers := createProbers()

	detector := discovery.NewDetector(probers, discovery.WithDetectorTimeout(5*time.Second))

	fmt.Printf("Connecting to miner at %s...\n", ip)

	// First detect which firmware
	client, fwType, err := detector.GetClient(ctx, ip)
	if err != nil {
		log.Fatalf("Failed to detect miner: %v", err)
	}

	fmt.Printf("Detected firmware: %s\n\n", fwType)

	// Get basic info
	info, err := client.GetMinerInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get miner info: %v", err)
	}

	fmt.Printf("Miner Information:\n")
	fmt.Printf("  Miner:    %s\n", info.Miner)
	fmt.Printf("  Model:    %s\n", info.Model)
	fmt.Printf("  Series:   %s\n", info.Series)
	fmt.Printf("  Firmware: %s %s\n", info.Firmware, info.FirmwareVersion)
	fmt.Printf("  IP:       %s\n", info.IP)
	fmt.Printf("  MAC:      %s\n", info.MAC)
	fmt.Printf("  Hostname: %s\n", info.Hostname)

	// Get status
	status, err := client.GetMinerStatus(ctx)
	if err != nil {
		log.Printf("Failed to get status: %v", err)
	} else {
		fmt.Printf("\nStatus:\n")
		fmt.Printf("  State:       %s\n", status.State)
		fmt.Printf("  Description: %s\n", status.Description)
	}

	// If VNish, try to get more detailed info
	if fwType == miner.FirmwareVNish {
		runVNishDetails(ctx, ip)
	}
}

func runVNishDetails(ctx context.Context, ip string) {
	vnishAuth := vnish.NewAuthManager(getVNishPassword())
	client := vnish.NewClient(ip, vnishAuth, vnish.WithTimeout(30*time.Second))

	// Get summary (requires auth)
	fmt.Println("\nFetching VNish summary (authenticating)...")
	summary, err := client.GetSummary(ctx)
	if err != nil {
		log.Printf("Failed to get VNish summary: %v", err)
		return
	}

	fmt.Printf("\nMining Summary:\n")
	fmt.Printf("  Hashrate:   %.2f GH/s\n", summary.Miner.InstantHashrate)
	fmt.Printf("  Power:      %d W\n", summary.Miner.PowerConsumption)
	fmt.Printf("  Efficiency: %.2f J/TH\n", summary.Miner.PowerEfficiency)
	fmt.Printf("  Chip Temp:  %d-%d C\n", summary.Miner.ChipTemp.Min, summary.Miner.ChipTemp.Max)
	fmt.Printf("  PCB Temp:   %d-%d C\n", summary.Miner.PCBTemp.Min, summary.Miner.PCBTemp.Max)
	fmt.Printf("  HW Errors:  %.2f%%\n", summary.Miner.HWErrorsPercent)
}

func getVNishPassword() string {
	if pw := os.Getenv("VNISH_PASSWORD"); pw != "" {
		return pw
	}
	return "admin"
}

func getStockUsername() string {
	if u := os.Getenv("STOCK_USERNAME"); u != "" {
		return u
	}
	return "root"
}

func getStockPassword() string {
	if pw := os.Getenv("STOCK_PASSWORD"); pw != "" {
		return pw
	}
	return "root"
}
