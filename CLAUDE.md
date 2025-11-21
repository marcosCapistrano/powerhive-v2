# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PowerHive is a Go application for managing cryptocurrency mining hardware (Antminer series). It discovers miners on the network running either VNish or Stock Bitmain firmware, gathers power consumption/generation data, and adjusts miner power settings to match available power generation.

**Supported Miners:**
- Antminer S19 series (SHA256d/Bitcoin) - VNish or Stock firmware
- Antminer KS5/KS5 Pro (KHeavyHash/Kaspa) - Stock firmware

## Build & Development Commands

```bash
# Build the CLI
go build -o powerhive ./cmd/powerhive

# Build all packages
go build ./...

# Run tests
go test ./...

# Run a specific test
go test -v -run TestFunctionName ./pkg/...

# Run with race detection
go test -race ./...

# Lint (requires golangci-lint)
golangci-lint run
```

## CLI Usage

```bash
# Scan a network for miners
./powerhive scan 192.168.1.0/24

# Detect firmware on a specific miner
./powerhive detect 192.168.1.27

# Get detailed info from a miner
./powerhive info 192.168.1.27
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VNISH_PASSWORD` | VNish firmware password for authentication | `admin` |
| `STOCK_USERNAME` | Stock firmware username | `root` |
| `STOCK_PASSWORD` | Stock firmware password | `root` |

## Architecture

### Package Structure

```
powerhive-v2/
├── cmd/powerhive/         # CLI entry point (scan, detect, info commands)
├── pkg/
│   ├── miner/             # Shared miner abstractions (interfaces, types)
│   ├── vnish/             # VNish firmware API client
│   ├── stock/             # Stock Bitmain firmware API client
│   └── discovery/         # Network discovery for miners
└── internal/
    └── netutil/           # Internal network utilities (CIDR, port scanning)
```

### Core Abstractions (`pkg/miner/`)

Firmware-agnostic interfaces:

- `miner.Client` - Basic operations (GetMinerInfo, GetMinerStatus, Host)
- `miner.FirmwareProber` - Detects specific firmware and creates clients
- `miner.ClientFactory` - Creates clients for hosts
- `miner.Info`, `miner.Status` - Generic data types
- `miner.FirmwareType` - Enum (vnish, stock, braiins, unknown)

### VNish Package (`pkg/vnish/`)

VNish firmware implementation:

- `HTTPClient` - Full VNish API client
- `Prober` - Detects VNish firmware (implements `miner.FirmwareProber`)
- `AuthManager` - Bearer token + API key caching per miner

**Authentication:** Bearer tokens via `/api/v1/unlock`, API keys for sensitive ops

### Stock Package (`pkg/stock/`)

Bitmain stock firmware implementation:

- `HTTPClient` - Stock firmware API client with full CRUD operations
- `Prober` - Detects stock firmware (implements `miner.FirmwareProber`)
- `DigestAuth` - HTTP Digest authentication (default: root:root)

**GET Endpoints:**
- `GetSystemInfo()` - `/cgi-bin/get_system_info.cgi`
- `GetMinerConfig()` - `/cgi-bin/get_miner_conf.cgi`
- `GetSummary()` - `/cgi-bin/summary.cgi` (KS5/newer)
- `GetStats()` - `/cgi-bin/stats.cgi` (KS5/newer)
- `GetPools()` - `/cgi-bin/pools.cgi`
- `GetNetworkInfo()` - `/cgi-bin/get_network_info.cgi`
- `GetBlinkStatus()` - `/cgi-bin/get_blink_status.cgi`
- `GetLogs()` - `/cgi-bin/log.cgi`
- `GetMinerStatusFull()` - `/cgi-bin/get_miner_status.cgi` (S19/older)

**POST Endpoints:**
- `SetMinerConfig()` - `/cgi-bin/set_miner_conf.cgi`
- `SetNetworkConfig()` - `/cgi-bin/set_network_conf.cgi`
- `SetBlink()` - `/cgi-bin/blink.cgi`
- `Reboot()` - `/cgi-bin/reboot.cgi`
- `ResetConfig()` - `/cgi-bin/reset_conf.cgi`

**Model Differences:** KS5/newer models use `summary.cgi`/`stats.cgi`, S19/older use `get_miner_status.cgi`. The client automatically tries both with fallback.

### Discovery Package (`pkg/discovery/`)

Multi-firmware discovery:

- `Scanner` - Scans networks using multiple probers
- `Detector` - Tries each prober to identify firmware type
- `MultiProber` - Orchestrates multiple `FirmwareProber` implementations

### Dependency Injection Flow

```
cmd/powerhive/main.go
    │ creates
    ▼
[]miner.FirmwareProber  [vnish.Prober, stock.Prober]
    │ passed to
    ▼
discovery.Scanner / discovery.Detector
    │ tries each prober
    ▼
First successful → returns miner.Client
```

### Detection Order

1. **VNish** - Try `/api/v1/info`, check `fw_name == "Vnish"`
2. **Stock** - Try `/cgi-bin/get_system_info.cgi`, check for `minertype`

### VNish API Authentication

Three levels:

1. **Public** - No auth: `/info`, `/model`, `/status`, `/logs/*`, `/metrics`
2. **Bearer token** - Via `POST /unlock`: `/summary`, `/mining/*`, `/autotune/presets`
3. **Bearer + x-api-key** - Sensitive: `/settings`, `/system/reboot`

### Stock Firmware Authentication

HTTP Digest Auth with credentials (default: root:root)

### Key Domain Concepts

- **Autotune presets**: Power/hashrate profiles (1100W-4690W, 53TH-124TH)
- **Chains**: Mining boards with ASIC chips (typically 3 per miner)
- **Hashrate**: TH/s (terahashes), GH/s (gigahashes), or H/s depending on algorithm
- **Power efficiency**: J/TH (joules per terahash)
- **Algorithms**: SHA256d (Bitcoin), KHeavyHash (Kaspa)

## API Reference

- VNish: See `pkg/vnish/vnish-api.md`
- Stock: See `pkg/stock/stock-api.md`
