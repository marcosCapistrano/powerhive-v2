package database

// Schema contains the SQLite database schema.
// All tables use INTEGER PRIMARY KEY for auto-increment IDs.
const Schema = `
-- Miners table: Core device identity
-- MAC address is the primary unique identifier (IPs can change with DHCP)
CREATE TABLE IF NOT EXISTS miners (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mac_address TEXT NOT NULL UNIQUE,  -- Primary identifier (hardware-level)
    ip_address TEXT NOT NULL,          -- Current IP (can change)
    hostname TEXT,
    serial_number TEXT,
    firmware_type TEXT NOT NULL, -- 'vnish', 'stock', 'braiins', 'unknown'
    firmware_version TEXT,
    model TEXT,                  -- e.g., "s19", "ks5"
    miner_type TEXT,             -- Full name e.g., "Antminer S19"
    algorithm TEXT,              -- e.g., "sha256d", "KHeavyHash"
    platform TEXT,               -- VNish: "xil"
    hr_measure TEXT,             -- Hashrate unit e.g., "GH/s"
    is_online INTEGER DEFAULT 1, -- 1 = online, 0 = offline
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_miners_ip ON miners(ip_address);
CREATE INDEX IF NOT EXISTS idx_miners_serial ON miners(serial_number);
CREATE INDEX IF NOT EXISTS idx_miners_firmware ON miners(firmware_type);
CREATE INDEX IF NOT EXISTS idx_miners_model ON miners(miner_type);
CREATE INDEX IF NOT EXISTS idx_miners_online ON miners(is_online);

-- Miner network configuration
CREATE TABLE IF NOT EXISTS miner_network (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    dhcp INTEGER DEFAULT 1,      -- 0 = static, 1 = dhcp
    ip_address TEXT,
    netmask TEXT,
    gateway TEXT,
    dns_servers TEXT,            -- Comma-separated or JSON
    net_device TEXT,             -- e.g., "eth0"
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id)
);

-- Miner hardware specifications
CREATE TABLE IF NOT EXISTS miner_hardware (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    num_chains INTEGER,
    chips_per_chain INTEGER,
    total_asic_count INTEGER,
    min_voltage INTEGER,         -- mV
    max_voltage INTEGER,
    default_voltage INTEGER,
    min_freq INTEGER,            -- MHz
    max_freq INTEGER,
    default_freq INTEGER,
    min_fan_pwm INTEGER,
    min_target_temp INTEGER,
    max_target_temp INTEGER,
    fan_count INTEGER,
    psu_model TEXT,
    psu_serial TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id)
);

-- Current miner operational status (snapshot)
CREATE TABLE IF NOT EXISTS miner_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    state TEXT,                  -- "running", "stopped", "failure"
    state_time INTEGER,          -- Time in current state (seconds)
    description TEXT,
    failure_code INTEGER,
    uptime_seconds INTEGER,
    unlocked INTEGER DEFAULT 0,  -- VNish: unlocked for modifications
    restart_required INTEGER DEFAULT 0,
    reboot_required INTEGER DEFAULT 0,
    find_miner INTEGER DEFAULT 0, -- LED blink status
    rate_status TEXT,            -- Stock: "s", "w", "e"
    network_status TEXT,
    fans_status TEXT,
    temp_status TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id)
);

-- Current mining performance (snapshot)
CREATE TABLE IF NOT EXISTS miner_summary (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    hashrate_instant REAL,
    hashrate_avg REAL,
    hashrate_5s REAL,
    hashrate_30m REAL,
    hashrate_ideal REAL,
    hashrate_nominal REAL,
    power_consumption INTEGER,   -- Watts
    power_efficiency REAL,       -- J/TH
    pcb_temp_min INTEGER,
    pcb_temp_max INTEGER,
    chip_temp_min INTEGER,
    chip_temp_max INTEGER,
    hw_errors INTEGER,
    hw_error_percent REAL,
    accepted INTEGER,
    rejected INTEGER,
    stale INTEGER,
    best_share INTEGER,
    found_blocks INTEGER,
    devfee_percent REAL,         -- VNish
    fan_count INTEGER,
    fan_duty INTEGER,
    fan_mode TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id)
);

-- Hash boards (chains)
CREATE TABLE IF NOT EXISTS miner_chains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    chain_index INTEGER NOT NULL,
    serial_number TEXT,
    freq_avg INTEGER,            -- MHz
    hashrate_ideal REAL,
    hashrate_real REAL,
    asic_num INTEGER,
    voltage INTEGER,             -- mV (VNish)
    temp_pcb INTEGER,
    temp_chip INTEGER,
    temp_pic INTEGER,            -- Stock
    hw_errors INTEGER,
    eeprom_loaded INTEGER DEFAULT 1, -- Stock
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id, chain_index)
);

CREATE INDEX IF NOT EXISTS idx_miner_chains_miner ON miner_chains(miner_id);

-- Pool configurations
CREATE TABLE IF NOT EXISTS miner_pools (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    pool_index INTEGER NOT NULL, -- 0, 1, 2
    url TEXT,
    user TEXT,
    password TEXT,
    status TEXT,                 -- "Alive", "Dead", "working"
    priority INTEGER,
    accepted INTEGER,
    rejected INTEGER,
    stale INTEGER,
    discarded INTEGER,
    difficulty TEXT,
    diff_accepted REAL,
    asic_boost INTEGER DEFAULT 0, -- VNish
    ping INTEGER,                -- VNish: latency ms
    pool_type TEXT,              -- VNish: "DevFee"
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id, pool_index)
);

CREATE INDEX IF NOT EXISTS idx_miner_pools_miner ON miner_pools(miner_id);

-- Fan status
CREATE TABLE IF NOT EXISTS miner_fans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    fan_index INTEGER NOT NULL,
    rpm INTEGER,
    duty_cycle INTEGER,          -- 0-100%
    status TEXT,                 -- "ok", "failed"
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id, fan_index)
);

CREATE INDEX IF NOT EXISTS idx_miner_fans_miner ON miner_fans(miner_id);

-- Historical time-series metrics
CREATE TABLE IF NOT EXISTS miner_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    hashrate REAL,
    power_consumption INTEGER,
    pcb_temp_max INTEGER,
    chip_temp_max INTEGER,
    fan_duty INTEGER,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_miner_metrics_miner_time ON miner_metrics(miner_id, timestamp);

-- VNish autotune presets
CREATE TABLE IF NOT EXISTS autotune_presets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    name TEXT NOT NULL,          -- e.g., "1100", "1300"
    pretty_name TEXT,            -- e.g., "1100 watt ~ 53 TH"
    status TEXT,                 -- "tuned", "untuned"
    modded_psu_required INTEGER DEFAULT 0,
    target_power INTEGER,        -- Watts
    target_hashrate REAL,        -- TH/s
    voltage INTEGER,             -- mV
    frequency INTEGER,           -- MHz
    is_current INTEGER DEFAULT 0, -- Is this the active preset
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id, name)
);

CREATE INDEX IF NOT EXISTS idx_autotune_presets_miner ON autotune_presets(miner_id);

-- VNish notes (key-value storage)
CREATE TABLE IF NOT EXISTS miner_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    UNIQUE(miner_id, key)
);

CREATE INDEX IF NOT EXISTS idx_miner_notes_miner ON miner_notes(miner_id);

-- Schema version for migrations
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// SchemaVersion is the current schema version.
const SchemaVersion = 1

// Migrations contains SQL migrations indexed by version.
// Each migration upgrades from version N-1 to version N.
var Migrations = map[int]string{
	1: Schema, // Initial schema
}
