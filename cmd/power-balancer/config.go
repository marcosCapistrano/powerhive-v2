package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the power-balancer.
type Config struct {
	// Database (separate from powerhive.db)
	DBPath string

	// Energy Aggregator API
	AggregatorURL    string
	AggregatorAPIKey string

	// Network discovery (comma-separated CIDRs supported)
	NetworkCIDRs      []string
	DiscoveryInterval time.Duration

	// VNish authentication
	VNishPassword string

	// Margin thresholds (percentage of generation)
	EmergencyMargin float64 // Below this = EMERGENCY mode
	CriticalMargin  float64 // Below this = REDUCING mode
	SafeMargin      float64 // Target margin when reducing
	RecoveryMargin  float64 // Above this for 2min = INCREASING mode

	// Timing
	PollInterval     time.Duration // How often to check aggregator
	ChangeSpacing    time.Duration // Time between preset changes (reduce)
	RecoverySpacing  time.Duration // Time between preset changes (increase)
	CooldownDuration time.Duration // Per-miner cooldown after change
	SettleTime       time.Duration // Time for preset change to take effect

	// Limits
	MaxParallelEmergency int // Max parallel changes in emergency

	// Dashboard
	DashboardPort int
}

// DefaultConfig returns configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		DBPath:               "power-balancer.db",
		AggregatorURL:        "https://energy-aggregator.fly.dev/data/latest",
		AggregatorAPIKey:     "",
		NetworkCIDRs:         []string{},
		DiscoveryInterval:    5 * time.Minute,
		VNishPassword:        "admin",
		EmergencyMargin:      5.0,
		CriticalMargin:       10.0,
		SafeMargin:           15.0,
		RecoveryMargin:       20.0,
		PollInterval:         5 * time.Second,
		ChangeSpacing:        10 * time.Second,
		RecoverySpacing:      30 * time.Second,
		CooldownDuration:     10 * time.Minute,
		SettleTime:           5 * time.Minute,
		MaxParallelEmergency: 5,
		DashboardPort:        8081,
	}
}

// LoadConfig loads configuration from .env file and environment variables.
func LoadConfig() *Config {
	// Load .env file if it exists (doesn't error if missing)
	_ = godotenv.Load()

	cfg := DefaultConfig()

	if v := os.Getenv("POWER_BALANCER_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("AGGREGATOR_URL"); v != "" {
		cfg.AggregatorURL = v
	}
	if v := os.Getenv("AGGREGATOR_API_KEY"); v != "" {
		cfg.AggregatorAPIKey = v
	}
	if v := os.Getenv("NETWORK_CIDR"); v != "" {
		// Parse comma-separated CIDRs
		cidrs := strings.Split(v, ",")
		for _, cidr := range cidrs {
			cidr = strings.TrimSpace(cidr)
			if cidr != "" {
				cfg.NetworkCIDRs = append(cfg.NetworkCIDRs, cidr)
			}
		}
	}
	if v := os.Getenv("DISCOVERY_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.DiscoveryInterval = d
		}
	}
	if v := os.Getenv("VNISH_PASSWORD"); v != "" {
		cfg.VNishPassword = v
	}
	if v := os.Getenv("EMERGENCY_MARGIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.EmergencyMargin = f
		}
	}
	if v := os.Getenv("CRITICAL_MARGIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.CriticalMargin = f
		}
	}
	if v := os.Getenv("SAFE_MARGIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.SafeMargin = f
		}
	}
	if v := os.Getenv("RECOVERY_MARGIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.RecoveryMargin = f
		}
	}
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.PollInterval = d
		}
	}
	if v := os.Getenv("CHANGE_SPACING"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ChangeSpacing = d
		}
	}
	if v := os.Getenv("RECOVERY_SPACING"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RecoverySpacing = d
		}
	}
	if v := os.Getenv("COOLDOWN_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.CooldownDuration = d
		}
	}
	if v := os.Getenv("SETTLE_TIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.SettleTime = d
		}
	}
	if v := os.Getenv("MAX_PARALLEL_EMERGENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxParallelEmergency = n
		}
	}
	if v := os.Getenv("DASHBOARD_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.DashboardPort = n
		}
	}

	return cfg
}
