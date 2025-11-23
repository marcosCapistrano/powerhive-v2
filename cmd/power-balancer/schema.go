package main

// Schema contains all SQL statements for creating the power-balancer database.
const Schema = `
-- Discovered miner models
CREATE TABLE IF NOT EXISTS models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    min_preset_id INTEGER,
    max_preset_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME,
    FOREIGN KEY (min_preset_id) REFERENCES model_presets(id),
    FOREIGN KEY (max_preset_id) REFERENCES model_presets(id)
);

-- Available presets for each model (discovered from VNish API)
CREATE TABLE IF NOT EXISTS model_presets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    watts INTEGER NOT NULL,
    hashrate_th REAL,
    display_name TEXT,
    requires_modded_psu INTEGER DEFAULT 0,
    sort_order INTEGER,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    UNIQUE(model_id, name)
);

-- Discovered miners
CREATE TABLE IF NOT EXISTS miners (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mac_address TEXT UNIQUE NOT NULL,
    ip_address TEXT NOT NULL,
    model_id INTEGER,
    firmware_type TEXT,
    current_preset_id INTEGER,
    is_online INTEGER DEFAULT 0,
    last_seen DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id),
    FOREIGN KEY (current_preset_id) REFERENCES model_presets(id)
);

-- Balancing config per miner
CREATE TABLE IF NOT EXISTS balance_config (
    miner_id INTEGER PRIMARY KEY,
    enabled INTEGER DEFAULT 0,
    priority INTEGER DEFAULT 50,
    locked INTEGER DEFAULT 0,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE
);

-- Pending changes awaiting settlement
CREATE TABLE IF NOT EXISTS pending_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER NOT NULL,
    from_preset_id INTEGER,
    to_preset_id INTEGER,
    expected_delta_w INTEGER,
    issued_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    settles_at DATETIME,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE,
    FOREIGN KEY (from_preset_id) REFERENCES model_presets(id),
    FOREIGN KEY (to_preset_id) REFERENCES model_presets(id)
);

-- Cooldowns
CREATE TABLE IF NOT EXISTS cooldowns (
    miner_id INTEGER PRIMARY KEY,
    until DATETIME NOT NULL,
    FOREIGN KEY (miner_id) REFERENCES miners(id) ON DELETE CASCADE
);

-- Energy readings history
CREATE TABLE IF NOT EXISTS energy_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    generation_mw REAL,
    consumption_mw REAL,
    margin_mw REAL,
    margin_percent REAL,
    generoso_mw REAL,
    generoso_status TEXT,
    nogueira_mw REAL,
    nogueira_status TEXT
);

-- Audit log
CREATE TABLE IF NOT EXISTS change_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    miner_id INTEGER,
    miner_ip TEXT,
    model_name TEXT,
    from_preset TEXT,
    to_preset TEXT,
    expected_delta_w INTEGER,
    reason TEXT,
    margin_at_time REAL,
    issued_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    success INTEGER,
    error_message TEXT
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_miners_model ON miners(model_id);
CREATE INDEX IF NOT EXISTS idx_model_presets_model ON model_presets(model_id);
CREATE INDEX IF NOT EXISTS idx_pending_settles ON pending_changes(settles_at);
CREATE INDEX IF NOT EXISTS idx_cooldowns_until ON cooldowns(until);
CREATE INDEX IF NOT EXISTS idx_energy_readings_timestamp ON energy_readings(timestamp);
CREATE INDEX IF NOT EXISTS idx_change_log_issued ON change_log(issued_at);
`
