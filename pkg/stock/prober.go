package stock

import (
	"context"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Prober implements miner.FirmwareProber for stock Bitmain firmware.
type Prober struct {
	auth    *DigestAuth
	timeout time.Duration
}

// ProberOption configures a Prober.
type ProberOption func(*Prober)

// WithProberTimeout sets the probe timeout.
func WithProberTimeout(timeout time.Duration) ProberOption {
	return func(p *Prober) {
		p.timeout = timeout
	}
}

// NewProber creates a new stock firmware prober.
func NewProber(auth *DigestAuth, opts ...ProberOption) *Prober {
	p := &Prober{
		auth:    auth,
		timeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Probe attempts to connect and identify stock firmware.
// Implements miner.FirmwareProber.
func (p *Prober) Probe(ctx context.Context, host string) (*miner.Info, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	client := NewClient(host, p.auth, WithTimeout(p.timeout))

	// Try to get system info
	sysInfo, err := client.GetSystemInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Verify it looks like a valid stock firmware response
	if sysInfo.MinerType == "" {
		return nil, ErrNotStockFirmware
	}

	// Use algorithm from API if available, fallback to sha256d for older models
	algorithm := sysInfo.Algorithm
	if algorithm == "" {
		algorithm = "sha256d"
	}

	return &miner.Info{
		Miner:           sysInfo.MinerType,
		Model:           sysInfo.MinerType,
		Series:          extractSeries(sysInfo.MinerType),
		Firmware:        "Stock",
		FirmwareVersion: sysInfo.SystemFilesystemVersion,
		Algorithm:       algorithm,
		IP:              sysInfo.IPAddress,
		MAC:             sysInfo.MACAddr,
		Hostname:        sysInfo.Hostname,
	}, nil
}

// FirmwareType returns the firmware type this prober detects.
// Implements miner.FirmwareProber.
func (p *Prober) FirmwareType() miner.FirmwareType {
	return miner.FirmwareStock
}

// NewClient creates a client for hosts confirmed to run stock firmware.
// Implements miner.FirmwareProber.
func (p *Prober) NewClient(host string) miner.Client {
	return NewClient(host, p.auth, WithTimeout(p.timeout))
}

// Ensure Prober implements miner.FirmwareProber.
var _ miner.FirmwareProber = (*Prober)(nil)
