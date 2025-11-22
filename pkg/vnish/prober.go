package vnish

import (
	"context"
	"errors"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

var (
	// ErrNotVNishFirmware indicates the host is not running VNish firmware.
	ErrNotVNishFirmware = errors.New("host is not running VNish firmware")
)

// Prober implements miner.FirmwareProber for VNish firmware.
type Prober struct {
	auth    *AuthManager
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

// NewProber creates a new VNish firmware prober.
func NewProber(auth *AuthManager, opts ...ProberOption) *Prober {
	p := &Prober{
		auth:    auth,
		timeout: 3 * time.Second,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Probe attempts to connect and identify VNish firmware.
// Implements miner.FirmwareProber.
// Note: Only calls GetInfo for fast detection. GetModel is skipped to reduce latency.
func (p *Prober) Probe(ctx context.Context, host string) (*miner.Info, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	client := NewClient(host, p.auth, WithTimeout(p.timeout))

	// Try to get info from VNish API (single request for fast detection)
	info, err := client.GetInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Verify it's VNish firmware
	if info.FWName != "Vnish" {
		return nil, ErrNotVNishFirmware
	}

	// Return basic info without fetching model (can be fetched later if needed)
	return &miner.Info{
		Miner:           info.Miner,
		Model:           info.Model,
		Firmware:        info.FWName,
		FirmwareVersion: info.FWVersion,
		Algorithm:       info.Algorithm,
		IP:              info.System.NetworkStatus.IP,
		MAC:             info.System.NetworkStatus.MAC,
		Hostname:        info.System.NetworkStatus.Hostname,
	}, nil
}

// FirmwareType returns the firmware type this prober detects.
// Implements miner.FirmwareProber.
func (p *Prober) FirmwareType() miner.FirmwareType {
	return miner.FirmwareVNish
}

// NewClient creates a client for hosts confirmed to run VNish firmware.
// Implements miner.FirmwareProber.
func (p *Prober) NewClient(host string) miner.Client {
	return NewClient(host, p.auth, WithTimeout(p.timeout))
}

// Ensure Prober implements miner.FirmwareProber.
var _ miner.FirmwareProber = (*Prober)(nil)
