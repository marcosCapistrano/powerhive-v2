package discovery

import (
	"context"
	"errors"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Common detection errors.
var (
	// ErrNotMiner indicates the host is not a recognized miner.
	ErrNotMiner = errors.New("host is not a recognized miner")

	// ErrConnectionFailed indicates connection to the host failed.
	ErrConnectionFailed = errors.New("connection failed")
)

// Detector detects miner type and retrieves miner information.
// It supports multiple firmware types through the MultiProber.
type Detector struct {
	multiProber *MultiProber
	timeout     time.Duration
}

// DetectorOption configures a Detector.
type DetectorOption func(*Detector)

// WithDetectorTimeout sets the detection timeout.
func WithDetectorTimeout(timeout time.Duration) DetectorOption {
	return func(d *Detector) {
		d.timeout = timeout
	}
}

// NewDetector creates a new miner detector with multiple firmware probers.
func NewDetector(probers []miner.FirmwareProber, opts ...DetectorOption) *Detector {
	d := &Detector{
		timeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(d)
	}

	d.multiProber = NewMultiProber(probers, WithMultiProberTimeout(d.timeout))

	return d
}

// NewDetectorWithMultiProber creates a detector with an existing MultiProber.
func NewDetectorWithMultiProber(mp *MultiProber, opts ...DetectorOption) *Detector {
	d := &Detector{
		multiProber: mp,
		timeout:     5 * time.Second,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// DetectMiner attempts to detect and identify a miner at the given IP.
// It tries multiple firmware types and returns the first successful detection.
func (d *Detector) DetectMiner(ctx context.Context, ip string) (*DiscoveredMiner, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	result, err := d.multiProber.Probe(ctx, ip)
	if err != nil {
		return nil, err
	}

	// Get status if possible
	state := ""
	if result.Client != nil {
		status, err := result.Client.GetMinerStatus(ctx)
		if err == nil && status != nil {
			state = status.State
		}
	}

	return &DiscoveredMiner{
		IP:              ip,
		Hostname:        result.Info.Hostname,
		MAC:             result.Info.MAC,
		Model:           result.Info.Miner,
		Series:          result.Info.Series,
		Firmware:        result.Info.Firmware,
		FirmwareVersion: result.Info.FirmwareVersion,
		Algorithm:       result.Info.Algorithm,
		State:           state,
		FirmwareType:    result.FirmwareType,
		DiscoveredAt:    time.Now(),
	}, nil
}

// DetectFirmwareType attempts to identify what firmware a host is running.
func (d *Detector) DetectFirmwareType(ctx context.Context, ip string) (miner.FirmwareType, error) {
	return d.multiProber.DetectFirmwareType(ctx, ip)
}

// GetClient returns a client for the detected firmware type.
func (d *Detector) GetClient(ctx context.Context, ip string) (miner.Client, miner.FirmwareType, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	result, err := d.multiProber.Probe(ctx, ip)
	if err != nil {
		return nil, miner.FirmwareUnknown, err
	}

	return result.Client, result.FirmwareType, nil
}
