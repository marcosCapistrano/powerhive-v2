// Package discovery provides network discovery for Bitcoin miners.
package discovery

import (
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// DiscoveredMiner contains information about a discovered miner.
type DiscoveredMiner struct {
	// IP is the miner's IP address.
	IP string

	// Hostname is the miner's hostname.
	Hostname string

	// MAC is the miner's MAC address.
	MAC string

	// Model is the miner model (e.g., "Antminer S19").
	Model string

	// Series is the product series (e.g., "x19").
	Series string

	// Firmware is the firmware name (e.g., "Vnish").
	Firmware string

	// FirmwareVersion is the firmware version (e.g., "1.2.6").
	FirmwareVersion string

	// Algorithm is the mining algorithm (e.g., "sha256d").
	Algorithm string

	// State is the miner's operational state (e.g., "running", "failure").
	State string

	// FirmwareType indicates the detected firmware type.
	FirmwareType miner.FirmwareType

	// DiscoveredAt is when the miner was discovered.
	DiscoveredAt time.Time
}

// ScanResult contains the results of a network scan.
type ScanResult struct {
	// Miners is the list of discovered miners.
	Miners []DiscoveredMiner

	// Errors contains errors encountered during scanning, keyed by IP.
	Errors map[string]error

	// Duration is how long the scan took.
	Duration time.Duration

	// ScannedIPs is the number of IPs that were scanned.
	ScannedIPs int

	// ResponsiveHosts is the number of hosts that responded on the target port.
	ResponsiveHosts int
}

// ScanOptions configures network scanning behavior.
type ScanOptions struct {
	// Timeout is the timeout for each host scan (default: 3s).
	Timeout time.Duration

	// Concurrency is the maximum number of concurrent scans (default: 50).
	Concurrency int

	// Port is the HTTP port to scan (default: 80).
	Port int

	// SkipDetection only checks port connectivity, doesn't try to detect miner type.
	SkipDetection bool
}

// DefaultScanOptions returns the default scan options.
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		Timeout:       3 * time.Second,
		Concurrency:   50,
		Port:          80,
		SkipDetection: false,
	}
}
