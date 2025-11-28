package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

// Controller handles VNish miner operations.
type Controller struct {
	auth *vnish.AuthManager
	cfg  *Config
}

// NewController creates a new controller instance.
func NewController(auth *vnish.AuthManager, cfg *Config) *Controller {
	return &Controller{
		auth: auth,
		cfg:  cfg,
	}
}

// SetPreset changes the preset on a miner.
func (c *Controller) SetPreset(ctx context.Context, ip string, presetName string) error {
	client := vnish.NewClient(ip, c.auth, vnish.WithTimeout(30*time.Second))

	// Ensure authenticated
	if err := client.EnsureAuthenticated(ctx); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	// Ensure API key for settings changes
	if err := client.EnsureAPIKey(ctx); err != nil {
		return fmt.Errorf("get API key: %w", err)
	}

	// Set the preset
	if err := client.SetPreset(ctx, presetName); err != nil {
		return fmt.Errorf("set preset: %w", err)
	}

	// Check if restart is required
	status, err := client.GetStatus(ctx)
	if err != nil {
		log.Printf("[%s] Warning: couldn't check restart status: %v", ip, err)
		return nil
	}

	if status.RestartRequired {
		log.Printf("[%s] Restart required, restart it manually: ", ip)
		// if err := client.RestartMining(ctx); err != nil {
		// 	log.Printf("[%s] Warning: restart failed: %v", ip, err)
		// }
	}

	return nil
}

// GetCurrentPreset gets the current preset from a miner.
func (c *Controller) GetCurrentPreset(ctx context.Context, ip string) (string, error) {
	client := vnish.NewClient(ip, c.auth, vnish.WithTimeout(10*time.Second))

	perf, err := client.GetPerfSummary(ctx)
	if err != nil {
		return "", fmt.Errorf("get perf summary: %w", err)
	}

	return perf.CurrentPreset.Name, nil
}

// GetAvailablePresets gets all available presets for a miner.
func (c *Controller) GetAvailablePresets(ctx context.Context, ip string) ([]vnish.AutotunePreset, error) {
	client := vnish.NewClient(ip, c.auth, vnish.WithTimeout(10*time.Second))

	// Ensure authenticated for presets endpoint
	if err := client.EnsureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	presets, err := client.GetAutotunePresets(ctx)
	if err != nil {
		return nil, fmt.Errorf("get presets: %w", err)
	}

	return presets, nil
}

// GetMinerInfo gets info from a miner.
func (c *Controller) GetMinerInfo(ctx context.Context, ip string) (*vnish.MinerInfo, error) {
	client := vnish.NewClient(ip, c.auth, vnish.WithTimeout(10*time.Second))
	return client.GetInfo(ctx)
}
