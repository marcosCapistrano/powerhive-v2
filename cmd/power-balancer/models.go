package main

import (
	"time"
)

// BalancerState represents the current state of the power balancer.
type BalancerState string

const (
	StateIdle       BalancerState = "IDLE"
	StateReducing   BalancerState = "REDUCING"
	StateHolding    BalancerState = "HOLDING"
	StateIncreasing BalancerState = "INCREASING"
	StateEmergency  BalancerState = "EMERGENCY"
)

// Model represents a discovered miner model with its preset limits.
type Model struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	MinPresetID *int64     `json:"min_preset_id"`
	MaxPresetID *int64     `json:"max_preset_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`

	// Joined fields (not stored directly)
	MinPreset   *ModelPreset   `json:"min_preset,omitempty"`
	MaxPreset   *ModelPreset   `json:"max_preset,omitempty"`
	Presets     []*ModelPreset `json:"presets,omitempty"`
	MinerCount  int            `json:"miner_count,omitempty"`
	IsConfigured bool          `json:"is_configured"`
}

// ModelPreset represents an available preset for a specific model.
type ModelPreset struct {
	ID               int64   `json:"id"`
	ModelID          int64   `json:"model_id"`
	Name             string  `json:"name"`
	Watts            int     `json:"watts"`
	HashrateTH       float64 `json:"hashrate_th"`
	DisplayName      string  `json:"display_name"`
	RequiresModdedPSU bool   `json:"requires_modded_psu"`
	SortOrder        int     `json:"sort_order"`
}

// Miner represents a discovered miner.
type Miner struct {
	ID              int64      `json:"id"`
	MACAddress      string     `json:"mac_address"`
	IPAddress       string     `json:"ip_address"`
	ModelID         *int64     `json:"model_id"`
	FirmwareType    string     `json:"firmware_type"`
	CurrentPresetID *int64     `json:"current_preset_id"`
	LastSeen        *time.Time `json:"last_seen"`
	CreatedAt       time.Time  `json:"created_at"`

	// Joined fields (not stored directly)
	Model         *Model       `json:"model,omitempty"`
	CurrentPreset *ModelPreset `json:"current_preset,omitempty"`
	Config        *BalanceConfig `json:"config,omitempty"`
	Cooldown      *Cooldown    `json:"cooldown,omitempty"`
}

// BalanceConfig holds per-miner balancing configuration.
type BalanceConfig struct {
	MinerID  int64 `json:"miner_id"`
	Enabled  bool  `json:"enabled"`
	Priority int   `json:"priority"`
	Locked   bool  `json:"locked"`
}

// PendingChange represents a preset change waiting to settle.
type PendingChange struct {
	ID             int64     `json:"id"`
	MinerID        int64     `json:"miner_id"`
	FromPresetID   *int64    `json:"from_preset_id"`
	ToPresetID     *int64    `json:"to_preset_id"`
	ExpectedDeltaW int       `json:"expected_delta_w"`
	IssuedAt       time.Time `json:"issued_at"`
	SettlesAt      time.Time `json:"settles_at"`

	// Joined fields
	FromPreset *ModelPreset `json:"from_preset,omitempty"`
	ToPreset   *ModelPreset `json:"to_preset,omitempty"`
	Miner      *Miner       `json:"miner,omitempty"`
}

// Cooldown represents a miner's cooldown period.
type Cooldown struct {
	MinerID int64     `json:"miner_id"`
	Until   time.Time `json:"until"`
}

// EnergyReading stores a snapshot of generation/consumption data.
type EnergyReading struct {
	ID             int64     `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	GenerationMW   float64   `json:"generation_mw"`
	ConsumptionMW  float64   `json:"consumption_mw"`
	MarginMW       float64   `json:"margin_mw"`
	MarginPercent  float64   `json:"margin_percent"`
	GenerosoMW     float64   `json:"generoso_mw"`
	GenerosoStatus string    `json:"generoso_status"`
	NogueiraMW     float64   `json:"nogueira_mw"`
	NogueiraStatus string    `json:"nogueira_status"`
}

// ChangeLog records preset changes for audit purposes.
type ChangeLog struct {
	ID             int64      `json:"id"`
	MinerID        *int64     `json:"miner_id"`
	MinerIP        string     `json:"miner_ip"`
	ModelName      string     `json:"model_name"`
	FromPreset     string     `json:"from_preset"`
	ToPreset       string     `json:"to_preset"`
	ExpectedDeltaW int        `json:"expected_delta_w"`
	Reason         string     `json:"reason"`
	MarginAtTime   float64    `json:"margin_at_time"`
	IssuedAt       time.Time  `json:"issued_at"`
	Success        bool       `json:"success"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

// MinerWithContext is a miner with all related data for balancing decisions.
type MinerWithContext struct {
	Miner         *Miner
	Model         *Model
	CurrentPreset *ModelPreset
	MinPreset     *ModelPreset
	MaxPreset     *ModelPreset
	Config        *BalanceConfig
	HeadroomWatts int  // current_watts - min_watts
	OnCooldown    bool
	CooldownUntil *time.Time
}

// SystemStatus represents the current state of the power balancer system.
type SystemStatus struct {
	State                 BalancerState `json:"state"`
	GenerationMW          float64       `json:"generation_mw"`
	ConsumptionMW         float64       `json:"consumption_mw"`
	MarginMW              float64       `json:"margin_mw"`
	MarginPercent         float64       `json:"margin_percent"`
	PendingDeltaW         int           `json:"pending_delta_w"`
	EffectiveMarginPercent float64      `json:"effective_margin_percent"`
	ManagedMinersCount    int           `json:"managed_miners_count"`
	MinersOnCooldown      int           `json:"miners_on_cooldown"`
	GenerosoStatus        string        `json:"generoso_status"`
	NogueiraStatus        string        `json:"nogueira_status"`
	LastUpdated           time.Time     `json:"last_updated"`
}
