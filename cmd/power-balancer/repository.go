package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Repository handles all database operations for the power balancer.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository and initializes the database schema.
func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(Schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return &Repository{db: db}, nil
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// --- Models ---

// GetOrCreateModel gets or creates a model by name.
func (r *Repository) GetOrCreateModel(ctx context.Context, name string) (*Model, error) {
	// Try to get existing
	model, err := r.GetModelByName(ctx, name)
	if err == nil {
		return model, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO models (name, created_at) VALUES (?, ?)`,
		name, time.Now())
	if err != nil {
		return nil, fmt.Errorf("insert model: %w", err)
	}

	id, _ := result.LastInsertId()
	return &Model{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

// GetModelByName retrieves a model by name.
func (r *Repository) GetModelByName(ctx context.Context, name string) (*Model, error) {
	model := &Model{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, min_preset_id, max_preset_id, created_at, updated_at FROM models WHERE name = ?`,
		name).Scan(&model.ID, &model.Name, &model.MinPresetID, &model.MaxPresetID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return nil, err
	}
	model.IsConfigured = model.MinPresetID != nil && model.MaxPresetID != nil
	return model, nil
}

// GetModelByID retrieves a model by ID.
func (r *Repository) GetModelByID(ctx context.Context, id int64) (*Model, error) {
	model := &Model{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, min_preset_id, max_preset_id, created_at, updated_at FROM models WHERE id = ?`,
		id).Scan(&model.ID, &model.Name, &model.MinPresetID, &model.MaxPresetID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return nil, err
	}
	model.IsConfigured = model.MinPresetID != nil && model.MaxPresetID != nil
	return model, nil
}

// ListModels returns all models with their preset counts and miner counts.
func (r *Repository) ListModels(ctx context.Context) ([]*Model, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.name, m.min_preset_id, m.max_preset_id, m.created_at, m.updated_at,
		       (SELECT COUNT(*) FROM miners WHERE model_id = m.id) as miner_count
		FROM models m
		ORDER BY m.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*Model
	for rows.Next() {
		m := &Model{}
		if err := rows.Scan(&m.ID, &m.Name, &m.MinPresetID, &m.MaxPresetID, &m.CreatedAt, &m.UpdatedAt, &m.MinerCount); err != nil {
			return nil, err
		}
		m.IsConfigured = m.MinPresetID != nil && m.MaxPresetID != nil
		models = append(models, m)
	}
	return models, rows.Err()
}

// UpdateModelLimits sets the min and max preset limits for a model.
func (r *Repository) UpdateModelLimits(ctx context.Context, modelID int64, minPresetID, maxPresetID *int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE models SET min_preset_id = ?, max_preset_id = ?, updated_at = ? WHERE id = ?`,
		minPresetID, maxPresetID, time.Now(), modelID)
	return err
}

// --- Model Presets ---

// UpsertModelPreset inserts or updates a preset for a model.
func (r *Repository) UpsertModelPreset(ctx context.Context, p *ModelPreset) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO model_presets (model_id, name, watts, hashrate_th, display_name, requires_modded_psu, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(model_id, name) DO UPDATE SET
			watts = excluded.watts,
			hashrate_th = excluded.hashrate_th,
			display_name = excluded.display_name,
			requires_modded_psu = excluded.requires_modded_psu,
			sort_order = excluded.sort_order`,
		p.ModelID, p.Name, p.Watts, p.HashrateTH, p.DisplayName, p.RequiresModdedPSU, p.SortOrder)
	return err
}

// GetModelPresets returns all presets for a model, ordered by watts.
func (r *Repository) GetModelPresets(ctx context.Context, modelID int64) ([]*ModelPreset, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, model_id, name, watts, hashrate_th, display_name, requires_modded_psu, sort_order
		FROM model_presets WHERE model_id = ? ORDER BY watts ASC`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []*ModelPreset
	for rows.Next() {
		p := &ModelPreset{}
		if err := rows.Scan(&p.ID, &p.ModelID, &p.Name, &p.Watts, &p.HashrateTH, &p.DisplayName, &p.RequiresModdedPSU, &p.SortOrder); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

// GetPresetByID retrieves a preset by ID.
func (r *Repository) GetPresetByID(ctx context.Context, id int64) (*ModelPreset, error) {
	p := &ModelPreset{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, model_id, name, watts, hashrate_th, display_name, requires_modded_psu, sort_order
		FROM model_presets WHERE id = ?`, id).Scan(
		&p.ID, &p.ModelID, &p.Name, &p.Watts, &p.HashrateTH, &p.DisplayName, &p.RequiresModdedPSU, &p.SortOrder)
	return p, err
}

// GetPresetByModelAndName retrieves a preset by model ID and name.
func (r *Repository) GetPresetByModelAndName(ctx context.Context, modelID int64, name string) (*ModelPreset, error) {
	p := &ModelPreset{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, model_id, name, watts, hashrate_th, display_name, requires_modded_psu, sort_order
		FROM model_presets WHERE model_id = ? AND name = ?`, modelID, name).Scan(
		&p.ID, &p.ModelID, &p.Name, &p.Watts, &p.HashrateTH, &p.DisplayName, &p.RequiresModdedPSU, &p.SortOrder)
	return p, err
}

// --- Miners ---

// UpsertMiner inserts or updates a miner by MAC address.
func (r *Repository) UpsertMiner(ctx context.Context, m *Miner) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO miners (mac_address, ip_address, model_id, firmware_type, current_preset_id, is_online, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac_address) DO UPDATE SET
			ip_address = excluded.ip_address,
			model_id = excluded.model_id,
			firmware_type = excluded.firmware_type,
			current_preset_id = excluded.current_preset_id,
			is_online = excluded.is_online,
			last_seen = excluded.last_seen`,
		m.MACAddress, m.IPAddress, m.ModelID, m.FirmwareType, m.CurrentPresetID, m.IsOnline, m.LastSeen, time.Now())
	if err != nil {
		return err
	}

	// Get the ID (either from insert or existing)
	if m.ID == 0 {
		id, _ := result.LastInsertId()
		if id == 0 {
			// Was an update, need to get existing ID
			if err := r.db.QueryRowContext(ctx, `SELECT id FROM miners WHERE mac_address = ?`, m.MACAddress).Scan(&m.ID); err != nil {
				return fmt.Errorf("get miner id after upsert: %w", err)
			}
		} else {
			m.ID = id
		}
	}

	// Validate ID was set
	if m.ID == 0 {
		return fmt.Errorf("miner ID not set after upsert for MAC %s", m.MACAddress)
	}

	return nil
}

// UpsertMinerWithBalanceConfig atomically upserts a miner and ensures balance config exists.
// This wraps both operations in a single transaction to avoid FK constraint failures.
func (r *Repository) UpsertMinerWithBalanceConfig(ctx context.Context, m *Miner) (*BalanceConfig, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert miner within transaction
	result, err := tx.ExecContext(ctx, `
		INSERT INTO miners (mac_address, ip_address, model_id, firmware_type, current_preset_id, is_online, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac_address) DO UPDATE SET
			ip_address = excluded.ip_address,
			model_id = excluded.model_id,
			firmware_type = excluded.firmware_type,
			current_preset_id = excluded.current_preset_id,
			is_online = excluded.is_online,
			last_seen = excluded.last_seen`,
		m.MACAddress, m.IPAddress, m.ModelID, m.FirmwareType, m.CurrentPresetID, m.IsOnline, m.LastSeen, time.Now())
	if err != nil {
		return nil, fmt.Errorf("upsert miner: %w", err)
	}

	// Get the ID (either from insert or existing)
	if m.ID == 0 {
		id, _ := result.LastInsertId()
		if id == 0 {
			// Was an update, need to get existing ID
			if err := tx.QueryRowContext(ctx, `SELECT id FROM miners WHERE mac_address = ?`, m.MACAddress).Scan(&m.ID); err != nil {
				return nil, fmt.Errorf("get miner id: %w", err)
			}
		} else {
			m.ID = id
		}
	}

	// Validate ID was set
	if m.ID == 0 {
		return nil, fmt.Errorf("miner ID not set after upsert for MAC %s", m.MACAddress)
	}

	// Get or create balance config within same transaction
	config := &BalanceConfig{MinerID: m.ID}
	err = tx.QueryRowContext(ctx,
		`SELECT miner_id, enabled, priority, locked FROM balance_config WHERE miner_id = ?`,
		m.ID).Scan(&config.MinerID, &config.Enabled, &config.Priority, &config.Locked)

	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO balance_config (miner_id, enabled, priority, locked) VALUES (?, 0, 50, 0)`,
			m.ID)
		if err != nil {
			return nil, fmt.Errorf("create balance config: %w", err)
		}
		config.Enabled = false
		config.Priority = 50
		config.Locked = false
	} else if err != nil {
		return nil, fmt.Errorf("get balance config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return config, nil
}

// GetMinerByID retrieves a miner by ID with joined data.
func (r *Repository) GetMinerByID(ctx context.Context, id int64) (*Miner, error) {
	m := &Miner{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, mac_address, ip_address, model_id, firmware_type, current_preset_id, is_online, last_seen, created_at
		FROM miners WHERE id = ?`, id).Scan(
		&m.ID, &m.MACAddress, &m.IPAddress, &m.ModelID, &m.FirmwareType, &m.CurrentPresetID, &m.IsOnline, &m.LastSeen, &m.CreatedAt)
	return m, err
}

// GetMinerByMAC retrieves a miner by MAC address.
func (r *Repository) GetMinerByMAC(ctx context.Context, mac string) (*Miner, error) {
	m := &Miner{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, mac_address, ip_address, model_id, firmware_type, current_preset_id, is_online, last_seen, created_at
		FROM miners WHERE mac_address = ?`, mac).Scan(
		&m.ID, &m.MACAddress, &m.IPAddress, &m.ModelID, &m.FirmwareType, &m.CurrentPresetID, &m.IsOnline, &m.LastSeen, &m.CreatedAt)
	return m, err
}

// ListMiners returns all miners with optional filtering.
func (r *Repository) ListMiners(ctx context.Context) ([]*Miner, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, mac_address, ip_address, model_id, firmware_type, current_preset_id, is_online, last_seen, created_at
		FROM miners ORDER BY ip_address`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var miners []*Miner
	for rows.Next() {
		m := &Miner{}
		if err := rows.Scan(&m.ID, &m.MACAddress, &m.IPAddress, &m.ModelID, &m.FirmwareType, &m.CurrentPresetID, &m.IsOnline, &m.LastSeen, &m.CreatedAt); err != nil {
			return nil, err
		}
		miners = append(miners, m)
	}
	return miners, rows.Err()
}

// UpdateMinerPreset updates the current preset for a miner.
func (r *Repository) UpdateMinerPreset(ctx context.Context, minerID int64, presetID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE miners SET current_preset_id = ?, last_seen = ? WHERE id = ?`,
		presetID, time.Now(), minerID)
	return err
}

// --- Balance Config ---

// GetOrCreateBalanceConfig gets or creates a balance config for a miner.
func (r *Repository) GetOrCreateBalanceConfig(ctx context.Context, minerID int64) (*BalanceConfig, error) {
	config := &BalanceConfig{MinerID: minerID}
	err := r.db.QueryRowContext(ctx,
		`SELECT miner_id, enabled, priority, locked FROM balance_config WHERE miner_id = ?`,
		minerID).Scan(&config.MinerID, &config.Enabled, &config.Priority, &config.Locked)
	if err == sql.ErrNoRows {
		// Create default config
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO balance_config (miner_id, enabled, priority, locked) VALUES (?, 0, 50, 0)`,
			minerID)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
	return config, err
}

// UpdateBalanceConfig updates a miner's balance configuration.
func (r *Repository) UpdateBalanceConfig(ctx context.Context, config *BalanceConfig) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO balance_config (miner_id, enabled, priority, locked) VALUES (?, ?, ?, ?)
		ON CONFLICT(miner_id) DO UPDATE SET
			enabled = excluded.enabled,
			priority = excluded.priority,
			locked = excluded.locked`,
		config.MinerID, config.Enabled, config.Priority, config.Locked)
	return err
}

// --- Cooldowns ---

// SetCooldown sets a cooldown for a miner.
func (r *Repository) SetCooldown(ctx context.Context, minerID int64, until time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cooldowns (miner_id, until) VALUES (?, ?)
		ON CONFLICT(miner_id) DO UPDATE SET until = excluded.until`,
		minerID, until)
	return err
}

// GetCooldown gets the cooldown for a miner.
func (r *Repository) GetCooldown(ctx context.Context, minerID int64) (*Cooldown, error) {
	c := &Cooldown{}
	err := r.db.QueryRowContext(ctx,
		`SELECT miner_id, until FROM cooldowns WHERE miner_id = ?`,
		minerID).Scan(&c.MinerID, &c.Until)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// ClearExpiredCooldowns removes cooldowns that have passed.
func (r *Repository) ClearExpiredCooldowns(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cooldowns WHERE until < ?`, time.Now())
	return err
}

// CountMinersOnCooldown counts miners currently on cooldown.
func (r *Repository) CountMinersOnCooldown(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cooldowns WHERE until > ?`, time.Now()).Scan(&count)
	return count, err
}

// --- Pending Changes ---

// CreatePendingChange records a pending preset change.
func (r *Repository) CreatePendingChange(ctx context.Context, p *PendingChange) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO pending_changes (miner_id, from_preset_id, to_preset_id, expected_delta_w, issued_at, settles_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.MinerID, p.FromPresetID, p.ToPresetID, p.ExpectedDeltaW, p.IssuedAt, p.SettlesAt)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

// GetPendingChanges returns all pending changes that haven't settled yet.
func (r *Repository) GetPendingChanges(ctx context.Context) ([]*PendingChange, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, from_preset_id, to_preset_id, expected_delta_w, issued_at, settles_at
		FROM pending_changes WHERE settles_at > ?`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []*PendingChange
	for rows.Next() {
		p := &PendingChange{}
		if err := rows.Scan(&p.ID, &p.MinerID, &p.FromPresetID, &p.ToPresetID, &p.ExpectedDeltaW, &p.IssuedAt, &p.SettlesAt); err != nil {
			return nil, err
		}
		changes = append(changes, p)
	}
	return changes, rows.Err()
}

// SumPendingDelta returns the sum of expected power changes from pending changes.
func (r *Repository) SumPendingDelta(ctx context.Context) (int, error) {
	var sum sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT SUM(expected_delta_w) FROM pending_changes WHERE settles_at > ?`,
		time.Now()).Scan(&sum)
	if err != nil {
		return 0, err
	}
	return int(sum.Int64), nil
}

// ClearSettledChanges removes pending changes that have settled.
func (r *Repository) ClearSettledChanges(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM pending_changes WHERE settles_at <= ?`, time.Now())
	return err
}

// --- Energy Readings ---

// InsertEnergyReading stores an energy reading.
func (r *Repository) InsertEnergyReading(ctx context.Context, e *EnergyReading) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO energy_readings (timestamp, generation_mw, consumption_mw, margin_mw, margin_percent,
			generoso_mw, generoso_status, nogueira_mw, nogueira_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp, e.GenerationMW, e.ConsumptionMW, e.MarginMW, e.MarginPercent,
		e.GenerosoMW, e.GenerosoStatus, e.NogueiraMW, e.NogueiraStatus)
	if err != nil {
		return err
	}
	e.ID, _ = result.LastInsertId()
	return nil
}

// GetLatestEnergyReading returns the most recent energy reading.
func (r *Repository) GetLatestEnergyReading(ctx context.Context) (*EnergyReading, error) {
	e := &EnergyReading{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, timestamp, generation_mw, consumption_mw, margin_mw, margin_percent,
			generoso_mw, generoso_status, nogueira_mw, nogueira_status
		FROM energy_readings ORDER BY timestamp DESC LIMIT 1`).Scan(
		&e.ID, &e.Timestamp, &e.GenerationMW, &e.ConsumptionMW, &e.MarginMW, &e.MarginPercent,
		&e.GenerosoMW, &e.GenerosoStatus, &e.NogueiraMW, &e.NogueiraStatus)
	return e, err
}

// GetRecentEnergyReadings returns recent energy readings for charting.
func (r *Repository) GetRecentEnergyReadings(ctx context.Context, limit int) ([]*EnergyReading, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, timestamp, generation_mw, consumption_mw, margin_mw, margin_percent,
			generoso_mw, generoso_status, nogueira_mw, nogueira_status
		FROM energy_readings ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []*EnergyReading
	for rows.Next() {
		e := &EnergyReading{}
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.GenerationMW, &e.ConsumptionMW, &e.MarginMW, &e.MarginPercent,
			&e.GenerosoMW, &e.GenerosoStatus, &e.NogueiraMW, &e.NogueiraStatus); err != nil {
			return nil, err
		}
		readings = append(readings, e)
	}
	return readings, rows.Err()
}

// --- Change Log ---

// InsertChangeLog records a preset change for audit purposes.
func (r *Repository) InsertChangeLog(ctx context.Context, c *ChangeLog) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO change_log (miner_id, miner_ip, model_name, from_preset, to_preset,
			expected_delta_w, reason, margin_at_time, issued_at, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.MinerID, c.MinerIP, c.ModelName, c.FromPreset, c.ToPreset,
		c.ExpectedDeltaW, c.Reason, c.MarginAtTime, c.IssuedAt, c.Success, c.ErrorMessage)
	if err != nil {
		return err
	}
	c.ID, _ = result.LastInsertId()
	return nil
}

// GetRecentChangeLogs returns recent change logs.
func (r *Repository) GetRecentChangeLogs(ctx context.Context, limit int) ([]*ChangeLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, miner_ip, model_name, from_preset, to_preset,
			expected_delta_w, reason, margin_at_time, issued_at, success, error_message
		FROM change_log ORDER BY issued_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*ChangeLog
	for rows.Next() {
		c := &ChangeLog{}
		var errMsg sql.NullString
		if err := rows.Scan(&c.ID, &c.MinerID, &c.MinerIP, &c.ModelName, &c.FromPreset, &c.ToPreset,
			&c.ExpectedDeltaW, &c.Reason, &c.MarginAtTime, &c.IssuedAt, &c.Success, &errMsg); err != nil {
			return nil, err
		}
		c.ErrorMessage = errMsg.String
		logs = append(logs, c)
	}
	return logs, rows.Err()
}

// --- Aggregated Queries for Balancing ---

// GetManageableMiners returns miners that can be managed (configured model, enabled, not locked, online).
func (r *Repository) GetManageableMiners(ctx context.Context) ([]*MinerWithContext, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			m.id, m.mac_address, m.ip_address, m.model_id, m.firmware_type, m.current_preset_id, m.last_seen,
			mo.id, mo.name, mo.min_preset_id, mo.max_preset_id,
			cp.id, cp.name, cp.watts, cp.hashrate_th,
			minp.id, minp.name, minp.watts,
			maxp.id, maxp.name, maxp.watts,
			bc.enabled, bc.priority, bc.locked,
			c.until
		FROM miners m
		JOIN models mo ON m.model_id = mo.id
		JOIN model_presets cp ON m.current_preset_id = cp.id
		JOIN model_presets minp ON mo.min_preset_id = minp.id
		JOIN model_presets maxp ON mo.max_preset_id = maxp.id
		JOIN balance_config bc ON m.id = bc.miner_id
		LEFT JOIN cooldowns c ON m.id = c.miner_id
		WHERE bc.enabled = 1 AND bc.locked = 0 AND m.firmware_type = 'vnish' AND m.is_online = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*MinerWithContext
	now := time.Now()
	for rows.Next() {
		mwc := &MinerWithContext{
			Miner:         &Miner{},
			Model:         &Model{},
			CurrentPreset: &ModelPreset{},
			MinPreset:     &ModelPreset{},
			MaxPreset:     &ModelPreset{},
			Config:        &BalanceConfig{},
		}
		var cooldownUntil sql.NullTime

		if err := rows.Scan(
			&mwc.Miner.ID, &mwc.Miner.MACAddress, &mwc.Miner.IPAddress, &mwc.Miner.ModelID,
			&mwc.Miner.FirmwareType, &mwc.Miner.CurrentPresetID, &mwc.Miner.LastSeen,
			&mwc.Model.ID, &mwc.Model.Name, &mwc.Model.MinPresetID, &mwc.Model.MaxPresetID,
			&mwc.CurrentPreset.ID, &mwc.CurrentPreset.Name, &mwc.CurrentPreset.Watts, &mwc.CurrentPreset.HashrateTH,
			&mwc.MinPreset.ID, &mwc.MinPreset.Name, &mwc.MinPreset.Watts,
			&mwc.MaxPreset.ID, &mwc.MaxPreset.Name, &mwc.MaxPreset.Watts,
			&mwc.Config.Enabled, &mwc.Config.Priority, &mwc.Config.Locked,
			&cooldownUntil,
		); err != nil {
			return nil, err
		}

		mwc.HeadroomWatts = mwc.CurrentPreset.Watts - mwc.MinPreset.Watts

		// Calculate efficiency: hashrate per watt (TH/W) - higher is better
		if mwc.CurrentPreset.Watts > 0 && mwc.CurrentPreset.HashrateTH > 0 {
			mwc.Efficiency = mwc.CurrentPreset.HashrateTH / float64(mwc.CurrentPreset.Watts)
		}

		if cooldownUntil.Valid {
			mwc.CooldownUntil = &cooldownUntil.Time
			mwc.OnCooldown = cooldownUntil.Time.After(now)
		}

		results = append(results, mwc)
	}
	return results, rows.Err()
}

// CountManagedMiners returns the count of enabled miners.
func (r *Repository) CountManagedMiners(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM balance_config bc
		JOIN miners m ON bc.miner_id = m.id
		JOIN models mo ON m.model_id = mo.id
		WHERE bc.enabled = 1 AND mo.min_preset_id IS NOT NULL AND mo.max_preset_id IS NOT NULL`).Scan(&count)
	return count, err
}

// --- Online Status Management ---

// SetMinerOnlineStatus sets the online status of a miner.
func (r *Repository) SetMinerOnlineStatus(ctx context.Context, minerID int64, isOnline bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE miners SET is_online = ?, last_seen = CASE WHEN ? = 1 THEN ? ELSE last_seen END WHERE id = ?`,
		isOnline, isOnline, time.Now(), minerID)
	return err
}

// MarkAllMinersOffline marks all miners as offline (used before discovery scan).
func (r *Repository) MarkAllMinersOffline(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `UPDATE miners SET is_online = 0`)
	return err
}

// ClearPendingChangesForOfflineMiners removes pending changes for miners that went offline.
func (r *Repository) ClearPendingChangesForOfflineMiners(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM pending_changes WHERE miner_id IN (
			SELECT id FROM miners WHERE is_online = 0
		)`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountOnlineMiners returns the count of online miners.
func (r *Repository) CountOnlineMiners(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM miners WHERE is_online = 1`).Scan(&count)
	return count, err
}
