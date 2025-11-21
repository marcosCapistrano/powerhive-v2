# Stock Bitmain Firmware API Documentation

Complete API reference for stock Bitmain firmware (Antminer series including KS5 Pro, S19, etc.).

## Table of Contents

- [Authentication](#authentication)
- [System Information Endpoints](#system-information-endpoints)
- [Mining Status Endpoints](#mining-status-endpoints)
- [Configuration Endpoints](#configuration-endpoints)
- [Control Endpoints](#control-endpoints)
- [Log Endpoints](#log-endpoints)
- [Error Codes](#error-codes)

---

## Authentication

Stock Bitmain firmware uses HTTP Digest Authentication for all API endpoints.

**Default Credentials:**
- Username: `root`
- Password: `root`

**Realm:** `antMiner Configuration`

All requests must include Digest authentication. With curl, use the `--digest -u username:password` flags.

**Example:**
```bash
curl --digest -u root:root 'http://192.168.1.27/cgi-bin/get_system_info.cgi'
```

---

## System Information Endpoints

### Get System Info

**Endpoint:** `GET /cgi-bin/get_system_info.cgi`

**Description:** Returns detailed system information including miner type, network configuration, firmware version, and hardware serial number.

**Response:**
```json
{
  "minertype": "Antminer KS5 Pro",
  "nettype": "DHCP",
  "netdevice": "eth0",
  "macaddr": "30:CB:1B:49:A4:DB",
  "hostname": "Antminer",
  "ipaddress": "192.168.1.27",
  "netmask": "255.255.255.0",
  "gateway": "",
  "dnsservers": "",
  "system_mode": "GNU/Linux",
  "system_kernel_version": "Linux 4.9.38-tag- #1 SMP PREEMPT Wed Aug 28 11:28:30 CST 2024",
  "system_filesystem_version": "Wed Aug 28 11:12:11 CST 2024",
  "firmware_type": "Release",
  "Algorithm": "KHeavyHash",
  "serinum": "JYZZF1UBDJHJE01W0"
}
```

**Fields:**
- `minertype`: Hardware model (e.g., "Antminer KS5 Pro", "Antminer S19")
- `nettype`: Network configuration type ("DHCP" or "Static")
- `netdevice`: Network interface name
- `macaddr`: MAC address
- `hostname`: Device hostname
- `ipaddress`: Current IP address
- `netmask`: Network mask
- `gateway`: Default gateway
- `dnsservers`: DNS server addresses
- `system_mode`: Operating system
- `system_kernel_version`: Linux kernel version
- `system_filesystem_version`: Firmware build date
- `firmware_type`: Firmware type ("Release")
- `Algorithm`: Mining algorithm (e.g., "KHeavyHash" for Kaspa, "sha256d" for Bitcoin)
- `serinum`: Hardware serial number

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/get_system_info.cgi'
```

---

### Get Network Info

**Endpoint:** `GET /cgi-bin/get_network_info.cgi`

**Description:** Returns current and configured network settings.

**Response:**
```json
{
  "nettype": "DHCP",
  "netdevice": "eth0",
  "macaddr": "30:CB:1B:49:A4:DB",
  "ipaddress": "192.168.1.27",
  "netmask": "255.255.255.0",
  "conf_nettype": "DHCP",
  "conf_hostname": "Antminer",
  "conf_ipaddress": "",
  "conf_netmask": "",
  "conf_gateway": "",
  "conf_dnsservers": ""
}
```

**Fields:**
- `nettype`: Current network type
- `netdevice`: Network interface
- `macaddr`: MAC address
- `ipaddress`: Current IP address
- `netmask`: Current network mask
- `conf_nettype`: Configured network type
- `conf_hostname`: Configured hostname
- `conf_ipaddress`: Configured static IP (empty if DHCP)
- `conf_netmask`: Configured network mask
- `conf_gateway`: Configured gateway
- `conf_dnsservers`: Configured DNS servers

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/get_network_info.cgi'
```

---

## Mining Status Endpoints

### Get Summary (KS5/Newer Models)

**Endpoint:** `GET /cgi-bin/summary.cgi`

**Description:** Returns mining summary including hashrate and system status alerts. This endpoint is available on newer models like KS5 Pro.

**Response:**
```json
{
  "STATUS": {
    "STATUS": "S",
    "when": 1763763762,
    "Msg": "summary",
    "api_version": "1.0.0"
  },
  "INFO": {
    "miner_version": "0.0-2.0.0",
    "CompileTime": "Wed Aug 28 11:12:11 CST 2024",
    "type": "Antminer KS5"
  },
  "SUMMARY": [
    {
      "elapsed": 0,
      "rate_5s": 0.0,
      "rate_avg": 0.0,
      "rate_30m": 0.0,
      "rate_ideal": 0.0,
      "rate_unit": "H/s",
      "hw_all": 0,
      "bestshare": 0,
      "status": [
        {
          "type": "rate",
          "status": "s",
          "code": 0,
          "msg": ""
        },
        {
          "type": "network",
          "status": "s",
          "code": 0,
          "msg": ""
        },
        {
          "type": "fans",
          "status": "s",
          "code": 0,
          "msg": ""
        },
        {
          "type": "temp",
          "status": "s",
          "code": 0,
          "msg": ""
        }
      ]
    }
  ]
}
```

**STATUS Fields:**
- `STATUS`: Response status ("S" = success)
- `when`: Unix timestamp
- `Msg`: Response message type
- `api_version`: API version

**INFO Fields:**
- `miner_version`: CGMiner version
- `CompileTime`: Firmware compile time
- `type`: Miner type

**SUMMARY Fields:**
- `elapsed`: Uptime in seconds
- `rate_5s`: 5-second average hashrate
- `rate_avg`: Average hashrate
- `rate_30m`: 30-minute average hashrate
- `rate_ideal`: Expected/ideal hashrate
- `rate_unit`: Hashrate unit (e.g., "H/s", "TH/s")
- `hw_all`: Total hardware errors
- `bestshare`: Best share difficulty found
- `status`: Array of system status checks

**Status Item Fields:**
- `type`: Check type ("rate", "network", "fans", "temp")
- `status`: Status code ("s" = success, "e" = error, "w" = warning)
- `code`: Error code (0 = no error)
- `msg`: Error/warning message

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/summary.cgi'
```

---

### Get Stats (KS5/Newer Models)

**Endpoint:** `GET /cgi-bin/stats.cgi`

**Description:** Returns detailed mining statistics including per-chain information, temperatures, and fan data.

**Response:**
```json
{
  "STATUS": {
    "STATUS": "S",
    "when": 1763763763,
    "Msg": "stats",
    "api_version": "1.0.0"
  },
  "INFO": {
    "miner_version": "0.0-2.0.0",
    "CompileTime": "Wed Aug 28 11:12:11 CST 2024",
    "type": "Antminer KS5"
  },
  "STATS": [
    {
      "elapsed": 0,
      "rate_5s": 0.0,
      "rate_avg": 0.0,
      "rate_30m": 0.0,
      "rate_ideal": 0.0,
      "rate_unit": "H/s",
      "chain_num": 3,
      "fan_num": 0,
      "fan": [],
      "hwp_total": 0.0,
      "chain": [
        {
          "index": 0,
          "freq_avg": 0,
          "rate_ideal": 0.0,
          "rate_real": 0.0,
          "asic_num": 0,
          "temp_pic": [],
          "temp_pcb": [],
          "temp_chip": [],
          "hw": 0,
          "eeprom_loaded": true,
          "sn": "JYZZYRRBDJHJE023P"
        },
        {
          "index": 1,
          "freq_avg": 0,
          "rate_ideal": 0.0,
          "rate_real": 0.0,
          "asic_num": 0,
          "temp_pic": [],
          "temp_pcb": [],
          "temp_chip": [],
          "hw": 0,
          "eeprom_loaded": true,
          "sn": "JYZZYRRBDJHJE033G"
        },
        {
          "index": 2,
          "freq_avg": 0,
          "rate_ideal": 0.0,
          "rate_real": 0.0,
          "asic_num": 0,
          "temp_pic": [],
          "temp_pcb": [],
          "temp_chip": [],
          "hw": 0,
          "eeprom_loaded": true,
          "sn": "JYZZYRRBDJHJE047H"
        }
      ]
    }
  ]
}
```

**STATS Fields:**
- `elapsed`: Uptime in seconds
- `rate_5s`, `rate_avg`, `rate_30m`, `rate_ideal`: Hashrate metrics
- `rate_unit`: Hashrate unit
- `chain_num`: Number of hash boards
- `fan_num`: Number of fans detected
- `fan`: Array of fan RPM values
- `hwp_total`: Total hardware error percentage
- `chain`: Array of chain (hash board) details

**Chain Fields:**
- `index`: Chain index (0, 1, 2)
- `freq_avg`: Average frequency (MHz)
- `rate_ideal`: Expected hashrate for this chain
- `rate_real`: Actual hashrate for this chain
- `asic_num`: Number of ASIC chips detected
- `temp_pic`: PIC controller temperatures
- `temp_pcb`: PCB temperatures
- `temp_chip`: Chip temperatures
- `hw`: Hardware errors on this chain
- `eeprom_loaded`: EEPROM status
- `sn`: Chain serial number

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/stats.cgi'
```

---

### Get Pools Status

**Endpoint:** `GET /cgi-bin/pools.cgi`

**Description:** Returns status of configured mining pools.

**Response:**
```json
{
  "STATUS": {
    "STATUS": "S",
    "when": 1763763764,
    "Msg": "pools",
    "api_version": "1.0.0"
  },
  "INFO": {
    "miner_version": "0.0-2.0.0",
    "CompileTime": "Wed Aug 28 11:12:11 CST 2024",
    "type": "Antminer KS5"
  },
  "POOLS": [
    {
      "index": 0,
      "url": "stratum+tcp://mining.viabtc.io:3015",
      "user": "Minerdanoite.0x102",
      "status": "Alive",
      "priority": 0,
      "getworks": 0,
      "accepted": 0,
      "rejected": 0,
      "discarded": 0,
      "stale": 0,
      "diff": "2048.0000",
      "diff1": 0,
      "diffa": 0.0,
      "diffr": 0,
      "diffs": 0,
      "lsdiff": 0,
      "lstime": "0"
    },
    {
      "index": 1,
      "url": "stratum+tcp://mining.viabtc.io:3015",
      "user": "Minerdanoite.0x102",
      "status": "Alive",
      "priority": 1,
      "getworks": 0,
      "accepted": 0,
      "rejected": 0,
      "discarded": 0,
      "stale": 0,
      "diff": "2048.0000",
      "diff1": 0,
      "diffa": 0.0,
      "diffr": 0,
      "diffs": 0,
      "lsdiff": 0,
      "lstime": "0"
    }
  ]
}
```

**Pool Fields:**
- `index`: Pool index (0-2)
- `url`: Stratum URL
- `user`: Worker username
- `status`: Connection status ("Alive", "Dead")
- `priority`: Pool priority (lower = higher priority)
- `getworks`: Number of getwork requests
- `accepted`: Accepted shares
- `rejected`: Rejected shares
- `discarded`: Discarded shares
- `stale`: Stale shares
- `diff`: Current difficulty
- `diff1`: Difficulty 1 shares
- `diffa`: Accepted difficulty
- `diffr`: Rejected difficulty
- `diffs`: Stale difficulty
- `lsdiff`: Last share difficulty
- `lstime`: Last share time

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/pools.cgi'
```

---

### Get Miner Status (S19/Older Models)

**Endpoint:** `GET /cgi-bin/get_miner_status.cgi`

**Description:** Returns miner status for older models (S19, etc.). This endpoint may return 404 on newer models like KS5.

**Note:** For KS5 and newer models, use `summary.cgi` and `stats.cgi` instead.

**Response (S19 Example):**
```json
{
  "summary": {
    "elapsed": 12345,
    "ghs5s": 95000.0,
    "ghsav": 94500.0,
    "temp": 65
  },
  "devs": [...],
  "pools": [...]
}
```

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/get_miner_status.cgi'
```

---

## Configuration Endpoints

### Get Miner Configuration

**Endpoint:** `GET /cgi-bin/get_miner_conf.cgi`

**Description:** Returns current miner configuration including pool settings, frequency, voltage, and fan settings.

**Response:**
```json
{
  "pools": [
    {
      "url": "stratum+tcp://mining.viabtc.io:3015",
      "user": "Minerdanoite.0x102",
      "pass": "123"
    },
    {
      "url": "stratum+tcp://mining.viabtc.io:3015",
      "user": "Minerdanoite.0x102",
      "pass": "123"
    },
    {
      "url": "stratum+tcp://mining.viabtc.io:3015",
      "user": "Minerdanoite.0x102",
      "pass": "123"
    }
  ],
  "algo": "ks5_2382",
  "bitmain-fan-ctrl": false,
  "bitmain-fan-pwm": "100",
  "bitmain-freq": "500",
  "bitmain-voltage": "1300",
  "bitmain-work-mode": "0",
  "bitmain-freq-level": "100"
}
```

**Fields:**
- `pools`: Array of 3 pool configurations
  - `url`: Stratum pool URL
  - `user`: Worker username
  - `pass`: Worker password
- `algo`: Mining algorithm preset
- `bitmain-fan-ctrl`: Manual fan control enabled
- `bitmain-fan-pwm`: Fan PWM percentage (0-100)
- `bitmain-freq`: Mining frequency (MHz)
- `bitmain-voltage`: Voltage (mV)
- `bitmain-work-mode`: Work mode (0 = normal)
- `bitmain-freq-level`: Frequency level percentage

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/get_miner_conf.cgi'
```

---

### Set Miner Configuration

**Endpoint:** `POST /cgi-bin/set_miner_conf.cgi`

**Description:** Updates miner configuration. Accepts JSON or form-encoded data.

**Request Body (JSON):**
```json
{
  "pools": [
    {
      "url": "stratum+tcp://pool.example.com:3333",
      "user": "worker.1",
      "pass": "x"
    },
    {
      "url": "stratum+tcp://pool2.example.com:3333",
      "user": "worker.1",
      "pass": "x"
    },
    {
      "url": "",
      "user": "",
      "pass": ""
    }
  ],
  "bitmain-fan-ctrl": false,
  "bitmain-fan-pwm": "100",
  "bitmain-freq": "500",
  "bitmain-voltage": "1300"
}
```

**Response (Success):**
```json
{
  "stats": "success",
  "code": "M000",
  "msg": "OK!"
}
```

**Response (Error):**
```json
{
  "stats": "error",
  "code": "M001",
  "msg": "Invalid configuration"
}
```

**Example:**
```bash
curl --digest -u root:root \
  -H 'Content-Type: application/json' \
  -d '{"pools":[{"url":"stratum+tcp://pool.example.com:3333","user":"worker.1","pass":"x"}]}' \
  'http://192.168.1.27/cgi-bin/set_miner_conf.cgi'
```

---

### Set Network Configuration

**Endpoint:** `POST /cgi-bin/set_network_conf.cgi`

**Description:** Updates network configuration.

**Request Body (Form-encoded):**
```
conf_nettype=DHCP&conf_hostname=Antminer
```

Or for static IP:
```
conf_nettype=Static&conf_hostname=Antminer&conf_ipaddress=192.168.1.100&conf_netmask=255.255.255.0&conf_gateway=192.168.1.1&conf_dnsservers=8.8.8.8
```

**Response (Success):**
```json
{
  "stats": "success",
  "code": "N000",
  "msg": "OK!"
}
```

**Response (Error):**
```json
{
  "stats": "error",
  "code": "N001",
  "msg": "Hostname invalid!"
}
```

**Example:**
```bash
curl --digest -u root:root \
  -d 'conf_nettype=DHCP&conf_hostname=MyMiner' \
  'http://192.168.1.27/cgi-bin/set_network_conf.cgi'
```

---

## Control Endpoints

### Reboot System

**Endpoint:** `GET /cgi-bin/reboot.cgi`

**Description:** Reboots the miner system.

**Response:**
```
Status 200
```

**Warning:** This will restart the miner and temporarily stop mining.

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/reboot.cgi'
```

---

### Reset Configuration

**Endpoint:** `GET /cgi-bin/reset_conf.cgi`

**Description:** Resets miner configuration to factory defaults.

**Response:**
```
Status 200
```

**Warning:** This will erase all custom configurations including pool settings.

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/reset_conf.cgi'
```

---

### Get Blink Status

**Endpoint:** `GET /cgi-bin/get_blink_status.cgi`

**Description:** Returns LED blink status for miner identification.

**Response:**
```json
{
  "blink": false
}
```

**Fields:**
- `blink`: LED blink state (true = blinking, false = normal)

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/get_blink_status.cgi'
```

---

### Toggle LED Blink

**Endpoint:** `POST /cgi-bin/blink.cgi`

**Description:** Toggles LED blinking to help physically identify the miner.

**Request Body:**
```
blink=true
```
or
```
blink=false
```

**Response:**
```json
{
  "code": "B001"
}
```

**Example:**
```bash
# Turn on blinking
curl --digest -u root:root \
  -d 'blink=true' \
  'http://192.168.1.27/cgi-bin/blink.cgi'

# Turn off blinking
curl --digest -u root:root \
  -d 'blink=false' \
  'http://192.168.1.27/cgi-bin/blink.cgi'
```

---

### Firmware Upgrade

**Endpoint:** `POST /cgi-bin/upgrade.cgi`

**Description:** Uploads and installs firmware upgrade.

**Request:** Multipart form-data with firmware file.

**Response (Error without file):**
```json
{
  "stats": "error",
  "code": "U001",
  "msg": "6"
}
```

**Example:**
```bash
curl --digest -u root:root \
  -F 'file=@firmware.tar.gz' \
  'http://192.168.1.27/cgi-bin/upgrade.cgi'
```

---

### Clear Upgrade Status

**Endpoint:** `GET /cgi-bin/upgrade_clear.cgi`

**Description:** Clears upgrade status/cache.

**Response:**
```json
{
  "stats": "error",
  "code": "U006",
  "msg": "Fail!"
}
```

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/upgrade_clear.cgi'
```

---

## Log Endpoints

### Get System Logs

**Endpoint:** `GET /cgi-bin/log.cgi`

**Description:** Returns system/kernel boot logs.

**Response (Plain Text):**
```
[    0.000000] [0] Booting Linux on physical CPU 0x0
[    0.000000] [0] Linux version 4.9.38-tag- (jenkins@nomd-nomd-fwc-bj) ...
[    0.000000] [0] Boot CPU: AArch64 Processor [410fd034]
[    0.000000] [0] earlycon: uart0 at MMIO32 0x0000000004140000 (options '')
[    0.000000] [0] bootconsole [uart0] enabled
...
```

**Example:**
```bash
curl --digest -u root:root \
  'http://192.168.1.27/cgi-bin/log.cgi'
```

---

## Error Codes

### Common Response Structure

Success responses:
```json
{
  "stats": "success",
  "code": "X000",
  "msg": "OK!"
}
```

Error responses:
```json
{
  "stats": "error",
  "code": "X001",
  "msg": "Error description"
}
```

### Code Prefixes

| Prefix | Category |
|--------|----------|
| M | Miner configuration |
| N | Network configuration |
| B | Blink/LED control |
| U | Upgrade/firmware |

### Known Error Codes

| Code | Description |
|------|-------------|
| M000 | Miner config success |
| M001 | Invalid miner configuration |
| N000 | Network config success |
| N001 | Invalid hostname |
| B001 | Blink command received |
| U001 | Upgrade error (invalid file) |
| U006 | Upgrade clear failed |

---

## Notes

### Model Differences

Different Antminer models may have different API endpoints available:

| Endpoint | S19/Older | KS5/Newer |
|----------|-----------|-----------|
| `get_miner_status.cgi` | Yes | No (404) |
| `summary.cgi` | Maybe | Yes |
| `stats.cgi` | Maybe | Yes |
| `pools.cgi` | Yes | Yes |
| `get_system_info.cgi` | Yes | Yes |
| `get_miner_conf.cgi` | Yes | Yes |

### Hashrate Units

- S19 (SHA256d): GH/s or TH/s
- KS5 (KHeavyHash/Kaspa): H/s (may show 0 when miner is idle)

### Best Practices

1. Always use Digest authentication
2. Check `get_system_info.cgi` first to determine miner type
3. Use `summary.cgi`/`stats.cgi` for newer models, `get_miner_status.cgi` for older models
4. Implement fallback logic when endpoints return 404
5. Save configuration backups before making changes
6. Wait for miner to stabilize after configuration changes before reading status

### Rate Limiting

The API does not have explicit rate limiting, but rapid requests may cause instability. Allow 1-2 seconds between requests during normal operation.
