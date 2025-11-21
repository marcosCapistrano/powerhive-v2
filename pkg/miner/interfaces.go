// Package miner provides shared interfaces and types for miner interaction.
// This package defines abstractions that decouple discovery from specific
// firmware implementations like VNish.
package miner

import "context"

// Client abstracts miner API operations.
// Implementations include vnish.HTTPClient for VNish firmware.
type Client interface {
	// Host returns the miner's host address (IP or hostname).
	Host() string

	// GetMinerInfo returns basic miner information.
	// This should be a public endpoint that doesn't require authentication.
	GetMinerInfo(ctx context.Context) (*Info, error)

	// GetMinerStatus returns the miner's operational status.
	GetMinerStatus(ctx context.Context) (*Status, error)
}

// ClientFactory creates miner clients for specific hosts.
// This is injected into the discovery package to decouple it from
// specific firmware implementations.
type ClientFactory interface {
	// NewClient creates a new miner client for the given host.
	NewClient(host string) Client
}

// FirmwareProber attempts to detect a specific firmware type on a host.
// Each firmware implementation (vnish, stock, braiins) provides its own prober.
type FirmwareProber interface {
	// Probe attempts to connect to the host and retrieve miner information.
	// Returns an error if this firmware type is not detected.
	Probe(ctx context.Context, host string) (*Info, error)

	// FirmwareType returns which firmware this prober detects.
	FirmwareType() FirmwareType

	// NewClient creates a client for hosts confirmed to run this firmware.
	NewClient(host string) Client
}
