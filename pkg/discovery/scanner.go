package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/powerhive/powerhive-v2/internal/netutil"
	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Scanner discovers miners on the network.
type Scanner struct {
	probers     []miner.FirmwareProber
	portScanner *netutil.PortScanner
	detector    *Detector
	opts        ScanOptions
}

// ScannerOption configures a Scanner.
type ScannerOption func(*Scanner)

// WithTimeout sets the timeout for each host.
func WithTimeout(timeout time.Duration) ScannerOption {
	return func(s *Scanner) {
		s.opts.Timeout = timeout
	}
}

// WithConcurrency sets the maximum concurrent scans.
func WithConcurrency(concurrency int) ScannerOption {
	return func(s *Scanner) {
		s.opts.Concurrency = concurrency
	}
}

// WithPort sets the port to scan.
func WithPort(port int) ScannerOption {
	return func(s *Scanner) {
		s.opts.Port = port
	}
}

// WithSkipDetection only checks port connectivity without miner detection.
func WithSkipDetection(skip bool) ScannerOption {
	return func(s *Scanner) {
		s.opts.SkipDetection = skip
	}
}

// NewScanner creates a new network scanner with firmware probers.
// Probers are tried in order - typically VNish first (more features), then Stock.
func NewScanner(probers []miner.FirmwareProber, opts ...ScannerOption) *Scanner {
	s := &Scanner{
		probers: probers,
		opts:    DefaultScanOptions(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize port scanner with matching settings
	s.portScanner = netutil.NewPortScanner(
		netutil.WithScanTimeout(s.opts.Timeout),
		netutil.WithScanConcurrency(s.opts.Concurrency),
	)

	// Initialize detector with probers
	s.detector = NewDetector(probers, WithDetectorTimeout(s.opts.Timeout))

	return s
}

// ScanNetwork scans a CIDR range for miners.
// Example: "192.168.1.0/24"
func (s *Scanner) ScanNetwork(ctx context.Context, cidr string) (*ScanResult, error) {
	ips, err := netutil.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	return s.scanIPs(ctx, ips)
}

// ScanRange scans an IP range for miners.
// Example: "192.168.1.1" to "192.168.1.254"
func (s *Scanner) ScanRange(ctx context.Context, startIP, endIP string) (*ScanResult, error) {
	ips, err := netutil.ParseRange(startIP, endIP)
	if err != nil {
		return nil, err
	}

	return s.scanIPs(ctx, ips)
}

// ScanHosts scans specific IP addresses for miners.
func (s *Scanner) ScanHosts(ctx context.Context, hosts []string) (*ScanResult, error) {
	return s.scanIPs(ctx, hosts)
}

// scanIPs performs the actual scanning of IP addresses.
func (s *Scanner) scanIPs(ctx context.Context, ips []string) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		Miners:     make([]DiscoveredMiner, 0),
		Errors:     make(map[string]error),
		ScannedIPs: len(ips),
	}

	// Phase 1: Port scan to find responsive hosts
	responsiveHosts := s.portScanner.ScanHosts(ctx, ips, s.opts.Port)
	result.ResponsiveHosts = len(responsiveHosts)

	// If skip detection, just return hosts with open ports
	if s.opts.SkipDetection {
		for _, host := range responsiveHosts {
			result.Miners = append(result.Miners, DiscoveredMiner{
				IP:           host,
				DiscoveredAt: time.Now(),
			})
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Phase 2: Detect miners on responsive hosts
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, s.opts.Concurrency)
	)

scanLoop:
	for _, host := range responsiveHosts {
		select {
		case <-ctx.Done():
			break scanLoop
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			discovered, err := s.detector.DetectMiner(ctx, ip)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Errors[ip] = err
			} else if discovered != nil {
				result.Miners = append(result.Miners, *discovered)
			}
		}(host)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	return result, nil
}
