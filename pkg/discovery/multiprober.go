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
// Probers are tried concurrently - first successful result wins.
func NewMultiProber(probers []miner.FirmwareProber, opts ...MultiProberOption) *MultiProber {
	mp := &MultiProber{
		probers: probers,
		timeout: 3 * time.Second,
	}

	for _, opt := range opts {
		opt(mp)
	}

	return mp
}

// Probe attempts to detect the firmware type on a host.
// It tries all probers concurrently and returns the first successful result.
func (mp *MultiProber) Probe(ctx context.Context, host string) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, mp.timeout)
	defer cancel()

	// Channel for results - buffered to avoid goroutine leaks
	type probeAttempt struct {
		result *ProbeResult
		err    error
	}
	resultCh := make(chan probeAttempt, len(mp.probers))

	// Launch all probers concurrently
	for _, prober := range mp.probers {
		go func(p miner.FirmwareProber) {
			info, err := p.Probe(ctx, host)
			if err == nil && info != nil {
				resultCh <- probeAttempt{
					result: &ProbeResult{
						Info:         info,
						FirmwareType: p.FirmwareType(),
						Client:       p.NewClient(host),
					},
				}
			} else {
				resultCh <- probeAttempt{err: err}
			}
		}(prober)
	}

	// Wait for first success or all failures
	var lastErr error
	for i := 0; i < len(mp.probers); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case attempt := <-resultCh:
			if attempt.result != nil {
				return attempt.result, nil
			}
			lastErr = attempt.err
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
