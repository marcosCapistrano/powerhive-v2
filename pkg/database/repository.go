package database

import (
	"context"
	"time"
)

// MinerFilter defines filtering and sorting options for listing miners.
type MinerFilter struct {
	// Filters
	MinerType    string // Filter by model/type (e.g., "Antminer S19")
	FirmwareType string // Filter by firmware ("vnish", "stock")
	OnlineStatus string // "online", "offline", "all" (default: "all")

	// Sorting
	SortBy    string // "ip", "model", "hashrate", "power", "efficiency", "temp", "uptime", "last_seen"
	SortOrder string // "asc", "desc" (default: "desc")
}

// Repository defines the interface for miner data storage.
type Repository interface {
	// Database lifecycle
	Close() error

	// Miner CRUD
	CreateMiner(ctx context.Context, m *Miner) error
	GetMiner(ctx context.Context, id int64) (*Miner, error)
	GetMinerByIP(ctx context.Context, ip string) (*Miner, error)
	GetMinerByMAC(ctx context.Context, mac string) (*Miner, error)
	ListMiners(ctx context.Context) ([]*Miner, error)
	ListMinersFiltered(ctx context.Context, filter MinerFilter) ([]*Miner, error)
	UpdateMiner(ctx context.Context, m *Miner) error
	DeleteMiner(ctx context.Context, id int64) error
	UpsertMinerByIP(ctx context.Context, m *Miner) error
	UpsertMinerByMAC(ctx context.Context, m *Miner) error
	SetMinerOnlineStatus(ctx context.Context, id int64, online bool) error
	MarkAllMinersOffline(ctx context.Context) error
	GetDistinctMinerTypes(ctx context.Context) ([]string, error)

	// Network
	GetMinerNetwork(ctx context.Context, minerID int64) (*MinerNetwork, error)
	UpsertMinerNetwork(ctx context.Context, n *MinerNetwork) error

	// Hardware
	GetMinerHardware(ctx context.Context, minerID int64) (*MinerHardware, error)
	UpsertMinerHardware(ctx context.Context, h *MinerHardware) error

	// Status
	GetMinerStatus(ctx context.Context, minerID int64) (*MinerStatus, error)
	UpsertMinerStatus(ctx context.Context, s *MinerStatus) error

	// Summary
	GetMinerSummary(ctx context.Context, minerID int64) (*MinerSummary, error)
	UpsertMinerSummary(ctx context.Context, s *MinerSummary) error
	ZeroMinerSummary(ctx context.Context, minerID int64) error

	// Chains
	GetMinerChains(ctx context.Context, minerID int64) ([]*MinerChain, error)
	UpsertMinerChain(ctx context.Context, c *MinerChain) error
	DeleteMinerChains(ctx context.Context, minerID int64) error

	// Pools
	GetMinerPools(ctx context.Context, minerID int64) ([]*MinerPool, error)
	UpsertMinerPool(ctx context.Context, p *MinerPool) error
	DeleteMinerPools(ctx context.Context, minerID int64) error

	// Fans
	GetMinerFans(ctx context.Context, minerID int64) ([]*MinerFan, error)
	UpsertMinerFan(ctx context.Context, f *MinerFan) error
	DeleteMinerFans(ctx context.Context, minerID int64) error

	// Metrics (time-series)
	InsertMinerMetric(ctx context.Context, m *MinerMetric) error
	GetMinerMetrics(ctx context.Context, minerID int64, from, to time.Time) ([]*MinerMetric, error)
	GetAggregatedMetrics(ctx context.Context, from, to time.Time) ([]*AggregatedMetric, error)
	GetAggregatedMetricsForMiners(ctx context.Context, minerIDs []int64, from, to time.Time) ([]*AggregatedMetric, error)
	DeleteOldMetrics(ctx context.Context, minerID int64, before time.Time) error

	// Fan metrics (per-fan time-series)
	InsertFanMetrics(ctx context.Context, metrics []*FanMetric) error
	GetFanMetrics(ctx context.Context, minerID int64, from, to time.Time) ([]*FanMetric, error)
	DeleteOldFanMetrics(ctx context.Context, minerID int64, before time.Time) error

	// Autotune presets (VNish)
	GetAutotunePresets(ctx context.Context, minerID int64) ([]*AutotunePreset, error)
	UpsertAutotunePreset(ctx context.Context, p *AutotunePreset) error
	DeleteAutotunePresets(ctx context.Context, minerID int64) error
	SetCurrentAutotunePreset(ctx context.Context, minerID int64, presetName string) error

	// Notes (VNish)
	GetMinerNotes(ctx context.Context, minerID int64) ([]*MinerNote, error)
	GetMinerNote(ctx context.Context, minerID int64, key string) (*MinerNote, error)
	UpsertMinerNote(ctx context.Context, n *MinerNote) error
	DeleteMinerNote(ctx context.Context, minerID int64, key string) error

	// Log Sessions
	GetCurrentLogSession(ctx context.Context, minerID int64) (*MinerLogSession, error)
	GetLogSessionByBootTime(ctx context.Context, minerID int64, bootTime time.Time) (*MinerLogSession, error)
	GetLogSessions(ctx context.Context, minerID int64) ([]*MinerLogSession, error)
	CreateLogSession(ctx context.Context, session *MinerLogSession) error
	EndLogSession(ctx context.Context, sessionID int64, endTime time.Time, reason string) error

	// Logs
	InsertLogs(ctx context.Context, logs []*MinerLog) error
	GetSessionLogs(ctx context.Context, sessionID int64, logType string, limit, offset int) ([]*MinerLog, error)
	GetLastLogTime(ctx context.Context, sessionID int64, logType string) (*time.Time, error)
	GetLogCount(ctx context.Context, sessionID int64, logType string) (int, error)
}

// MinerWithDetails contains a miner with all related data.
type MinerWithDetails struct {
	Miner    *Miner
	Network  *MinerNetwork
	Hardware *MinerHardware
	Status   *MinerStatus
	Summary  *MinerSummary
	Chains   []*MinerChain
	Pools    []*MinerPool
	Fans     []*MinerFan
}

// GetMinerWithDetails retrieves a miner with all related data.
func GetMinerWithDetails(ctx context.Context, repo Repository, minerID int64) (*MinerWithDetails, error) {
	m, err := repo.GetMiner(ctx, minerID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	details := &MinerWithDetails{Miner: m}

	details.Network, _ = repo.GetMinerNetwork(ctx, minerID)
	details.Hardware, _ = repo.GetMinerHardware(ctx, minerID)
	details.Status, _ = repo.GetMinerStatus(ctx, minerID)
	details.Summary, _ = repo.GetMinerSummary(ctx, minerID)
	details.Chains, _ = repo.GetMinerChains(ctx, minerID)
	details.Pools, _ = repo.GetMinerPools(ctx, minerID)
	details.Fans, _ = repo.GetMinerFans(ctx, minerID)

	return details, nil
}
