package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the data-harvest CLI.
type Config struct {
	// Database
	DBPath string

	// Authentication
	VNishPassword string
	StockUsername string
	StockPassword string

	// Harvesting
	HarvestInterval time.Duration
	Concurrency     int
	Timeout         time.Duration

	// Network (comma-separated CIDRs supported via NETWORK_CIDR env var)
	NetworkCIDRs []string
}

// DefaultConfig returns configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		DBPath:          "powerhive.db",
		VNishPassword:   "admin",
		StockUsername:   "root",
		StockPassword:   "root",
		HarvestInterval: 30 * time.Second,
		Concurrency:     25,
		Timeout:         10 * time.Second,
	}
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	// Load .env file if present (ignore error if not found)
	_ = godotenv.Load()

	cfg := DefaultConfig()

	if v := os.Getenv("POWERHIVE_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("VNISH_PASSWORD"); v != "" {
		cfg.VNishPassword = v
	}
	if v := os.Getenv("STOCK_USERNAME"); v != "" {
		cfg.StockUsername = v
	}
	if v := os.Getenv("STOCK_PASSWORD"); v != "" {
		cfg.StockPassword = v
	}
	if v := os.Getenv("HARVEST_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.HarvestInterval = d
		}
	}
	if v := os.Getenv("HARVEST_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Concurrency = n
		}
	}
	if v := os.Getenv("HARVEST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	// Parse NETWORK_CIDR (comma-separated CIDRs, same format as power-balancer)
	if v := os.Getenv("NETWORK_CIDR"); v != "" {
		cidrs := strings.Split(v, ",")
		for _, cidr := range cidrs {
			cidr = strings.TrimSpace(cidr)
			if cidr != "" {
				cfg.NetworkCIDRs = append(cfg.NetworkCIDRs, cidr)
			}
		}
	}

	return cfg
}
