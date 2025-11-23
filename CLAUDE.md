# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PowerHive is a Go application for managing cryptocurrency mining hardware (Antminer series). It discovers miners on the network running either VNish or Stock Bitmain firmware, gathers power consumption/generation data, stores metrics in SQLite, and provides a real-time web dashboard.

## Build & Development Commands

```bash
# Build all binaries
go build ./...

# Build individual components
go build -o powerhive ./cmd/powerhive
go build -o data-harvest ./cmd/data-harvest
go build -o harvest-dashboard ./cmd/harvest-dashboard

# Run tests
go test ./...

# Run a specific test
go test -v -run TestFunctionName ./pkg/...

# Run with race detection
go test -race ./...

# Lint (requires golangci-lint)
golangci-lint run
```

## CLI Tools

### powerhive - Quick discovery tool

```bash
# Scan a network for miners
./powerhive scan 192.168.1.0/24

# Detect firmware on a specific miner
./powerhive detect 192.168.1.27

# Get detailed info from a miner
./powerhive info 192.168.1.27
```

### data-harvest - Data collection daemon

```bash
# Scan network and harvest data to SQLite
./data-harvest scan 192.168.1.0/24

# Harvest specific miners
./data-harvest harvest 192.168.1.27 192.168.1.28

# Run as daemon (continuous harvesting)
./data-harvest daemon 192.168.1.0/24

# List all known miners
./data-harvest list

# Show miner details
./data-harvest show 192.168.1.27
```

### harvest-dashboard - Web dashboard

```bash
# Start dashboard (default port 8080)
./harvest-dashboard

# Custom port
DASHBOARD_PORT=3000 ./harvest-dashboard
```

Dashboard features:
- Real-time miner overview (online/offline, hashrate, power)
- Miner detail pages with performance charts
- SSE-based live updates (no page refresh needed)
- Log viewer for miner system logs
- Filtering and sorting

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VNISH_PASSWORD` | VNish firmware password | `admin` |
| `STOCK_USERNAME` | Stock firmware username | `root` |
| `STOCK_PASSWORD` | Stock firmware password | `root` |
| `POWERHIVE_DB` | SQLite database path | `powerhive.db` |
| `DASHBOARD_PORT` | Dashboard HTTP port | `8080` |
| `HARVEST_INTERVAL` | Daemon polling interval | `60s` |
| `HARVEST_CONCURRENCY` | Parallel harvest workers | `10` |
| `HARVEST_TIMEOUT` | Per-miner timeout | `10s` |
| `DEFAULT_NETWORK` | Default network for daemon | - |

## Architecture

### Package Structure

```
powerhive-v2/
├── cmd/
│   ├── powerhive/           # Quick discovery CLI
│   ├── data-harvest/        # Data collection daemon
│   │   ├── main.go          # CLI entry point
│   │   ├── config.go        # Configuration loading
│   │   ├── harvest.go       # Harvester orchestration
│   │   ├── collector.go     # Per-miner data collection
│   │   └── log_collector.go # Log fetching and parsing
│   └── harvest-dashboard/   # Web dashboard
│       ├── main.go          # HTTP server and routes
│       ├── sse.go           # Server-Sent Events for live updates
│       ├── templates/       # HTML templates
│       └── static/          # CSS/JS assets
├── pkg/
│   ├── miner/               # Shared miner abstractions
│   ├── vnish/               # VNish firmware API client
│   ├── stock/               # Stock Bitmain firmware API client
│   ├── discovery/           # Network discovery
│   └── database/            # SQLite storage layer
│       ├── models.go        # Data models (Miner, Status, Summary, etc.)
│       ├── schema.go        # Database schema
│       ├── repository.go    # Repository interface
│       ├── sqlite.go        # SQLite implementation
│       ├── mapper_vnish.go  # VNish API → database mapping
│       ├── mapper_stock.go  # Stock API → database mapping
│       └── log_parser.go    # Log line parsing
└── internal/
    └── netutil/             # CIDR parsing, port scanning
```

### Core Abstractions (`pkg/miner/`)

Firmware-agnostic interfaces:

- `miner.Client` - Basic operations (GetMinerInfo, GetMinerStatus, Host)
- `miner.FirmwareProber` - Detects specific firmware and creates clients
- `miner.ClientFactory` - Creates clients for hosts
- `miner.Info`, `miner.Status` - Generic data types
- `miner.FirmwareType` - Enum (vnish, stock, unknown)

### Database Models (`pkg/database/`)

SQLite storage with the following tables:

| Table | Description |
|-------|-------------|
| `miners` | Core miner registry (MAC, IP, model, firmware) |
| `miner_network` | Network configuration (DHCP, gateway, DNS) |
| `miner_hardware` | Hardware specs (chains, chips, voltage limits) |
| `miner_status` | Current state (running/stopped/failure, uptime) |
| `miner_summary` | Performance snapshot (hashrate, power, temps) |
| `miner_chains` | Per-chain data (hashrate, temp, errors) |
| `miner_pools` | Pool configurations and share stats |
| `miner_fans` | Fan RPM and status |
| `miner_metrics` | Time-series data for charts |
| `autotune_presets` | VNish autotune profiles |
| `miner_log_sessions` | Boot cycle tracking |
| `miner_logs` | Collected log entries |

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

### Dashboard (`cmd/harvest-dashboard/`)

Web dashboard with:

- **Routes:**
  - `GET /` - Dashboard index (miner list with stats)
  - `GET /miner/{id}` - Miner detail page
  - `GET /api/miners` - JSON miner list
  - `GET /api/miner/{id}` - JSON miner details
  - `GET /api/miner/{id}/metrics` - JSON metrics
  - `GET /api/miner/{id}/logs` - JSON logs
  - `GET /api/sse/dashboard` - SSE live dashboard updates
  - `GET /api/sse/miner/{id}` - SSE live miner updates

- **SSE Live Updates:**
  - Dashboard: 10-second interval (online/offline counts, total hashrate/power, per-miner stats)
  - Miner detail: 5-second interval (status, summary, chains)
  - Auto-reconnect with exponential backoff
  - Visual indicators (pulsing green dot when connected)

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
- **Log sessions**: Boot cycles tracked by uptime changes

## API Reference

- VNish: See `pkg/vnish/vnish-api.md`
- Stock: See `pkg/stock/stock-api.md`
