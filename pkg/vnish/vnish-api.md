# VNish API Documentation

Complete API reference for VNish firmware (Antminer S19).

## Table of Contents

- [Authentication](#authentication)
- [Info & Status Endpoints](#info--status-endpoints)
- [Chain Endpoints](#chain-endpoints)
- [Autotune Endpoints](#autotune-endpoints)
- [Logs Endpoints](#logs-endpoints)
- [Metrics Endpoint](#metrics-endpoint)
- [Notes Endpoints (CRUD)](#notes-endpoints-crud)
- [API Keys Management](#api-keys-management)
- [Settings Endpoints](#settings-endpoints)
- [Mining Control Endpoints](#mining-control-endpoints)
- [Find Miner Endpoint](#find-miner-endpoint)
- [System Operations](#system-operations)

---

## Authentication

### Unlock (Get Bearer Token)

**Endpoint:** `POST /api/v1/unlock`

**Description:** Authenticate and receive a bearer token for subsequent API requests.

**Headers:**
- `accept: application/json`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "pw": "admin"
}
```

**Response:**
```json
{
  "token": "OwpWakIHG7ZE7xg7MGd6uL6UWOMeIzgW"
}
```

**Usage:** The token returned should be used in the `Authorization: Bearer <token>` header for protected endpoints.

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/unlock' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{"pw": "admin"}'
```

---

## Info & Status Endpoints

### Get Miner Info

**Endpoint:** `GET /api/v1/info`

**Description:** Returns detailed information about the miner including firmware version, platform, system info, and network status.

**Headers:**
- `accept: application/json`

**Response:**
```json
{
  "miner": "Antminer S19",
  "model": "s19",
  "fw_name": "Vnish",
  "fw_version": "1.2.6",
  "build_uuid": "5b50805f-a372-4f07-bbc2-5020aa03321a",
  "build_name": "vnishnet",
  "platform": "xil",
  "install_type": "nand",
  "build_time": "2025-05-29 19:22:26+00:00",
  "algorithm": "sha256d",
  "hr_measure": "GH/s",
  "system": {
    "os": "GNU/Linux",
    "miner_name": "s19",
    "file_system_version": "",
    "mem_total": 233712,
    "mem_free": 173956,
    "mem_free_percent": 74,
    "mem_buf": 31928,
    "mem_buf_percent": 13,
    "network_status": {
      "mac": "FE:07:26:3C:BE:1B",
      "dhcp": true,
      "ip": "192.168.1.2",
      "netmask": "255.255.255.0",
      "gateway": "192.168.1.1",
      "dns": ["192.168.1.1"],
      "hostname": "antminer"
    },
    "uptime": "0:06"
  },
  "serial": "N/A"
}
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/info' \
  -H 'accept: application/json'
```

---

### Get Miner Model Info

**Endpoint:** `GET /api/v1/model`

**Description:** Returns model-specific information including chip topology, cooling specifications, and overclocking parameters.

**Headers:**
- `accept: application/json`

**Response:**
```json
{
  "full_name": "Antminer S19",
  "model": "s19",
  "algorithm": "sha256d",
  "series": "x19",
  "platform": "xil",
  "install_type": "nand",
  "hr_measure": "GH/s",
  "serial": "N/A",
  "chain": {
    "chips_per_chain": 76,
    "chips_per_domain": 2,
    "num_chains": 3,
    "topology": {
      "chips": [
        [16, 17, 18, 19, 56, 57, 58, 59],
        [15, 14, 21, 20, 55, 54, 61, 60],
        [12, 13, 22, 23, 52, 53, 62, 63],
        [11, 10, 25, 24, 51, 50, 65, 64],
        [8, 9, 26, 27, 48, 49, 66, 67],
        [7, 6, 29, 28, 47, 46, 69, 68],
        [4, 5, 30, 31, 44, 45, 70, 71],
        [3, 2, 33, 32, 43, 42, 73, 72],
        [0, 1, 34, 35, 40, 41, 74, 75],
        [-1, -1, 37, 36, 39, 38, -1, -1]
      ],
      "num_cols": 8,
      "num_rows": 10
    }
  },
  "cooling": {
    "min_fan_pwm": 10,
    "min_target_temp": 20,
    "max_target_temp": 90,
    "fan_min_count": {
      "min": 1,
      "max": 4,
      "default": 4
    }
  },
  "overclock": {
    "max_voltage": 1535,
    "min_voltage": 1200,
    "default_voltage": 1380,
    "max_freq": 1000,
    "min_freq": 50,
    "default_freq": 675,
    "warn_freq": 850,
    "max_voltage_stock_psu": 1500
  }
}
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/model' \
  -H 'accept: application/json'
```

---

### Get Status

**Endpoint:** `GET /api/v1/status`

**Description:** Returns current miner operational status.

**Headers:**
- `accept: application/json`

**Response:**
```json
{
  "miner_state": "failure",
  "miner_state_time": 1,
  "description": "Failed to parse miner configuration",
  "failure_code": 1002,
  "find_miner": false,
  "restart_required": false,
  "reboot_required": false,
  "unlocked": false
}
```

**Fields:**
- `miner_state`: Current state (e.g., "failure", "running", "stopped")
- `miner_state_time`: Time in current state
- `description`: Human-readable status description
- `failure_code`: Error code if in failure state
- `find_miner`: LED blink status
- `restart_required`: Whether restart is needed
- `reboot_required`: Whether reboot is needed
- `unlocked`: Whether unlocked for modifications

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/status' \
  -H 'accept: application/json'
```

---

### Get Summary

**Endpoint:** `GET /api/v1/summary`

**Description:** Returns comprehensive mining summary including hashrate, temperatures, power consumption, pool information, and chain details.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`

**Response:**
```json
{
  "miner": {
    "miner_status": {
      "miner_state": "failure",
      "miner_state_time": 1,
      "description": "Failed to parse miner configuration",
      "failure_code": 1002
    },
    "miner_type": "Antminer S19 (Vnish 1.2.6)",
    "hr_stock": 0.0,
    "average_hashrate": 0.0,
    "instant_hashrate": 0.0,
    "hr_realtime": 0.0,
    "hr_nominal": 0.0,
    "hr_average": 0.0,
    "pcb_temp": {
      "min": 0,
      "max": 0
    },
    "chip_temp": {
      "min": 0,
      "max": 0
    },
    "power_consumption": 0,
    "power_usage": 0,
    "power_efficiency": 0.0,
    "hw_errors_percent": 0.0,
    "hr_error": 0.0,
    "hw_errors": 0,
    "devfee_percent": 0.0,
    "devfee": 0.0,
    "pools": [
      {
        "id": 0,
        "url": "DevFee",
        "pool_type": "DevFee",
        "user": "DevFee",
        "status": "working",
        "asic_boost": false,
        "diff": "",
        "accepted": 0,
        "rejected": 0,
        "stale": 0,
        "ls_diff": 0.0,
        "ls_time": "0",
        "diffa": 0.0,
        "ping": 0
      }
    ],
    "cooling": {
      "fan_num": 0,
      "fans": [],
      "settings": {
        "mode": {
          "name": "auto"
        }
      },
      "fan_duty": 0
    },
    "chains": [],
    "found_blocks": 0,
    "best_share": 0
  }
}
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/summary' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>'
```

---

### Get Performance Summary

**Endpoint:** `GET /api/v1/perf-summary`

**Description:** Returns current autotune preset configuration and preset switcher settings.

**Headers:**
- `accept: application/json`

**Response:**
```json
{
  "current_preset": {
    "name": "disabled",
    "pretty": "Disabled",
    "status": "untuned",
    "modded_psu_required": false,
    "globals": {
      "volt": 1380,
      "freq": 675
    }
  },
  "preset_switcher": {
    "enabled": false,
    "top_preset": null,
    "decrease_temp": 75,
    "rise_temp": 55,
    "check_time": 300,
    "autochange_top_preset": false,
    "ignore_fan_speed": false,
    "min_preset": null,
    "power_delta": 0
  }
}
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/perf-summary' \
  -H 'accept: application/json'
```

---

## Chain Endpoints

### Get Miner Chains

**Endpoint:** `GET /api/v1/chains`

**Description:** Returns list of mining chains (boards) and their chip configurations.

**Headers:**
- `accept: application/json`

**Response:**
```json
[]
```

**Note:** Returns empty array when no chains are detected or miner is in failure state. When chains are present, each chain object contains chip-level details, frequencies, voltages, and temperatures.

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/chains' \
  -H 'accept: application/json'
```

---

### Get Chains Factory Info

**Endpoint:** `GET /api/v1/chains/factory-info`

**Description:** Returns factory information for mining chains including stock hashrate, PSU details, and chain-specific data.

**Headers:**
- `accept: application/json`

**Response:**
```json
{
  "hr_stock": null,
  "has_pics": null,
  "psu_model": null,
  "psu_serial": null,
  "chains": null
}
```

**Note:** Returns null values when factory info is not available or not configured.

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/chains/factory-info' \
  -H 'accept: application/json'
```

---

## Autotune Endpoints

### Get Autotune Preset List

**Endpoint:** `GET /api/v1/autotune/presets`

**Description:** Returns list of available autotune presets with power/hashrate profiles.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`

**Response:**
```json
[
  {
    "name": "disabled",
    "pretty": "Disabled",
    "status": "untuned",
    "modded_psu_required": false,
    "tune_settings": {
      "hashrate": 0,
      "volt": 13800,
      "freq": 675,
      "chains": [
        {
          "freq": 0,
          "chips": [0, 0, 0, ...]
        }
      ],
      "modified": true
    }
  },
  {
    "name": "1100",
    "pretty": "1100 watt ~ 53 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "1300",
    "pretty": "1300 watt ~ 58 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "1500",
    "pretty": "1500 watt ~ 63 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "1700",
    "pretty": "1700 watt ~ 68 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "1900",
    "pretty": "1900 watt ~ 73 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "2100",
    "pretty": "2100 watt ~ 78 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "2350",
    "pretty": "2350 watt ~ 85 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "2620",
    "pretty": "2620 watt ~ 91 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "2950",
    "pretty": "2950 watt ~ 99 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "3200",
    "pretty": "3200 watt ~ 102 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "3350",
    "pretty": "3350 watt ~ 105 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "3750",
    "pretty": "3750 watt ~ 110 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "4000",
    "pretty": "4000 watt ~ 115 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "4260",
    "pretty": "4260 watt ~ 120 TH",
    "status": "untuned",
    "modded_psu_required": false
  },
  {
    "name": "4690",
    "pretty": "4690 watt ~ 124 TH",
    "status": "untuned",
    "modded_psu_required": false
  }
]
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/autotune/presets' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>'
```

---

## Logs Endpoints

### Get Status Logs

**Endpoint:** `GET /api/v1/logs/status`

**Description:** Returns miner status log entries showing initialization, state changes, and operational events.

**Headers:**
- `accept: */*`

**Response Sample:**
```
[2025/10/14 20:38:59] INFO: Initializing [Antminer S19 Xilinx (1.2.6)]
[2025/10/14 20:39:11] INFO: Initializing [Antminer S19 Xilinx (1.2.6)]
[2025/10/14 20:39:29] INFO: Cooling down the miner
[2025/10/14 20:39:29] INFO: Cooling down completed
[2025/10/14 20:39:38] INFO: Restarting (1 of 3) - Miner initialization failed
[2025/10/14 20:39:38] INFO: Mining stopped
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/logs/status' \
  -H 'accept: */*'
```

---

### Get Miner Logs

**Endpoint:** `GET /api/v1/logs/miner`

**Description:** Returns detailed miner hardware logs including fan checks, chain detection, and FPGA operations.

**Headers:**
- `accept: */*`

**Response Sample:**
```
[1970/01/01 00:00:09] INFO: Detected 256 Mb of RAM
[1970/01/01 00:00:11] INFO: Set HW version to 0x4000b031
[1970/01/01 00:00:11] INFO: Starting FPGA queue
[1970/01/01 00:00:11] INFO: Initializing PSU
[1970/01/01 00:00:13] INFO: PSU model: 0x74
[1970/01/01 00:00:13] INFO: Switching to manual fan control (30%)
[2025/10/14 20:38:59] INFO: fan#1 - ok
[2025/10/14 20:38:59] INFO: fan#2 - ok
[2025/10/14 20:38:59] INFO: chain#1 - connected
[2025/10/14 20:38:59] INFO: chain#2 - connected
[2025/10/14 20:38:59] INFO: chain#3 - connected
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/logs/miner' \
  -H 'accept: */*'
```

---

### Get Autotune Logs

**Endpoint:** `GET /api/v1/logs/autotune`

**Description:** Returns autotune-specific logs (if autotune has been run).

**Headers:**
- `accept: */*`

**Response:**
Empty or autotune operation logs.

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/logs/autotune' \
  -H 'accept: */*'
```

---

### Get System Logs

**Endpoint:** `GET /api/v1/logs/system`

**Description:** Returns system boot logs and kernel messages.

**Headers:**
- `accept: */*`

**Response Sample:**
```
Booting Linux on physical CPU 0x0
Linux version 4.6.0-xilinx-g03c746f7
CPU: ARMv7 Processor [413fc090] revision 0 (ARMv7), cr=18c5387d
Memory: 200692K/245760K available
Virtual kernel memory layout:
    vector  : 0xffff0000 - 0xffff1000
    fixmap  : 0xffc00000 - 0xfff00000
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/logs/system' \
  -H 'accept: */*'
```

---

### Get Messages Logs

**Endpoint:** `GET /api/v1/logs/messages`

**Description:** Returns system messages including network events, authentication attempts, and service logs.

**Headers:**
- `accept: */*`

**Response Sample:**
```
Jan  1 00:00:02 buildroot syslog.info syslogd started: BusyBox v1.36.1
Oct 14 20:38:56 buildroot daemon.warn chronyd[799]: System clock was stepped by 1760474322.018289 seconds
Oct 14 20:39:18 buildroot authpriv.info dropbear[924]: Child connection from 192.168.1.26:34122
Oct 14 20:39:18 buildroot authpriv.warn dropbear[924]: Bad password attempt for 'miner' from 192.168.1.26:34122
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/logs/messages' \
  -H 'accept: */*'
```

---

### Get API Logs

**Endpoint:** `GET /api/v1/logs/api`

**Description:** Returns API access logs with timestamps, endpoints, status codes, and response times.

**Headers:**
- `accept: */*`

**Response Sample:**
```
[14 Oct 20:38:57] 192.168.1.3:46182 "POST /cgi-bin/blink.cgi" 405 "-" 0.034ms
[14 Oct 20:40:27] 192.168.1.26:55002 "POST /api/v1/settings" 200 "http://192.168.1.60/mining/pools" 13.613ms
[14 Oct 20:43:34] 192.168.1.3:35134 "POST /api/v1/find-miner" 200 "-" 0.82ms
[18 Nov 15:53:30] 192.168.1.38:35388 "POST /api/v1/notes" 200 "http://192.168.1.7/docs/" 6.17ms
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/logs/api' \
  -H 'accept: */*'
```

---

## Metrics Endpoint

### Get Metrics

**Endpoint:** `GET /api/v1/metrics`

**Description:** Returns time-series metrics data including hashrate, temperatures, fan duty, and power consumption.

**Headers:**
- `accept: application/json`

**Parameters:**
- `time_slice` (optional): Amount of seconds until now. Max is 3 days (259200). Default is 1 day (86400)
- `step` (optional): Resample step in seconds for averaging. Default is 15 min (900)

**Response:**
```json
{
  "timezone": "GMT",
  "metrics": [
    {
      "time": 1763482448,
      "data": {
        "hashrate": 0.0,
        "pcb_max_temp": 10,
        "chip_max_temp": 15,
        "fan_duty": 1,
        "power_consumption": 0
      }
    },
    {
      "time": 1763478000,
      "data": {
        "hashrate": 0.0,
        "pcb_max_temp": 0,
        "chip_max_temp": 0,
        "fan_duty": 0,
        "power_consumption": 0
      }
    }
  ],
  "annotations": [
    {
      "time": 1763481139,
      "data": {
        "type": "stop"
      }
    },
    {
      "time": 1763481119,
      "data": {
        "type": "restart"
      }
    }
  ]
}
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/metrics?time_slice=86400&step=900' \
  -H 'accept: application/json'
```

---

## Notes Endpoints (CRUD)

### Read All Notes

**Endpoint:** `GET /api/v1/notes`

**Description:** Returns all stored notes as key-value pairs.

**Headers:**
- `accept: application/json`

**Response:**
```json
{}
```

**Note:** Returns empty object when no notes exist.

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/notes' \
  -H 'accept: application/json'
```

---

### Add Note

**Endpoint:** `POST /api/v1/notes`

**Description:** Creates a new note with specified key and value.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "value": "test note value",
  "key": "test_note"
}
```

**Response:**
No output on success (HTTP 200)

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/notes' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"value": "test note value", "key": "test_note"}'
```

---

### Get One Note

**Endpoint:** `GET /api/v1/notes/{key}`

**Description:** Retrieves a specific note by its key.

**Headers:**
- `accept: application/json`

**Path Parameters:**
- `key`: The note key to retrieve

**Response:**
```json
{
  "value": "test note value"
}
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/notes/test_note' \
  -H 'accept: application/json'
```

---

### Update Note

**Endpoint:** `PUT /api/v1/notes/{key}`

**Description:** Updates an existing note's value.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Path Parameters:**
- `key`: The note key to update

**Request Body:**
```json
{
  "value": "updated test note"
}
```

**Response:**
No output on success (HTTP 200)

**Example:**
```bash
curl -X 'PUT' \
  'http://192.168.1.2/api/v1/notes/test_note' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"value": "updated test note"}'
```

---

### Delete Note

**Endpoint:** `DELETE /api/v1/notes/{key}`

**Description:** Deletes a note by its key.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`

**Path Parameters:**
- `key`: The note key to delete

**Response:**
No output on success (HTTP 200)

**Example:**
```bash
curl -X 'DELETE' \
  'http://192.168.1.2/api/v1/notes/test_note' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>'
```

---

## API Keys Management

### Get API Keys

**Endpoint:** `GET /api/v1/apikeys`

**Description:** Returns list of configured API keys.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`

**Response:**
```json
[]
```

or when keys exist:

```json
[
  {
    "key": "stringstringstringstringstringst",
    "description": "Test API Key"
  }
]
```

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/apikeys' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>'
```

---

### Add API Key

**Endpoint:** `POST /api/v1/apikeys`

**Description:** Creates a new API key.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`
- `x-api-key: <32-character-key>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "description": "Test API Key",
  "key": "stringstringstringstringstringst"
}
```

**Response:**
```json
{
  "status": "inserted"
}
```

**Note:** API key must be exactly 32 characters long.

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/apikeys' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: stringstringstringstringstringst' \
  -H 'Content-Type: application/json' \
  -d '{"description": "Test API Key", "key": "stringstringstringstringstringst"}'
```

---

### Delete API Key

**Endpoint:** `POST /api/v1/apikeys/delete`

**Description:** Deletes an existing API key.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`
- `x-api-key: <32-character-key>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "key": "stringstringstringstringstringst"
}
```

**Response:**
No output on success (HTTP 200)

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/apikeys/delete' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: stringstringstringstringstringst' \
  -H 'Content-Type: application/json' \
  -d '{"key": "stringstringstringstringstringst"}'
```

---

## Settings Endpoints

### Get All Miner Settings

**Endpoint:** `GET /api/v1/settings`

**Description:** Returns complete miner configuration settings.

**Headers:**
- `accept: application/json`
- `x-api-key: <api-key>` (optional)

**Response:**
No output (empty response observed)

**Example:**
```bash
curl -X 'GET' \
  'http://192.168.1.2/api/v1/settings' \
  -H 'accept: application/json'
```

---

### Save Miner Settings

**Endpoint:** `POST /api/v1/settings`

**Description:** Updates miner configuration settings. Can update specific nested settings.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`
- `x-api-key: <api-key>`
- `Content-Type: application/json`

**Request Body Example (changing quiet mode):**
```json
{
  "miner": {
    "misc": {
      "quiet_mode": true
    }
  }
}
```

**Response:**
JSON confirmation (structure varies based on what was updated)

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/settings' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: <api-key>' \
  -H 'Content-Type: application/json' \
  -d '{"miner": {"misc": {"quiet_mode": true}}}'
```

---

### Settings Backup

**Endpoint:** `POST /api/v1/settings/backup`

**Description:** Creates and downloads a backup of all miner settings as a compressed tar.gz file.

**Headers:**
- `accept: application/octet-stream`
- `Authorization: Bearer <token>`
- `x-api-key: <api-key>`

**Response:**
Binary data (gzip compressed tar archive, ~860 bytes)

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/settings/backup' \
  -H 'accept: application/octet-stream' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: <api-key>' \
  -d '' \
  --output backup.tar.gz
```

---

### Settings Factory Reset

**Endpoint:** `POST /api/v1/settings/factory-reset`

**Description:** Resets all settings to factory defaults.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`
- `x-api-key: <api-key>`

**Response:**
JSON confirmation

**Warning:** This operation will erase all custom configurations.

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/settings/factory-reset' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: <api-key>' \
  -d ''
```

---

### Settings Restore

**Endpoint:** `POST /api/v1/settings/restore`

**Description:** Restores settings from a backup file.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`
- `x-api-key: <api-key>`
- `Content-Type: multipart/form-data`

**Request Body:**
Multipart form data with backup file

**Response:**
JSON confirmation

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/settings/restore' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: <api-key>' \
  -H 'Content-Type: multipart/form-data' \
  -F 'file=@backup.tar.gz;type=application/gzip'
```

---

## Mining Control Endpoints

### Mining Pause

**Endpoint:** `POST /api/v1/mining/pause`

**Description:** Pauses mining operations.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`

**Response:**
No output on success, or error message:
```json
{
  "err": "CGMiner err: Cgminer error failed-to-activate-pause-mode "
}
```

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/mining/pause' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -d ''
```

---

### Mining Resume

**Endpoint:** `POST /api/v1/mining/resume`

**Description:** Resumes paused mining operations.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`

**Response:**
No output on success, or error message:
```json
{
  "err": "CGMiner err: Cgminer error failed-to-activate-resume-mode "
}
```

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/mining/resume' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -d ''
```

---

### Mining Start

**Endpoint:** `POST /api/v1/mining/start`

**Description:** Starts mining operations.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`

**Response:**
No output on success, or error message:
```json
{
  "err": "CGMiner err: Cgminer error failed-to-activate-resume-mode "
}
```

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/mining/start' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -d ''
```

---

### Mining Stop

**Endpoint:** `POST /api/v1/mining/stop`

**Description:** Stops mining operations.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`

**Response:**
No output on success, or error message:
```json
{
  "err": "CGMiner err: Cgminer error failed-to-activate-pause-mode "
}
```

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/mining/stop' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -d ''
```

---

### Mining Restart

**Endpoint:** `POST /api/v1/mining/restart`

**Description:** Restarts mining operations.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`

**Response:**
No output on success (HTTP 200)

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/mining/restart' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -d ''
```

---

### Mining Switch Pool

**Endpoint:** `POST /api/v1/mining/switch-pool`

**Description:** Switches to a different mining pool by pool ID.

**Headers:**
- `accept: */*`
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "pool_id": 1073741824
}
```

**Response:**
No output on success, or error message:
```json
{
  "err": "Could not find pool"
}
```

**Note:** Pool ID must be a valid configured pool identifier.

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/mining/switch-pool' \
  -H 'accept: */*' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"pool_id": 1073741824}'
```

---

## Find Miner Endpoint

### Find Miner (Toggle LED)

**Endpoint:** `POST /api/v1/find-miner`

**Description:** Toggles the miner's LED to help physically locate the device.

**Headers:**
- `accept: application/json`

**Response:**
```json
{
  "on": true
}
```

or

```json
{
  "on": false
}
```

**Note:** Each call toggles the LED state.

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/find-miner' \
  -H 'accept: application/json' \
  -d ''
```

---

## System Operations

### System Reboot

**Endpoint:** `POST /api/v1/system/reboot`

**Description:** Reboots the miner system.

**Headers:**
- `accept: application/json`
- `Authorization: Bearer <token>`
- `x-api-key: <api-key>`

**Response:**
JSON confirmation

**Warning:** This will restart the entire system.

**Example:**
```bash
curl -X 'POST' \
  'http://192.168.1.2/api/v1/system/reboot' \
  -H 'accept: application/json' \
  -H 'Authorization: Bearer <token>' \
  -H 'x-api-key: <api-key>' \
  -d ''
```

---

## Notes

### Authentication
- Most modification endpoints require a Bearer token obtained from `/api/v1/unlock`
- Sensitive operations also require an API key in the `x-api-key` header
- API keys must be exactly 32 characters long

### Error Handling
- The API returns appropriate HTTP status codes
- Error responses often include an `err` field with details
- Common errors: 403 (Forbidden), 404 (Not Found), 409 (Conflict), 500 (Internal Server Error)

### Response Formats
- Most endpoints return JSON
- Logs endpoints return plain text
- Backup endpoint returns binary data (gzip)

### Best Practices
- Always unlock first to get a bearer token
- Store the token for subsequent requests
- Use API keys for sensitive operations
- Test operations on a non-production miner first
- Create backups before making configuration changes