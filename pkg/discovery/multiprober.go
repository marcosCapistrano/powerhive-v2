package discovery

import (
	"context"
	"errors"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// ProbeResult contains the result of a firmware probe.
type ProbeResult struct {
	Info         *miner.Info
	FirmwareType miner.FirmwareType
	Client       miner.Client
}

// MultiProber tries multiple firmware probers to detect the firmware type.
type MultiProber struct {
	probers []miner.FirmwareProber
	timeout time.Duration
}

// MultiProberOption configures a MultiProber.
type MultiProberOption func(*MultiProber)

// WithMultiProberTimeout sets the timeout for each probe attempt.
func WithMultiProberTimeout(timeout time.Duration) MultiProberOption {
	return func(mp *MultiProber) {
		mp.timeout = timeout
	}
}

// NewMultiProber creates a new multi-prober with the given firmware probers.
// Probers are tried in order - typically VNish first (more features), then Stock.
func NewMultiProber(probers []miner.FirmwareProber, opts ...MultiProberOption) *MultiProber {
	mp := &MultiProber{
		probers: probers,
		timeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(mp)
	}

	return mp
}

// Probe attempts to detect the firmware type on a host.
// It tries each prober in order and returns the first successful result.
func (mp *MultiProber) Probe(ctx context.Context, host string) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, mp.timeout)
	defer cancel()

	var lastErr error

	for _, prober := range mp.probers {
		info, err := prober.Probe(ctx, host)
		if err == nil && info != nil {
			return &ProbeResult{
				Info:         info,
				FirmwareType: prober.FirmwareType(),
				Client:       prober.NewClient(host),
			}, nil
		}
		lastErr = err

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("no firmware detected")
}

// ProbeAll attempts all probers and returns all successful results.
// Useful for hosts that might respond to multiple firmware APIs.
func (mp *MultiProber) ProbeAll(ctx context.Context, host string) ([]ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, mp.timeout)
	defer cancel()

	var results []ProbeResult
	var lastErr error

	for _, prober := range mp.probers {
		info, err := prober.Probe(ctx, host)
		if err == nil && info != nil {
			results = append(results, ProbeResult{
				Info:         info,
				FirmwareType: prober.FirmwareType(),
				Client:       prober.NewClient(host),
			})
		} else {
			lastErr = err
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			if len(results) > 0 {
				return results, nil
			}
			return nil, ctx.Err()
		default:
		}
	}

	if len(results) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return results, nil
}

// DetectFirmwareType detects just the firmware type without returning a client.
func (mp *MultiProber) DetectFirmwareType(ctx context.Context, host string) (miner.FirmwareType, error) {
	result, err := mp.Probe(ctx, host)
	if err != nil {
		return miner.FirmwareUnknown, err
	}
	return result.FirmwareType, nil
}
