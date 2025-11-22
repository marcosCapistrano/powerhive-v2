package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository.
// The dbPath can be a file path or ":memory:" for in-memory database.
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return repo, nil
}

// migrate runs database migrations.
func (r *SQLiteRepository) migrate() error {
	// Get current schema version
	var currentVersion int
	err := r.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		// Table doesn't exist, run initial schema
		if _, err := r.db.Exec(Schema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		_, err = r.db.Exec("INSERT INTO schema_version (version) VALUES (?)", SchemaVersion)
		return err
	}

	// Run any pending migrations
	for v := currentVersion + 1; v <= SchemaVersion; v++ {
		migration, ok := Migrations[v]
		if !ok {
			continue
		}
		if _, err := r.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", v, err)
		}
		if _, err := r.db.Exec("INSERT INTO schema_version (version) VALUES (?)", v); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", v, err)
		}
	}
	return nil
}

// Close closes the database connection.
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// DB returns the underlying database connection for advanced queries.
func (r *SQLiteRepository) DB() *sql.DB {
	return r.db
}

// =============================================================================
// Miner CRUD
// =============================================================================

func (r *SQLiteRepository) CreateMiner(ctx context.Context, m *Miner) error {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	m.LastSeenAt = now
	m.IsOnline = true // New miners are online by default

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO miners (mac_address, ip_address, hostname, serial_number, firmware_type,
			firmware_version, model, miner_type, algorithm, platform, hr_measure, is_online,
			created_at, updated_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.MACAddress, m.IPAddress, m.Hostname, m.SerialNumber, m.FirmwareType,
		m.FirmwareVersion, m.Model, m.MinerType, m.Algorithm, m.Platform, m.HRMeasure, m.IsOnline,
		m.CreatedAt, m.UpdatedAt, m.LastSeenAt)
	if err != nil {
		return err
	}
	m.ID, _ = result.LastInsertId()
	return nil
}

func (r *SQLiteRepository) GetMiner(ctx context.Context, id int64) (*Miner, error) {
	m := &Miner{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, mac_address, ip_address, hostname, serial_number, firmware_type,
			firmware_version, model, miner_type, algorithm, platform, hr_measure, is_online,
			created_at, updated_at, last_seen_at
		FROM miners WHERE id = ?`, id).Scan(
		&m.ID, &m.MACAddress, &m.IPAddress, &m.Hostname, &m.SerialNumber, &m.FirmwareType,
		&m.FirmwareVersion, &m.Model, &m.MinerType, &m.Algorithm, &m.Platform, &m.HRMeasure, &m.IsOnline,
		&m.CreatedAt, &m.UpdatedAt, &m.LastSeenAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *SQLiteRepository) GetMinerByIP(ctx context.Context, ip string) (*Miner, error) {
	m := &Miner{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, mac_address, ip_address, hostname, serial_number, firmware_type,
			firmware_version, model, miner_type, algorithm, platform, hr_measure, is_online,
			created_at, updated_at, last_seen_at
		FROM miners WHERE ip_address = ?`, ip).Scan(
		&m.ID, &m.MACAddress, &m.IPAddress, &m.Hostname, &m.SerialNumber, &m.FirmwareType,
		&m.FirmwareVersion, &m.Model, &m.MinerType, &m.Algorithm, &m.Platform, &m.HRMeasure, &m.IsOnline,
		&m.CreatedAt, &m.UpdatedAt, &m.LastSeenAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *SQLiteRepository) GetMinerByMAC(ctx context.Context, mac string) (*Miner, error) {
	m := &Miner{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, mac_address, ip_address, hostname, serial_number, firmware_type,
			firmware_version, model, miner_type, algorithm, platform, hr_measure, is_online,
			created_at, updated_at, last_seen_at
		FROM miners WHERE mac_address = ?`, mac).Scan(
		&m.ID, &m.MACAddress, &m.IPAddress, &m.Hostname, &m.SerialNumber, &m.FirmwareType,
		&m.FirmwareVersion, &m.Model, &m.MinerType, &m.Algorithm, &m.Platform, &m.HRMeasure, &m.IsOnline,
		&m.CreatedAt, &m.UpdatedAt, &m.LastSeenAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *SQLiteRepository) ListMiners(ctx context.Context) ([]*Miner, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, mac_address, ip_address, hostname, serial_number, firmware_type,
			firmware_version, model, miner_type, algorithm, platform, hr_measure, is_online,
			created_at, updated_at, last_seen_at
		FROM miners ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var miners []*Miner
	for rows.Next() {
		m := &Miner{}
		if err := rows.Scan(
			&m.ID, &m.MACAddress, &m.IPAddress, &m.Hostname, &m.SerialNumber, &m.FirmwareType,
			&m.FirmwareVersion, &m.Model, &m.MinerType, &m.Algorithm, &m.Platform, &m.HRMeasure, &m.IsOnline,
			&m.CreatedAt, &m.UpdatedAt, &m.LastSeenAt); err != nil {
			return nil, err
		}
		miners = append(miners, m)
	}
	return miners, rows.Err()
}

func (r *SQLiteRepository) UpdateMiner(ctx context.Context, m *Miner) error {
	m.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE miners SET mac_address = ?, ip_address = ?, hostname = ?, serial_number = ?,
			firmware_type = ?, firmware_version = ?, model = ?, miner_type = ?,
			algorithm = ?, platform = ?, hr_measure = ?, is_online = ?, updated_at = ?, last_seen_at = ?
		WHERE id = ?`,
		m.MACAddress, m.IPAddress, m.Hostname, m.SerialNumber, m.FirmwareType,
		m.FirmwareVersion, m.Model, m.MinerType, m.Algorithm, m.Platform, m.HRMeasure, m.IsOnline,
		m.UpdatedAt, m.LastSeenAt, m.ID)
	return err
}

func (r *SQLiteRepository) DeleteMiner(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM miners WHERE id = ?", id)
	return err
}

func (r *SQLiteRepository) UpsertMinerByIP(ctx context.Context, m *Miner) error {
	existing, err := r.GetMinerByIP(ctx, m.IPAddress)
	if err != nil {
		return err
	}
	if existing != nil {
		m.ID = existing.ID
		m.CreatedAt = existing.CreatedAt
		m.LastSeenAt = time.Now()
		m.IsOnline = true
		return r.UpdateMiner(ctx, m)
	}
	return r.CreateMiner(ctx, m)
}

func (r *SQLiteRepository) UpsertMinerByMAC(ctx context.Context, m *Miner) error {
	existing, err := r.GetMinerByMAC(ctx, m.MACAddress)
	if err != nil {
		return err
	}
	if existing != nil {
		m.ID = existing.ID
		m.CreatedAt = existing.CreatedAt
		m.LastSeenAt = time.Now()
		m.IsOnline = true
		return r.UpdateMiner(ctx, m)
	}
	return r.CreateMiner(ctx, m)
}

func (r *SQLiteRepository) SetMinerOnlineStatus(ctx context.Context, id int64, online bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE miners SET is_online = ?, updated_at = ? WHERE id = ?`,
		online, time.Now(), id)
	return err
}

func (r *SQLiteRepository) MarkAllMinersOffline(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `UPDATE miners SET is_online = 0, updated_at = ?`, time.Now())
	return err
}

func (r *SQLiteRepository) GetDistinctMinerTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT miner_type FROM miners WHERE miner_type IS NOT NULL AND miner_type != '' ORDER BY miner_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

func (r *SQLiteRepository) ListMinersFiltered(ctx context.Context, filter MinerFilter) ([]*Miner, error) {
	// Build query with filters
	query := `
		SELECT m.id, m.mac_address, m.ip_address, m.hostname, m.serial_number, m.firmware_type,
			m.firmware_version, m.model, m.miner_type, m.algorithm, m.platform, m.hr_measure, m.is_online,
			m.created_at, m.updated_at, m.last_seen_at
		FROM miners m
		LEFT JOIN miner_summary s ON m.id = s.miner_id
		LEFT JOIN miner_status st ON m.id = st.miner_id
		WHERE 1=1`

	var args []interface{}

	// Apply filters
	if filter.MinerType != "" {
		query += " AND m.miner_type = ?"
		args = append(args, filter.MinerType)
	}
	if filter.FirmwareType != "" {
		query += " AND m.firmware_type = ?"
		args = append(args, filter.FirmwareType)
	}
	if filter.OnlineStatus == "online" {
		query += " AND m.is_online = 1"
	} else if filter.OnlineStatus == "offline" {
		query += " AND m.is_online = 0"
	}

	// Apply sorting
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	switch filter.SortBy {
	case "ip":
		query += " ORDER BY m.ip_address " + sortOrder
	case "model":
		query += " ORDER BY m.miner_type " + sortOrder
	case "hashrate":
		query += " ORDER BY COALESCE(s.hashrate_avg, 0) " + sortOrder
	case "power":
		query += " ORDER BY COALESCE(s.power_consumption, 0) " + sortOrder
	case "efficiency":
		query += " ORDER BY COALESCE(s.power_efficiency, 0) " + sortOrder
	case "temp":
		query += " ORDER BY COALESCE(s.chip_temp_max, 0) " + sortOrder
	case "uptime":
		query += " ORDER BY COALESCE(st.uptime_seconds, 0) " + sortOrder
	case "last_seen":
		query += " ORDER BY m.last_seen_at " + sortOrder
	default:
		query += " ORDER BY m.last_seen_at DESC"
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var miners []*Miner
	for rows.Next() {
		m := &Miner{}
		if err := rows.Scan(
			&m.ID, &m.MACAddress, &m.IPAddress, &m.Hostname, &m.SerialNumber, &m.FirmwareType,
			&m.FirmwareVersion, &m.Model, &m.MinerType, &m.Algorithm, &m.Platform, &m.HRMeasure, &m.IsOnline,
			&m.CreatedAt, &m.UpdatedAt, &m.LastSeenAt); err != nil {
			return nil, err
		}
		miners = append(miners, m)
	}
	return miners, rows.Err()
}

// =============================================================================
// Network
// =============================================================================

func (r *SQLiteRepository) GetMinerNetwork(ctx context.Context, minerID int64) (*MinerNetwork, error) {
	n := &MinerNetwork{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, miner_id, dhcp, ip_address, netmask, gateway, dns_servers, net_device, updated_at
		FROM miner_network WHERE miner_id = ?`, minerID).Scan(
		&n.ID, &n.MinerID, &n.DHCP, &n.IPAddress, &n.Netmask, &n.Gateway,
		&n.DNSServers, &n.NetDevice, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return n, err
}

func (r *SQLiteRepository) UpsertMinerNetwork(ctx context.Context, n *MinerNetwork) error {
	n.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_network (miner_id, dhcp, ip_address, netmask, gateway, dns_servers, net_device, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id) DO UPDATE SET
			dhcp = excluded.dhcp, ip_address = excluded.ip_address, netmask = excluded.netmask,
			gateway = excluded.gateway, dns_servers = excluded.dns_servers, net_device = excluded.net_device,
			updated_at = excluded.updated_at`,
		n.MinerID, n.DHCP, n.IPAddress, n.Netmask, n.Gateway, n.DNSServers, n.NetDevice, n.UpdatedAt)
	return err
}

// =============================================================================
// Hardware
// =============================================================================

func (r *SQLiteRepository) GetMinerHardware(ctx context.Context, minerID int64) (*MinerHardware, error) {
	h := &MinerHardware{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, miner_id, num_chains, chips_per_chain, total_asic_count,
			min_voltage, max_voltage, default_voltage, min_freq, max_freq, default_freq,
			min_fan_pwm, min_target_temp, max_target_temp, fan_count, psu_model, psu_serial, updated_at
		FROM miner_hardware WHERE miner_id = ?`, minerID).Scan(
		&h.ID, &h.MinerID, &h.NumChains, &h.ChipsPerChain, &h.TotalAsicCount,
		&h.MinVoltage, &h.MaxVoltage, &h.DefaultVoltage, &h.MinFreq, &h.MaxFreq, &h.DefaultFreq,
		&h.MinFanPWM, &h.MinTargetTemp, &h.MaxTargetTemp, &h.FanCount, &h.PSUModel, &h.PSUSerial, &h.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return h, err
}

func (r *SQLiteRepository) UpsertMinerHardware(ctx context.Context, h *MinerHardware) error {
	h.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_hardware (miner_id, num_chains, chips_per_chain, total_asic_count,
			min_voltage, max_voltage, default_voltage, min_freq, max_freq, default_freq,
			min_fan_pwm, min_target_temp, max_target_temp, fan_count, psu_model, psu_serial, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id) DO UPDATE SET
			num_chains = excluded.num_chains, chips_per_chain = excluded.chips_per_chain,
			total_asic_count = excluded.total_asic_count, min_voltage = excluded.min_voltage,
			max_voltage = excluded.max_voltage, default_voltage = excluded.default_voltage,
			min_freq = excluded.min_freq, max_freq = excluded.max_freq, default_freq = excluded.default_freq,
			min_fan_pwm = excluded.min_fan_pwm, min_target_temp = excluded.min_target_temp,
			max_target_temp = excluded.max_target_temp, fan_count = excluded.fan_count,
			psu_model = excluded.psu_model, psu_serial = excluded.psu_serial, updated_at = excluded.updated_at`,
		h.MinerID, h.NumChains, h.ChipsPerChain, h.TotalAsicCount,
		h.MinVoltage, h.MaxVoltage, h.DefaultVoltage, h.MinFreq, h.MaxFreq, h.DefaultFreq,
		h.MinFanPWM, h.MinTargetTemp, h.MaxTargetTemp, h.FanCount, h.PSUModel, h.PSUSerial, h.UpdatedAt)
	return err
}

// =============================================================================
// Status
// =============================================================================

func (r *SQLiteRepository) GetMinerStatus(ctx context.Context, minerID int64) (*MinerStatus, error) {
	s := &MinerStatus{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, miner_id, state, state_time, description, failure_code, uptime_seconds,
			unlocked, restart_required, reboot_required, find_miner,
			rate_status, network_status, fans_status, temp_status, updated_at
		FROM miner_status WHERE miner_id = ?`, minerID).Scan(
		&s.ID, &s.MinerID, &s.State, &s.StateTime, &s.Description, &s.FailureCode, &s.UptimeSeconds,
		&s.Unlocked, &s.RestartRequired, &s.RebootRequired, &s.FindMiner,
		&s.RateStatus, &s.NetworkStatus, &s.FansStatus, &s.TempStatus, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *SQLiteRepository) UpsertMinerStatus(ctx context.Context, s *MinerStatus) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_status (miner_id, state, state_time, description, failure_code, uptime_seconds,
			unlocked, restart_required, reboot_required, find_miner,
			rate_status, network_status, fans_status, temp_status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id) DO UPDATE SET
			state = excluded.state, state_time = excluded.state_time, description = excluded.description,
			failure_code = excluded.failure_code, uptime_seconds = excluded.uptime_seconds,
			unlocked = excluded.unlocked, restart_required = excluded.restart_required,
			reboot_required = excluded.reboot_required, find_miner = excluded.find_miner,
			rate_status = excluded.rate_status, network_status = excluded.network_status,
			fans_status = excluded.fans_status, temp_status = excluded.temp_status, updated_at = excluded.updated_at`,
		s.MinerID, s.State, s.StateTime, s.Description, s.FailureCode, s.UptimeSeconds,
		s.Unlocked, s.RestartRequired, s.RebootRequired, s.FindMiner,
		s.RateStatus, s.NetworkStatus, s.FansStatus, s.TempStatus, s.UpdatedAt)
	return err
}

// =============================================================================
// Summary
// =============================================================================

func (r *SQLiteRepository) GetMinerSummary(ctx context.Context, minerID int64) (*MinerSummary, error) {
	s := &MinerSummary{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, miner_id, hashrate_instant, hashrate_avg, hashrate_5s, hashrate_30m,
			hashrate_ideal, hashrate_nominal, power_consumption, power_efficiency,
			pcb_temp_min, pcb_temp_max, chip_temp_min, chip_temp_max,
			hw_errors, hw_error_percent, accepted, rejected, stale, best_share, found_blocks,
			devfee_percent, fan_count, fan_duty, fan_mode, updated_at
		FROM miner_summary WHERE miner_id = ?`, minerID).Scan(
		&s.ID, &s.MinerID, &s.HashrateInstant, &s.HashrateAvg, &s.Hashrate5s, &s.Hashrate30m,
		&s.HashrateIdeal, &s.HashrateNominal, &s.PowerConsumption, &s.PowerEfficiency,
		&s.PCBTempMin, &s.PCBTempMax, &s.ChipTempMin, &s.ChipTempMax,
		&s.HWErrors, &s.HWErrorPercent, &s.Accepted, &s.Rejected, &s.Stale, &s.BestShare, &s.FoundBlocks,
		&s.DevFeePercent, &s.FanCount, &s.FanDuty, &s.FanMode, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *SQLiteRepository) UpsertMinerSummary(ctx context.Context, s *MinerSummary) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_summary (miner_id, hashrate_instant, hashrate_avg, hashrate_5s, hashrate_30m,
			hashrate_ideal, hashrate_nominal, power_consumption, power_efficiency,
			pcb_temp_min, pcb_temp_max, chip_temp_min, chip_temp_max,
			hw_errors, hw_error_percent, accepted, rejected, stale, best_share, found_blocks,
			devfee_percent, fan_count, fan_duty, fan_mode, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id) DO UPDATE SET
			hashrate_instant = excluded.hashrate_instant, hashrate_avg = excluded.hashrate_avg,
			hashrate_5s = excluded.hashrate_5s, hashrate_30m = excluded.hashrate_30m,
			hashrate_ideal = excluded.hashrate_ideal, hashrate_nominal = excluded.hashrate_nominal,
			power_consumption = excluded.power_consumption, power_efficiency = excluded.power_efficiency,
			pcb_temp_min = excluded.pcb_temp_min, pcb_temp_max = excluded.pcb_temp_max,
			chip_temp_min = excluded.chip_temp_min, chip_temp_max = excluded.chip_temp_max,
			hw_errors = excluded.hw_errors, hw_error_percent = excluded.hw_error_percent,
			accepted = excluded.accepted, rejected = excluded.rejected, stale = excluded.stale,
			best_share = excluded.best_share, found_blocks = excluded.found_blocks,
			devfee_percent = excluded.devfee_percent, fan_count = excluded.fan_count,
			fan_duty = excluded.fan_duty, fan_mode = excluded.fan_mode, updated_at = excluded.updated_at`,
		s.MinerID, s.HashrateInstant, s.HashrateAvg, s.Hashrate5s, s.Hashrate30m,
		s.HashrateIdeal, s.HashrateNominal, s.PowerConsumption, s.PowerEfficiency,
		s.PCBTempMin, s.PCBTempMax, s.ChipTempMin, s.ChipTempMax,
		s.HWErrors, s.HWErrorPercent, s.Accepted, s.Rejected, s.Stale, s.BestShare, s.FoundBlocks,
		s.DevFeePercent, s.FanCount, s.FanDuty, s.FanMode, s.UpdatedAt)
	return err
}

// =============================================================================
// Chains
// =============================================================================

func (r *SQLiteRepository) GetMinerChains(ctx context.Context, minerID int64) ([]*MinerChain, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, chain_index, serial_number, freq_avg, hashrate_ideal, hashrate_real,
			asic_num, voltage, temp_pcb, temp_chip, temp_pic, hw_errors, eeprom_loaded, updated_at
		FROM miner_chains WHERE miner_id = ? ORDER BY chain_index`, minerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []*MinerChain
	for rows.Next() {
		c := &MinerChain{}
		if err := rows.Scan(&c.ID, &c.MinerID, &c.ChainIndex, &c.SerialNumber, &c.FreqAvg,
			&c.HashrateIdeal, &c.HashrateReal, &c.AsicNum, &c.Voltage,
			&c.TempPCB, &c.TempChip, &c.TempPIC, &c.HWErrors, &c.EepromLoaded, &c.UpdatedAt); err != nil {
			return nil, err
		}
		chains = append(chains, c)
	}
	return chains, rows.Err()
}

func (r *SQLiteRepository) UpsertMinerChain(ctx context.Context, c *MinerChain) error {
	c.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_chains (miner_id, chain_index, serial_number, freq_avg, hashrate_ideal, hashrate_real,
			asic_num, voltage, temp_pcb, temp_chip, temp_pic, hw_errors, eeprom_loaded, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id, chain_index) DO UPDATE SET
			serial_number = excluded.serial_number, freq_avg = excluded.freq_avg,
			hashrate_ideal = excluded.hashrate_ideal, hashrate_real = excluded.hashrate_real,
			asic_num = excluded.asic_num, voltage = excluded.voltage, temp_pcb = excluded.temp_pcb,
			temp_chip = excluded.temp_chip, temp_pic = excluded.temp_pic, hw_errors = excluded.hw_errors,
			eeprom_loaded = excluded.eeprom_loaded, updated_at = excluded.updated_at`,
		c.MinerID, c.ChainIndex, c.SerialNumber, c.FreqAvg, c.HashrateIdeal, c.HashrateReal,
		c.AsicNum, c.Voltage, c.TempPCB, c.TempChip, c.TempPIC, c.HWErrors, c.EepromLoaded, c.UpdatedAt)
	return err
}

func (r *SQLiteRepository) DeleteMinerChains(ctx context.Context, minerID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM miner_chains WHERE miner_id = ?", minerID)
	return err
}

// =============================================================================
// Pools
// =============================================================================

func (r *SQLiteRepository) GetMinerPools(ctx context.Context, minerID int64) ([]*MinerPool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, pool_index, url, user, password, status, priority,
			accepted, rejected, stale, discarded, difficulty, diff_accepted,
			asic_boost, ping, pool_type, updated_at
		FROM miner_pools WHERE miner_id = ? ORDER BY pool_index`, minerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*MinerPool
	for rows.Next() {
		p := &MinerPool{}
		if err := rows.Scan(&p.ID, &p.MinerID, &p.PoolIndex, &p.URL, &p.User, &p.Password,
			&p.Status, &p.Priority, &p.Accepted, &p.Rejected, &p.Stale, &p.Discarded,
			&p.Difficulty, &p.DiffA, &p.ASICBoost, &p.Ping, &p.PoolType, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}
	return pools, rows.Err()
}

func (r *SQLiteRepository) UpsertMinerPool(ctx context.Context, p *MinerPool) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_pools (miner_id, pool_index, url, user, password, status, priority,
			accepted, rejected, stale, discarded, difficulty, diff_accepted,
			asic_boost, ping, pool_type, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id, pool_index) DO UPDATE SET
			url = excluded.url, user = excluded.user, password = excluded.password,
			status = excluded.status, priority = excluded.priority, accepted = excluded.accepted,
			rejected = excluded.rejected, stale = excluded.stale, discarded = excluded.discarded,
			difficulty = excluded.difficulty, diff_accepted = excluded.diff_accepted,
			asic_boost = excluded.asic_boost, ping = excluded.ping, pool_type = excluded.pool_type,
			updated_at = excluded.updated_at`,
		p.MinerID, p.PoolIndex, p.URL, p.User, p.Password, p.Status, p.Priority,
		p.Accepted, p.Rejected, p.Stale, p.Discarded, p.Difficulty, p.DiffA,
		p.ASICBoost, p.Ping, p.PoolType, p.UpdatedAt)
	return err
}

func (r *SQLiteRepository) DeleteMinerPools(ctx context.Context, minerID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM miner_pools WHERE miner_id = ?", minerID)
	return err
}

// =============================================================================
// Fans
// =============================================================================

func (r *SQLiteRepository) GetMinerFans(ctx context.Context, minerID int64) ([]*MinerFan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, fan_index, rpm, duty_cycle, status, updated_at
		FROM miner_fans WHERE miner_id = ? ORDER BY fan_index`, minerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fans []*MinerFan
	for rows.Next() {
		f := &MinerFan{}
		if err := rows.Scan(&f.ID, &f.MinerID, &f.FanIndex, &f.RPM, &f.DutyCycle, &f.Status, &f.UpdatedAt); err != nil {
			return nil, err
		}
		fans = append(fans, f)
	}
	return fans, rows.Err()
}

func (r *SQLiteRepository) UpsertMinerFan(ctx context.Context, f *MinerFan) error {
	f.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_fans (miner_id, fan_index, rpm, duty_cycle, status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id, fan_index) DO UPDATE SET
			rpm = excluded.rpm, duty_cycle = excluded.duty_cycle, status = excluded.status,
			updated_at = excluded.updated_at`,
		f.MinerID, f.FanIndex, f.RPM, f.DutyCycle, f.Status, f.UpdatedAt)
	return err
}

func (r *SQLiteRepository) DeleteMinerFans(ctx context.Context, minerID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM miner_fans WHERE miner_id = ?", minerID)
	return err
}

// =============================================================================
// Metrics
// =============================================================================

func (r *SQLiteRepository) InsertMinerMetric(ctx context.Context, m *MinerMetric) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_metrics (miner_id, timestamp, hashrate, power_consumption, pcb_temp_max, chip_temp_max, fan_duty)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.MinerID, m.Timestamp, m.Hashrate, m.PowerConsumption, m.PCBTempMax, m.ChipTempMax, m.FanDuty)
	if err != nil {
		return err
	}
	m.ID, _ = result.LastInsertId()
	return nil
}

func (r *SQLiteRepository) GetMinerMetrics(ctx context.Context, minerID int64, from, to time.Time) ([]*MinerMetric, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, timestamp, hashrate, power_consumption, pcb_temp_max, chip_temp_max, fan_duty
		FROM miner_metrics WHERE miner_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp`, minerID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*MinerMetric
	for rows.Next() {
		m := &MinerMetric{}
		if err := rows.Scan(&m.ID, &m.MinerID, &m.Timestamp, &m.Hashrate,
			&m.PowerConsumption, &m.PCBTempMax, &m.ChipTempMax, &m.FanDuty); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *SQLiteRepository) DeleteOldMetrics(ctx context.Context, minerID int64, before time.Time) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM miner_metrics WHERE miner_id = ? AND timestamp < ?", minerID, before)
	return err
}

// =============================================================================
// Autotune Presets
// =============================================================================

func (r *SQLiteRepository) GetAutotunePresets(ctx context.Context, minerID int64) ([]*AutotunePreset, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, name, pretty_name, status, modded_psu_required,
			target_power, target_hashrate, voltage, frequency, is_current, updated_at
		FROM autotune_presets WHERE miner_id = ? ORDER BY name`, minerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []*AutotunePreset
	for rows.Next() {
		p := &AutotunePreset{}
		if err := rows.Scan(&p.ID, &p.MinerID, &p.Name, &p.PrettyName, &p.Status, &p.ModdedPSURequired,
			&p.TargetPower, &p.TargetHashrate, &p.Voltage, &p.Frequency, &p.IsCurrent, &p.UpdatedAt); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (r *SQLiteRepository) UpsertAutotunePreset(ctx context.Context, p *AutotunePreset) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO autotune_presets (miner_id, name, pretty_name, status, modded_psu_required,
			target_power, target_hashrate, voltage, frequency, is_current, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(miner_id, name) DO UPDATE SET
			pretty_name = excluded.pretty_name, status = excluded.status,
			modded_psu_required = excluded.modded_psu_required, target_power = excluded.target_power,
			target_hashrate = excluded.target_hashrate, voltage = excluded.voltage,
			frequency = excluded.frequency, is_current = excluded.is_current, updated_at = excluded.updated_at`,
		p.MinerID, p.Name, p.PrettyName, p.Status, p.ModdedPSURequired,
		p.TargetPower, p.TargetHashrate, p.Voltage, p.Frequency, p.IsCurrent, p.UpdatedAt)
	return err
}

func (r *SQLiteRepository) DeleteAutotunePresets(ctx context.Context, minerID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM autotune_presets WHERE miner_id = ?", minerID)
	return err
}

func (r *SQLiteRepository) SetCurrentAutotunePreset(ctx context.Context, minerID int64, presetName string) error {
	// Clear all current flags
	_, err := r.db.ExecContext(ctx, "UPDATE autotune_presets SET is_current = 0 WHERE miner_id = ?", minerID)
	if err != nil {
		return err
	}
	// Set the new current preset
	_, err = r.db.ExecContext(ctx, "UPDATE autotune_presets SET is_current = 1 WHERE miner_id = ? AND name = ?", minerID, presetName)
	return err
}

// =============================================================================
// Notes
// =============================================================================

func (r *SQLiteRepository) GetMinerNotes(ctx context.Context, minerID int64) ([]*MinerNote, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, miner_id, key, value, updated_at
		FROM miner_notes WHERE miner_id = ? ORDER BY key`, minerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*MinerNote
	for rows.Next() {
		n := &MinerNote{}
		if err := rows.Scan(&n.ID, &n.MinerID, &n.Key, &n.Value, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (r *SQLiteRepository) GetMinerNote(ctx context.Context, minerID int64, key string) (*MinerNote, error) {
	n := &MinerNote{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, miner_id, key, value, updated_at
		FROM miner_notes WHERE miner_id = ? AND key = ?`, minerID, key).Scan(
		&n.ID, &n.MinerID, &n.Key, &n.Value, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return n, err
}

func (r *SQLiteRepository) UpsertMinerNote(ctx context.Context, n *MinerNote) error {
	n.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO miner_notes (miner_id, key, value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(miner_id, key) DO UPDATE SET
			value = excluded.value, updated_at = excluded.updated_at`,
		n.MinerID, n.Key, n.Value, n.UpdatedAt)
	return err
}

func (r *SQLiteRepository) DeleteMinerNote(ctx context.Context, minerID int64, key string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM miner_notes WHERE miner_id = ? AND key = ?", minerID, key)
	return err
}

// Ensure SQLiteRepository implements Repository
var _ Repository = (*SQLiteRepository)(nil)
