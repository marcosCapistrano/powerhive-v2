# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PowerHive is a Go application for managing Bitcoin mining hardware (Antminer S19). It discovers miners on the network running either VNish or Stock Bitmain firmware, gathers power consumption/generation data, and adjusts miner power settings to match available power generation.

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

## Architecture

### Package Structure

```
powerhive-v2/
├── cmd/powerhive/         # CLI entry point
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

- `HTTPClient` - Stock firmware API client
- `Prober` - Detects stock firmware (implements `miner.FirmwareProber`)
- `DigestAuth` - HTTP Digest authentication (default: root:root)

**Endpoints:** `/cgi-bin/get_system_info.cgi`, `/cgi-bin/get_miner_status.cgi`

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
- **Chains**: Mining boards with ASIC chips (typically 3 per S19)
- **Hashrate**: TH/s (terahashes per second)
- **Power efficiency**: J/TH (joules per terahash)

## API Reference

- VNish: See `vnish-api.md`
- Stock: CGI endpoints at `/cgi-bin/*.cgi` (get_system_info, get_miner_status, etc.)
