// Package database provides SQLite storage for miner data.
package database

import (
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Miner represents a mining device in the database.
// This is the core entity that other tables reference.
type Miner struct {
	ID              int64            `json:"id"`
	IPAddress       string           `json:"ip_address"`
	MACAddress      string           `json:"mac_address"`
	Hostname        string           `json:"hostname"`
	SerialNumber    string           `json:"serial_number"`
	FirmwareType    miner.FirmwareType `json:"firmware_type"`
	FirmwareVersion string           `json:"firmware_version"`
	Model           string           `json:"model"`
	MinerType       string           `json:"miner_type"` // Full name e.g. "Antminer S19"
	Algorithm       string           `json:"algorithm"`
	Platform        string           `json:"platform"`        // VNish: "xil"
	HRMeasure       string           `json:"hr_measure"`      // Hashrate unit e.g. "GH/s", "TH/s"
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	LastSeenAt      time.Time        `json:"last_seen_at"`
}

// MinerNetwork represents network configuration for a miner.
type MinerNetwork struct {
	ID           int64     `json:"id"`
	MinerID      int64     `json:"miner_id"`
	DHCP         bool      `json:"dhcp"`
	IPAddress    string    `json:"ip_address"`
	Netmask      string    `json:"netmask"`
	Gateway      string    `json:"gateway"`
	DNSServers   string    `json:"dns_servers"` // Comma-separated or JSON array
	NetDevice    string    `json:"net_device"`  // e.g., "eth0"
	UpdatedAt    time.Time `json:"updated_at"`
}

// MinerHardware represents hardware specifications for a miner.
type MinerHardware struct {
	ID             int64     `json:"id"`
	MinerID        int64     `json:"miner_id"`
	NumChains      int       `json:"num_chains"`
	ChipsPerChain  int       `json:"chips_per_chain"`
	TotalAsicCount int       `json:"total_asic_count"`
	// Voltage limits (mV)
	MinVoltage     int       `json:"min_voltage"`
	MaxVoltage     int       `json:"max_voltage"`
	DefaultVoltage int       `json:"default_voltage"`
	// Frequency limits (MHz)
	MinFreq        int       `json:"min_freq"`
	MaxFreq        int       `json:"max_freq"`
	DefaultFreq    int       `json:"default_freq"`
	// Cooling limits
	MinFanPWM      int       `json:"min_fan_pwm"`
	MinTargetTemp  int       `json:"min_target_temp"`
	MaxTargetTemp  int       `json:"max_target_temp"`
	FanCount       int       `json:"fan_count"`
	// PSU info
	PSUModel       string    `json:"psu_model"`
	PSUSerial      string    `json:"psu_serial"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MinerStatus represents the current operational status of a miner.
type MinerStatus struct {
	ID              int64     `json:"id"`
	MinerID         int64     `json:"miner_id"`
	State           string    `json:"state"`           // "running", "stopped", "failure"
	StateTime       int       `json:"state_time"`      // Time in current state (seconds)
	Description     string    `json:"description"`     // Human-readable status
	FailureCode     int       `json:"failure_code"`    // Error code if in failure state
	UptimeSeconds   int       `json:"uptime_seconds"`
	Unlocked        bool      `json:"unlocked"`        // VNish: unlocked for modifications
	RestartRequired bool      `json:"restart_required"`
	RebootRequired  bool      `json:"reboot_required"`
	FindMiner       bool      `json:"find_miner"`      // LED blink status
	// Stock firmware status checks
	RateStatus      string    `json:"rate_status"`     // "s", "w", "e"
	NetworkStatus   string    `json:"network_status"`
	FansStatus      string    `json:"fans_status"`
	TempStatus      string    `json:"temp_status"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MinerSummary represents current mining performance snapshot.
type MinerSummary struct {
	ID               int64     `json:"id"`
	MinerID          int64     `json:"miner_id"`
	// Hashrate metrics (in native unit specified by HRMeasure)
	HashrateInstant  float64   `json:"hashrate_instant"`
	HashrateAvg      float64   `json:"hashrate_avg"`
	Hashrate5s       float64   `json:"hashrate_5s"`
	Hashrate30m      float64   `json:"hashrate_30m"`
	HashrateIdeal    float64   `json:"hashrate_ideal"`
	HashrateNominal  float64   `json:"hashrate_nominal"`
	// Power metrics
	PowerConsumption int       `json:"power_consumption"` // Watts
	PowerEfficiency  float64   `json:"power_efficiency"`  // J/TH
	// Temperature metrics
	PCBTempMin       int       `json:"pcb_temp_min"`
	PCBTempMax       int       `json:"pcb_temp_max"`
	ChipTempMin      int       `json:"chip_temp_min"`
	ChipTempMax      int       `json:"chip_temp_max"`
	// Error metrics
	HWErrors         int       `json:"hw_errors"`
	HWErrorPercent   float64   `json:"hw_error_percent"`
	// Share metrics
	Accepted         int       `json:"accepted"`
	Rejected         int       `json:"rejected"`
	Stale            int       `json:"stale"`
	BestShare        int64     `json:"best_share"`
	FoundBlocks      int       `json:"found_blocks"`
	// VNish specific
	DevFeePercent    float64   `json:"devfee_percent"`
	// Fan metrics
	FanCount         int       `json:"fan_count"`
	FanDuty          int       `json:"fan_duty"`
	FanMode          string    `json:"fan_mode"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// MinerChain represents a hash board (chain) in a miner.
type MinerChain struct {
	ID            int64     `json:"id"`
	MinerID       int64     `json:"miner_id"`
	ChainIndex    int       `json:"chain_index"`
	SerialNumber  string    `json:"serial_number"`
	// Performance
	FreqAvg       int       `json:"freq_avg"`      // MHz
	HashrateIdeal float64   `json:"hashrate_ideal"`
	HashrateReal  float64   `json:"hashrate_real"`
	// Hardware
	AsicNum       int       `json:"asic_num"`
	Voltage       int       `json:"voltage"`       // mV (VNish)
	// Temperature (may be arrays, stored as JSON or max values)
	TempPCB       int       `json:"temp_pcb"`
	TempChip      int       `json:"temp_chip"`
	TempPIC       int       `json:"temp_pic"`      // Stock firmware
	// Errors
	HWErrors      int       `json:"hw_errors"`
	// Status
	EepromLoaded  bool      `json:"eeprom_loaded"` // Stock firmware
	UpdatedAt     time.Time `json:"updated_at"`
}

// MinerPool represents a mining pool configuration.
type MinerPool struct {
	ID         int64     `json:"id"`
	MinerID    int64     `json:"miner_id"`
	PoolIndex  int       `json:"pool_index"`   // 0, 1, 2
	URL        string    `json:"url"`
	User       string    `json:"user"`
	Password   string    `json:"password"`
	Status     string    `json:"status"`       // "Alive", "Dead", "working"
	Priority   int       `json:"priority"`
	// Share statistics
	Accepted   int       `json:"accepted"`
	Rejected   int       `json:"rejected"`
	Stale      int       `json:"stale"`
	Discarded  int       `json:"discarded"`
	// Difficulty
	Difficulty string    `json:"difficulty"`
	DiffA      float64   `json:"diff_accepted"`
	// Other
	ASICBoost  bool      `json:"asic_boost"`   // VNish
	Ping       int       `json:"ping"`         // VNish: latency in ms
	PoolType   string    `json:"pool_type"`    // VNish: "DevFee", etc.
	UpdatedAt  time.Time `json:"updated_at"`
}

// MinerFan represents a fan in a miner.
type MinerFan struct {
	ID        int64     `json:"id"`
	MinerID   int64     `json:"miner_id"`
	FanIndex  int       `json:"fan_index"`
	RPM       int       `json:"rpm"`
	DutyCycle int       `json:"duty_cycle"` // 0-100%
	Status    string    `json:"status"`     // "ok", "failed"
	UpdatedAt time.Time `json:"updated_at"`
}

// MinerMetric represents a historical time-series data point.
type MinerMetric struct {
	ID               int64     `json:"id"`
	MinerID          int64     `json:"miner_id"`
	Timestamp        time.Time `json:"timestamp"`
	Hashrate         float64   `json:"hashrate"`
	PowerConsumption int       `json:"power_consumption"`
	PCBTempMax       int       `json:"pcb_temp_max"`
	ChipTempMax      int       `json:"chip_temp_max"`
	FanDuty          int       `json:"fan_duty"`
}

// AutotunePreset represents a VNish autotune preset.
type AutotunePreset struct {
	ID                int64     `json:"id"`
	MinerID           int64     `json:"miner_id"`
	Name              string    `json:"name"`        // e.g., "1100", "1300"
	PrettyName        string    `json:"pretty_name"` // e.g., "1100 watt ~ 53 TH"
	Status            string    `json:"status"`      // "tuned", "untuned"
	ModdedPSURequired bool      `json:"modded_psu_required"`
	TargetPower       int       `json:"target_power"`    // Watts
	TargetHashrate    float64   `json:"target_hashrate"` // TH/s
	Voltage           int       `json:"voltage"`         // mV
	Frequency         int       `json:"frequency"`       // MHz
	IsCurrent         bool      `json:"is_current"`      // Is this the active preset
	UpdatedAt         time.Time `json:"updated_at"`
}

// MinerNote represents a key-value note stored on a miner (VNish).
type MinerNote struct {
	ID        int64     `json:"id"`
	MinerID   int64     `json:"miner_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
