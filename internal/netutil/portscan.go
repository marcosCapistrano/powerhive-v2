package netutil

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// PortScanner scans TCP ports on network hosts.
type PortScanner struct {
	timeout     time.Duration
	concurrency int
}

// PortScannerOption configures a PortScanner.
type PortScannerOption func(*PortScanner)

// WithScanTimeout sets the timeout for each port scan attempt.
func WithScanTimeout(timeout time.Duration) PortScannerOption {
	return func(ps *PortScanner) {
		ps.timeout = timeout
	}
}

// WithScanConcurrency sets the maximum number of concurrent scans.
func WithScanConcurrency(concurrency int) PortScannerOption {
	return func(ps *PortScanner) {
		ps.concurrency = concurrency
	}
}

// NewPortScanner creates a new port scanner.
func NewPortScanner(opts ...PortScannerOption) *PortScanner {
	ps := &PortScanner{
		timeout:     2 * time.Second,
		concurrency: 100,
	}

	for _, opt := range opts {
		opt(ps)
	}

	return ps
}

// IsPortOpen checks if a TCP port is open on the given host.
func (ps *PortScanner) IsPortOpen(ctx context.Context, host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)

	dialer := &net.Dialer{
		Timeout: ps.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ScanResult contains the result of scanning a single host.
type ScanResult struct {
	Host string
	Open bool
}

// ScanHosts scans a list of hosts for an open port and returns hosts with the port open.
func (ps *PortScanner) ScanHosts(ctx context.Context, hosts []string, port int) []string {
	results := ps.ScanHostsWithResults(ctx, hosts, port)

	var openHosts []string
	for _, r := range results {
		if r.Open {
			openHosts = append(openHosts, r.Host)
		}
	}

	return openHosts
}

// ScanHostsWithResults scans hosts and returns detailed results.
func (ps *PortScanner) ScanHostsWithResults(ctx context.Context, hosts []string, port int) []ScanResult {
	if len(hosts) == 0 {
		return nil
	}

	// Create work channel
	work := make(chan string, len(hosts))
	for _, host := range hosts {
		work <- host
	}
	close(work)

	// Create results channel
	results := make(chan ScanResult, len(hosts))

	// Create semaphore for concurrency control
	sem := make(chan struct{}, ps.concurrency)

	// Start workers
	var wg sync.WaitGroup
	for host := range work {
		// Check context cancellation
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(h string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			open := ps.IsPortOpen(ctx, h, port)
			results <- ScanResult{Host: h, Open: open}
		}(host)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var scanResults []ScanResult
	for r := range results {
		scanResults = append(scanResults, r)
	}

	return scanResults
}
