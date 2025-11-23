package vnish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Client is the interface for interacting with VNish firmware API.
type Client interface {
	// Info & Status
	GetInfo(ctx context.Context) (*MinerInfo, error)
	GetModel(ctx context.Context) (*ModelInfo, error)
	GetStatus(ctx context.Context) (*MinerStatus, error)
	GetSummary(ctx context.Context) (*Summary, error)
	GetPerfSummary(ctx context.Context) (*PerfSummary, error)

	// Chains
	GetChains(ctx context.Context) ([]Chain, error)
	GetChainsFactoryInfo(ctx context.Context) (*ChainFactoryInfo, error)

	// Autotune
	GetAutotunePresets(ctx context.Context) ([]AutotunePreset, error)

	// Logs
	GetStatusLogs(ctx context.Context) (string, error)
	GetMinerLogs(ctx context.Context) (string, error)
	GetAutotuneLogs(ctx context.Context) (string, error)
	GetSystemLogs(ctx context.Context) (string, error)
	GetMessagesLogs(ctx context.Context) (string, error)
	GetAPILogs(ctx context.Context) (string, error)

	// Metrics
	GetMetrics(ctx context.Context, timeSlice, step int) (*Metrics, error)

	// Notes
	GetNotes(ctx context.Context) (map[string]string, error)
	GetNote(ctx context.Context, key string) (string, error)
	AddNote(ctx context.Context, key, value string) error
	UpdateNote(ctx context.Context, key, value string) error
	DeleteNote(ctx context.Context, key string) error

	// API Keys
	GetAPIKeys(ctx context.Context) ([]APIKey, error)
	AddAPIKey(ctx context.Context, key, description string) error
	DeleteAPIKey(ctx context.Context, key string) error

	// Settings
	GetSettings(ctx context.Context) (map[string]interface{}, error)
	SaveSettings(ctx context.Context, settings *SettingsUpdate) error
	BackupSettings(ctx context.Context) ([]byte, error)
	RestoreSettings(ctx context.Context, backup []byte) error
	FactoryReset(ctx context.Context) error

	// Mining Control
	StartMining(ctx context.Context) error
	StopMining(ctx context.Context) error
	PauseMining(ctx context.Context) error
	ResumeMining(ctx context.Context) error
	RestartMining(ctx context.Context) error
	SwitchPool(ctx context.Context, poolID int64) error

	// System Operations
	FindMiner(ctx context.Context) (bool, error)
	Reboot(ctx context.Context) error

	// Authentication
	Unlock(ctx context.Context) (string, error)
	EnsureAuthenticated(ctx context.Context) error
	EnsureAPIKey(ctx context.Context) error
}

// HTTPClient is the HTTP implementation of the VNish Client interface.
type HTTPClient struct {
	host       string
	baseURL    string
	httpClient *http.Client
	auth       *AuthManager
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

// NewClient creates a new VNish HTTP client.
func NewClient(host string, auth *AuthManager, opts ...ClientOption) *HTTPClient {
	c := &HTTPClient{
		host:    host,
		baseURL: fmt.Sprintf("http://%s/api/v1", host),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth: auth,
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

// request performs an HTTP request with optional authentication.
type requestOptions struct {
	method       string
	endpoint     string
	body         interface{}
	result       interface{}
	requiresAuth bool
	requiresKey  bool
	isText       bool // response is plain text, not JSON
}

func (c *HTTPClient) request(ctx context.Context, opts requestOptions) error {
	fullURL := c.baseURL + opts.endpoint

	var bodyReader io.Reader
	if opts.body != nil {
		bodyBytes, err := json.Marshal(opts.body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, opts.method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if opts.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if opts.isText {
		req.Header.Set("Accept", "*/*")
	} else {
		req.Header.Set("Accept", "application/json")
	}

	// Add authentication if required
	if opts.requiresAuth {
		token := c.auth.GetToken(c.host)
		if token == "" {
			// Try to authenticate first
			if err := c.EnsureAuthenticated(ctx); err != nil {
				return err
			}
			token = c.auth.GetToken(c.host)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Add API key if required
	if opts.requiresKey {
		apiKey := c.auth.GetAPIKey(c.host)
		if apiKey == "" {
			// Try to create an API key
			if err := c.EnsureAPIKey(ctx); err != nil {
				return err
			}
			apiKey = c.auth.GetAPIKey(c.host)
		}
		req.Header.Set("x-api-key", apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle auth errors with retry
	if resp.StatusCode == http.StatusUnauthorized {
		c.auth.ClearToken(c.host)
		if opts.requiresAuth {
			// Retry with fresh token
			if err := c.EnsureAuthenticated(ctx); err != nil {
				return err
			}
			return c.request(ctx, opts)
		}
		return &APIError{StatusCode: resp.StatusCode, Endpoint: opts.endpoint}
	}

	if resp.StatusCode == http.StatusForbidden {
		c.auth.ClearAPIKey(c.host)
		if opts.requiresKey {
			// Retry with fresh API key
			if err := c.EnsureAPIKey(ctx); err != nil {
				return err
			}
			return c.request(ctx, opts)
		}
		return &APIError{StatusCode: resp.StatusCode, Endpoint: opts.endpoint}
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Err != "" {
			return &APIError{StatusCode: resp.StatusCode, Message: errResp.Err, Endpoint: opts.endpoint}
		}
		return &APIError{StatusCode: resp.StatusCode, Message: string(bodyBytes), Endpoint: opts.endpoint}
	}

	// Parse successful response
	if opts.result != nil {
		if opts.isText {
			if strPtr, ok := opts.result.(*string); ok {
				*strPtr = string(bodyBytes)
				return nil
			}
		}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, opts.result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}
	}

	return nil
}

// requestRaw performs an HTTP request and returns the raw response bytes.
func (c *HTTPClient) requestRaw(ctx context.Context, opts requestOptions) ([]byte, error) {
	fullURL := c.baseURL + opts.endpoint

	req, err := http.NewRequestWithContext(ctx, opts.method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Add authentication if required
	if opts.requiresAuth {
		token := c.auth.GetToken(c.host)
		if token == "" {
			if err := c.EnsureAuthenticated(ctx); err != nil {
				return nil, err
			}
			token = c.auth.GetToken(c.host)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(bodyBytes), Endpoint: opts.endpoint}
	}

	return bodyBytes, nil
}

// Unlock authenticates with the miner and returns a bearer token.
func (c *HTTPClient) Unlock(ctx context.Context) (string, error) {
	var result UnlockResponse
	err := c.request(ctx, requestOptions{
		method:   http.MethodPost,
		endpoint: "/unlock",
		body:     &UnlockRequest{Password: c.auth.GetPassword()},
		result:   &result,
	})
	if err != nil {
		return "", fmt.Errorf("unlock failed: %w", err)
	}

	c.auth.SetToken(c.host, result.Token)
	return result.Token, nil
}

// EnsureAuthenticated ensures we have a valid bearer token.
func (c *HTTPClient) EnsureAuthenticated(ctx context.Context) error {
	if token := c.auth.GetToken(c.host); token != "" {
		return nil
	}

	_, err := c.Unlock(ctx)
	return err
}

// EnsureAPIKey ensures we have a valid API key, creating one if necessary.
func (c *HTTPClient) EnsureAPIKey(ctx context.Context) error {
	if key := c.auth.GetAPIKey(c.host); key != "" {
		return nil
	}

	// Ensure we're authenticated first
	if err := c.EnsureAuthenticated(ctx); err != nil {
		return err
	}

	// Generate a new API key
	newKey, err := GenerateAPIKey()
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	// Register the API key with the miner
	if err := c.AddAPIKey(ctx, newKey, "powerhive-auto-generated"); err != nil {
		return fmt.Errorf("failed to register API key: %w", err)
	}

	c.auth.SetAPIKey(c.host, newKey)
	return nil
}

// GetInfo returns detailed miner information.
func (c *HTTPClient) GetInfo(ctx context.Context) (*MinerInfo, error) {
	var result MinerInfo
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/info",
		result:   &result,
	})
	return &result, err
}

// GetModel returns model-specific information.
func (c *HTTPClient) GetModel(ctx context.Context) (*ModelInfo, error) {
	var result ModelInfo
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/model",
		result:   &result,
	})
	return &result, err
}

// GetStatus returns current miner operational status.
func (c *HTTPClient) GetStatus(ctx context.Context) (*MinerStatus, error) {
	var result MinerStatus
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/status",
		result:   &result,
	})
	return &result, err
}

// GetSummary returns comprehensive mining summary.
func (c *HTTPClient) GetSummary(ctx context.Context) (*Summary, error) {
	var result Summary
	err := c.request(ctx, requestOptions{
		method:       http.MethodGet,
		endpoint:     "/summary",
		result:       &result,
		requiresAuth: true,
	})
	return &result, err
}

// GetSummaryRaw returns the raw JSON response from the summary endpoint (for debugging).
func (c *HTTPClient) GetSummaryRaw(ctx context.Context) ([]byte, error) {
	return c.requestRaw(ctx, requestOptions{
		method:       http.MethodGet,
		endpoint:     "/summary",
		requiresAuth: true,
	})
}

// GetPerfSummary returns performance summary with autotune info.
func (c *HTTPClient) GetPerfSummary(ctx context.Context) (*PerfSummary, error) {
	var result PerfSummary
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/perf-summary",
		result:   &result,
	})
	return &result, err
}

// GetChains returns list of mining chains.
func (c *HTTPClient) GetChains(ctx context.Context) ([]Chain, error) {
	var result []Chain
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/chains",
		result:   &result,
	})
	return result, err
}

// GetChainsFactoryInfo returns factory information for chains.
func (c *HTTPClient) GetChainsFactoryInfo(ctx context.Context) (*ChainFactoryInfo, error) {
	var result ChainFactoryInfo
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/chains/factory-info",
		result:   &result,
	})
	return &result, err
}

// GetAutotunePresets returns available autotune presets.
func (c *HTTPClient) GetAutotunePresets(ctx context.Context) ([]AutotunePreset, error) {
	var result []AutotunePreset
	err := c.request(ctx, requestOptions{
		method:       http.MethodGet,
		endpoint:     "/autotune/presets",
		result:       &result,
		requiresAuth: true,
	})
	return result, err
}

// GetStatusLogs returns miner status logs.
func (c *HTTPClient) GetStatusLogs(ctx context.Context) (string, error) {
	var result string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/logs/status",
		result:   &result,
		isText:   true,
	})
	return result, err
}

// GetMinerLogs returns detailed miner hardware logs.
func (c *HTTPClient) GetMinerLogs(ctx context.Context) (string, error) {
	var result string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/logs/miner",
		result:   &result,
		isText:   true,
	})
	return result, err
}

// GetAutotuneLogs returns autotune-specific logs.
func (c *HTTPClient) GetAutotuneLogs(ctx context.Context) (string, error) {
	var result string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/logs/autotune",
		result:   &result,
		isText:   true,
	})
	return result, err
}

// GetSystemLogs returns system boot logs.
func (c *HTTPClient) GetSystemLogs(ctx context.Context) (string, error) {
	var result string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/logs/system",
		result:   &result,
		isText:   true,
	})
	return result, err
}

// GetMessagesLogs returns system messages.
func (c *HTTPClient) GetMessagesLogs(ctx context.Context) (string, error) {
	var result string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/logs/messages",
		result:   &result,
		isText:   true,
	})
	return result, err
}

// GetAPILogs returns API access logs.
func (c *HTTPClient) GetAPILogs(ctx context.Context) (string, error) {
	var result string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/logs/api",
		result:   &result,
		isText:   true,
	})
	return result, err
}

// GetMetrics returns time-series metrics data.
func (c *HTTPClient) GetMetrics(ctx context.Context, timeSlice, step int) (*Metrics, error) {
	endpoint := "/metrics"
	if timeSlice > 0 || step > 0 {
		params := url.Values{}
		if timeSlice > 0 {
			params.Set("time_slice", fmt.Sprintf("%d", timeSlice))
		}
		if step > 0 {
			params.Set("step", fmt.Sprintf("%d", step))
		}
		endpoint += "?" + params.Encode()
	}

	var result Metrics
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: endpoint,
		result:   &result,
	})
	return &result, err
}

// GetNotes returns all stored notes.
func (c *HTTPClient) GetNotes(ctx context.Context) (map[string]string, error) {
	var result map[string]string
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/notes",
		result:   &result,
	})
	if result == nil {
		result = make(map[string]string)
	}
	return result, err
}

// GetNote retrieves a specific note by key.
func (c *HTTPClient) GetNote(ctx context.Context, key string) (string, error) {
	var result Note
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/notes/" + url.PathEscape(key),
		result:   &result,
	})
	return result.Value, err
}

// AddNote creates a new note.
func (c *HTTPClient) AddNote(ctx context.Context, key, value string) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/notes",
		body:         &Note{Key: key, Value: value},
		requiresAuth: true,
	})
}

// UpdateNote updates an existing note.
func (c *HTTPClient) UpdateNote(ctx context.Context, key, value string) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPut,
		endpoint:     "/notes/" + url.PathEscape(key),
		body:         &Note{Value: value},
		requiresAuth: true,
	})
}

// DeleteNote deletes a note.
func (c *HTTPClient) DeleteNote(ctx context.Context, key string) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodDelete,
		endpoint:     "/notes/" + url.PathEscape(key),
		requiresAuth: true,
	})
}

// GetAPIKeys returns list of configured API keys.
func (c *HTTPClient) GetAPIKeys(ctx context.Context) ([]APIKey, error) {
	var result []APIKey
	err := c.request(ctx, requestOptions{
		method:       http.MethodGet,
		endpoint:     "/apikeys",
		result:       &result,
		requiresAuth: true,
	})
	if result == nil {
		result = []APIKey{}
	}
	return result, err
}

// AddAPIKey creates a new API key on the miner.
func (c *HTTPClient) AddAPIKey(ctx context.Context, key, description string) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/apikeys",
		body:         &APIKeyRequest{Key: key, Description: description},
		requiresAuth: true,
	})
}

// DeleteAPIKey deletes an API key from the miner.
func (c *HTTPClient) DeleteAPIKey(ctx context.Context, key string) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/apikeys/delete",
		body:         &APIKeyRequest{Key: key},
		requiresAuth: true,
		requiresKey:  true,
	})
}

// GetSettings returns complete miner configuration.
func (c *HTTPClient) GetSettings(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.request(ctx, requestOptions{
		method:   http.MethodGet,
		endpoint: "/settings",
		result:   &result,
	})
	return result, err
}

// SaveSettings updates miner configuration.
func (c *HTTPClient) SaveSettings(ctx context.Context, settings *SettingsUpdate) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/settings",
		body:         settings,
		requiresAuth: true,
		requiresKey:  true,
	})
}

// BackupSettings creates a backup of all miner settings.
func (c *HTTPClient) BackupSettings(ctx context.Context) ([]byte, error) {
	fullURL := c.baseURL + "/settings/backup"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Ensure authentication
	if err := c.EnsureAuthenticated(ctx); err != nil {
		return nil, err
	}
	if err := c.EnsureAPIKey(ctx); err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+c.auth.GetToken(c.host))
	req.Header.Set("x-api-key", c.auth.GetAPIKey(c.host))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Endpoint: "/settings/backup"}
	}

	return io.ReadAll(resp.Body)
}

// RestoreSettings restores settings from a backup.
func (c *HTTPClient) RestoreSettings(ctx context.Context, backup []byte) error {
	// This requires multipart form data - simplified implementation
	return fmt.Errorf("RestoreSettings not yet implemented (requires multipart form)")
}

// FactoryReset resets all settings to factory defaults.
func (c *HTTPClient) FactoryReset(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/settings/factory-reset",
		requiresAuth: true,
		requiresKey:  true,
	})
}

// StartMining starts mining operations.
func (c *HTTPClient) StartMining(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/mining/start",
		requiresAuth: true,
	})
}

// StopMining stops mining operations.
func (c *HTTPClient) StopMining(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/mining/stop",
		requiresAuth: true,
	})
}

// PauseMining pauses mining operations.
func (c *HTTPClient) PauseMining(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/mining/pause",
		requiresAuth: true,
	})
}

// ResumeMining resumes paused mining operations.
func (c *HTTPClient) ResumeMining(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/mining/resume",
		requiresAuth: true,
	})
}

// RestartMining restarts mining operations.
func (c *HTTPClient) RestartMining(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/mining/restart",
		requiresAuth: true,
	})
}

// SwitchPool switches to a different mining pool.
func (c *HTTPClient) SwitchPool(ctx context.Context, poolID int64) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/mining/switch-pool",
		body:         &SwitchPoolRequest{PoolID: poolID},
		requiresAuth: true,
	})
}

// FindMiner toggles the miner's LED for physical identification.
func (c *HTTPClient) FindMiner(ctx context.Context) (bool, error) {
	var result FindMinerResponse
	err := c.request(ctx, requestOptions{
		method:   http.MethodPost,
		endpoint: "/find-miner",
		result:   &result,
	})
	return result.On, err
}

// Reboot reboots the miner system.
func (c *HTTPClient) Reboot(ctx context.Context) error {
	return c.request(ctx, requestOptions{
		method:       http.MethodPost,
		endpoint:     "/system/reboot",
		requiresAuth: true,
		requiresKey:  true,
	})
}

// =============================================================================
// miner.Client interface implementation
// =============================================================================

// GetMinerInfo returns generic miner information (implements miner.Client).
func (c *HTTPClient) GetMinerInfo(ctx context.Context) (*miner.Info, error) {
	info, err := c.GetInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Get model info for series
	model, err := c.GetModel(ctx)
	series := ""
	if err == nil && model != nil {
		series = model.Series
	}

	return &miner.Info{
		Miner:           info.Miner,
		Model:           info.Model,
		Series:          series,
		Firmware:        info.FWName,
		FirmwareVersion: info.FWVersion,
		Algorithm:       info.Algorithm,
		IP:              info.System.NetworkStatus.IP,
		MAC:             info.System.NetworkStatus.MAC,
		Hostname:        info.System.NetworkStatus.Hostname,
	}, nil
}

// GetMinerStatus returns generic miner status (implements miner.Client).
func (c *HTTPClient) GetMinerStatus(ctx context.Context) (*miner.Status, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	return &miner.Status{
		State:       status.MinerState,
		Description: status.Description,
		FailureCode: status.FailureCode,
	}, nil
}

// Ensure HTTPClient implements miner.Client interface.
var _ miner.Client = (*HTTPClient)(nil)
