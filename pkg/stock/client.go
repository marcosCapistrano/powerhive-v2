package stock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Client is the interface for interacting with stock Bitmain firmware.
type Client interface {
	miner.Client

	// GetSystemInfo returns system information.
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)

	// GetMinerStatusFull returns full miner status with summary, pools, and devs.
	// This uses get_miner_status.cgi (S19 and older models).
	GetMinerStatusFull(ctx context.Context) (*MinerStatus, error)

	// GetMinerConfig returns miner configuration.
	GetMinerConfig(ctx context.Context) (*MinerConfig, error)

	// GetStats returns detailed mining statistics from stats.cgi (KS5, newer models).
	GetStats(ctx context.Context) (*StatsResponse, error)

	// GetSummary returns summary mining data from summary.cgi (KS5, newer models).
	GetSummary(ctx context.Context) (*SummaryResponse, error)
}

// HTTPClient is the HTTP implementation of the stock firmware Client.
type HTTPClient struct {
	host       string
	baseURL    string
	httpClient *http.Client
}

// ClientOption is a function that configures an HTTPClient.
type ClientOption func(*HTTPClient)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient = client
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new stock firmware HTTP client.
func NewClient(host string, auth *DigestAuth, opts ...ClientOption) *HTTPClient {
	transport := &DigestTransport{
		Auth:      auth,
		Transport: http.DefaultTransport,
	}

	c := &HTTPClient{
		host:    host,
		baseURL: fmt.Sprintf("http://%s/cgi-bin", host),
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Host returns the miner host address.
func (c *HTTPClient) Host() string {
	return c.host
}

// request performs an HTTP GET request.
func (c *HTTPClient) request(ctx context.Context, endpoint string, result interface{}) error {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// tryRequest performs an HTTP GET request and returns false if endpoint doesn't exist (404).
// Returns (true, nil) on success, (true, err) on other errors, (false, nil) if endpoint not found.
func (c *HTTPClient) tryRequest(ctx context.Context, endpoint string, result interface{}) (bool, error) {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return true, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return true, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// If 404, endpoint doesn't exist on this firmware
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return true, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return true, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return true, fmt.Errorf("failed to parse response: %w", err)
	}

	return true, nil
}

// GetSystemInfo returns system information.
func (c *HTTPClient) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	var result SystemInfo
	err := c.request(ctx, "/get_system_info.cgi", &result)
	return &result, err
}

// GetMinerStatusFull returns full miner status.
func (c *HTTPClient) GetMinerStatusFull(ctx context.Context) (*MinerStatus, error) {
	var result MinerStatus
	err := c.request(ctx, "/get_miner_status.cgi", &result)
	return &result, err
}

// GetMinerConfig returns miner configuration.
func (c *HTTPClient) GetMinerConfig(ctx context.Context) (*MinerConfig, error) {
	var result MinerConfig
	err := c.request(ctx, "/get_miner_conf.cgi", &result)
	return &result, err
}

// GetStats returns detailed mining statistics from stats.cgi (KS5, newer models).
func (c *HTTPClient) GetStats(ctx context.Context) (*StatsResponse, error) {
	var result StatsResponse
	err := c.request(ctx, "/stats.cgi", &result)
	return &result, err
}

// GetSummary returns summary mining data from summary.cgi (KS5, newer models).
func (c *HTTPClient) GetSummary(ctx context.Context) (*SummaryResponse, error) {
	var result SummaryResponse
	err := c.request(ctx, "/summary.cgi", &result)
	return &result, err
}

// =============================================================================
// miner.Client interface implementation
// =============================================================================

// GetMinerInfo returns generic miner information (implements miner.Client).
func (c *HTTPClient) GetMinerInfo(ctx context.Context) (*miner.Info, error) {
	sysInfo, err := c.GetSystemInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &miner.Info{
		Miner:           sysInfo.MinerType,
		Model:           sysInfo.MinerType,
		Series:          extractSeries(sysInfo.MinerType),
		Firmware:        "Stock",
		FirmwareVersion: sysInfo.SystemFilesystemVersion,
		Algorithm:       "sha256d",
		IP:              sysInfo.IPAddress,
		MAC:             sysInfo.MACAddr,
		Hostname:        sysInfo.Hostname,
	}, nil
}

// GetMinerStatus returns generic miner status (implements miner.Client).
// Tries summary.cgi first (KS5, newer models), falls back to get_miner_status.cgi (S19, older models).
func (c *HTTPClient) GetMinerStatus(ctx context.Context) (*miner.Status, error) {
	// Try summary.cgi first (KS5, newer models)
	var summaryResp SummaryResponse
	found, err := c.tryRequest(ctx, "/summary.cgi", &summaryResp)
	if found && err == nil && len(summaryResp.Summary) > 0 {
		summary := summaryResp.Summary[0]

		// Determine state based on hashrate and elapsed time
		state := "unknown"
		if summary.Elapsed > 0 {
			if summary.Rate5s > 0 {
				state = "running"
			} else {
				state = "idle"
			}
		}

		// Check status items for warnings/errors
		description := fmt.Sprintf("Hashrate: %.2f %s", summary.Rate5s, summary.RateUnit)
		for _, s := range summary.Status {
			if s.Status == "e" || s.Status == "w" {
				description += fmt.Sprintf(" [%s: %s]", s.Type, s.Msg)
			}
		}

		return &miner.Status{
			State:       state,
			Description: description,
			FailureCode: 0,
		}, nil
	}

	// Fallback to get_miner_status.cgi (S19, older models)
	var minerStatus MinerStatus
	found, err = c.tryRequest(ctx, "/get_miner_status.cgi", &minerStatus)
	if found && err == nil {
		// Determine state based on hashrate and elapsed time
		state := "unknown"
		if minerStatus.Summary.Elapsed > 0 {
			if minerStatus.Summary.GHS5s > 0 {
				state = "running"
			} else {
				state = "idle"
			}
		}

		return &miner.Status{
			State:       state,
			Description: fmt.Sprintf("Hashrate: %.2f GH/s", minerStatus.Summary.GHS5s),
			FailureCode: 0,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("no status endpoint available")
}

// extractSeries extracts the series from miner type (e.g., "Antminer S19" -> "x19").
func extractSeries(minerType string) string {
	// Simple extraction - look for S19, S17, T19, etc.
	if len(minerType) >= 3 {
		for i := range minerType {
			if i+3 <= len(minerType) {
				prefix := minerType[i : i+1]
				if prefix == "S" || prefix == "T" || prefix == "L" {
					num := minerType[i+1 : i+3]
					if num[0] >= '0' && num[0] <= '9' {
						return "x" + num
					}
				}
			}
		}
	}
	return ""
}

// Ensure HTTPClient implements miner.Client interface.
var _ miner.Client = (*HTTPClient)(nil)
