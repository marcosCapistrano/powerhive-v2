package vnish

import "encoding/json"

// UnlockRequest is the request body for authentication.
type UnlockRequest struct {
	Password string `json:"pw"`
}

// UnlockResponse contains the bearer token from authentication.
type UnlockResponse struct {
	Token string `json:"token"`
}

// NetworkStatus contains network configuration details.
type NetworkStatus struct {
	MAC      string   `json:"mac"`
	DHCP     bool     `json:"dhcp"`
	IP       string   `json:"ip"`
	Netmask  string   `json:"netmask"`
	Gateway  string   `json:"gateway"`
	DNS      []string `json:"dns"`
	Hostname string   `json:"hostname"`
}

// SystemInfo contains system-level information.
type SystemInfo struct {
	OS                string        `json:"os"`
	MinerName         string        `json:"miner_name"`
	FileSystemVersion string        `json:"file_system_version"`
	MemTotal          int           `json:"mem_total"`
	MemFree           int           `json:"mem_free"`
	MemFreePercent    int           `json:"mem_free_percent"`
	MemBuf            int           `json:"mem_buf"`
	MemBufPercent     int           `json:"mem_buf_percent"`
	NetworkStatus     NetworkStatus `json:"network_status"`
	Uptime            string        `json:"uptime"`
}

// MinerInfo contains detailed miner information.
type MinerInfo struct {
	Miner       string     `json:"miner"`
	Model       string     `json:"model"`
	FWName      string     `json:"fw_name"`
	FWVersion   string     `json:"fw_version"`
	BuildUUID   string     `json:"build_uuid"`
	BuildName   string     `json:"build_name"`
	Platform    string     `json:"platform"`
	InstallType string     `json:"install_type"`
	BuildTime   string     `json:"build_time"`
	Algorithm   string     `json:"algorithm"`
	HRMeasure   string     `json:"hr_measure"`
	System      SystemInfo `json:"system"`
	Serial      string     `json:"serial"`
}

// ChainTopology contains the chip topology for a chain.
type ChainTopology struct {
	Chips   [][]int `json:"chips"`
	NumCols int     `json:"num_cols"`
	NumRows int     `json:"num_rows"`
}

// ChainSpec contains chain-specific specifications.
type ChainSpec struct {
	ChipsPerChain  int           `json:"chips_per_chain"`
	ChipsPerDomain int           `json:"chips_per_domain"`
	NumChains      int           `json:"num_chains"`
	Topology       ChainTopology `json:"topology"`
}

// CoolingSpec contains cooling specifications.
type CoolingSpec struct {
	MinFanPWM     int `json:"min_fan_pwm"`
	MinTargetTemp int `json:"min_target_temp"`
	MaxTargetTemp int `json:"max_target_temp"`
	FanMinCount   struct {
		Min     int `json:"min"`
		Max     int `json:"max"`
		Default int `json:"default"`
	} `json:"fan_min_count"`
}

// OverclockSpec contains overclocking parameters.
type OverclockSpec struct {
	MaxVoltage         int `json:"max_voltage"`
	MinVoltage         int `json:"min_voltage"`
	DefaultVoltage     int `json:"default_voltage"`
	MaxFreq            int `json:"max_freq"`
	MinFreq            int `json:"min_freq"`
	DefaultFreq        int `json:"default_freq"`
	WarnFreq           int `json:"warn_freq"`
	MaxVoltageStockPSU int `json:"max_voltage_stock_psu"`
}

// ModelInfo contains model-specific information.
type ModelInfo struct {
	FullName    string        `json:"full_name"`
	Model       string        `json:"model"`
	Algorithm   string        `json:"algorithm"`
	Series      string        `json:"series"`
	Platform    string        `json:"platform"`
	InstallType string        `json:"install_type"`
	HRMeasure   string        `json:"hr_measure"`
	Serial      string        `json:"serial"`
	Chain       ChainSpec     `json:"chain"`
	Cooling     CoolingSpec   `json:"cooling"`
	Overclock   OverclockSpec `json:"overclock"`
}

// MinerStatus contains the current miner operational status.
type MinerStatus struct {
	MinerState     string `json:"miner_state"`
	MinerStateTime int    `json:"miner_state_time"`
	Description    string `json:"description"`
	FailureCode    int    `json:"failure_code"`
	FindMiner      bool   `json:"find_miner"`
	RestartRequired bool  `json:"restart_required"`
	RebootRequired  bool  `json:"reboot_required"`
	Unlocked       bool   `json:"unlocked"`
}

// TempRange contains min/max temperature values.
type TempRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Pool contains mining pool information.
type Pool struct {
	ID        int     `json:"id"`
	URL       string  `json:"url"`
	PoolType  string  `json:"pool_type"`
	User      string  `json:"user"`
	Status    string  `json:"status"`
	ASICBoost bool    `json:"asic_boost"`
	Diff      string  `json:"diff"`
	Accepted  int     `json:"accepted"`
	Rejected  int     `json:"rejected"`
	Stale     int     `json:"stale"`
	LSDiff    float64 `json:"ls_diff"`
	LSTime    string  `json:"ls_time"`
	DiffA     float64 `json:"diffa"`
	Ping      int     `json:"ping"`
}

// FanMode contains fan mode settings.
type FanMode struct {
	Name string `json:"name"`
}

// CoolingSettings contains cooling configuration.
type CoolingSettings struct {
	Mode FanMode `json:"mode"`
}

// Fan contains individual fan status (object format from newer VNish versions).
type Fan struct {
	ID     int    `json:"id"`
	RPM    int    `json:"rpm"`
	Status string `json:"status"`
	MaxRPM int    `json:"max_rpm"`
}

// FanData handles both VNish fan response formats:
// - Legacy: array of ints [5400, 5300, ...]
// - Modern: array of objects [{"rpm": 5400, "status": "ok"}, ...]
type FanData []Fan

// UnmarshalJSON implements custom unmarshaling for flexible fan data.
func (f *FanData) UnmarshalJSON(data []byte) error {
	// Try array of objects first (modern format)
	var fans []Fan
	if err := json.Unmarshal(data, &fans); err == nil {
		*f = fans
		return nil
	}

	// Fall back to array of ints (legacy format)
	var rpms []int
	if err := json.Unmarshal(data, &rpms); err != nil {
		return err
	}

	// Convert ints to Fan structs
	*f = make([]Fan, len(rpms))
	for i, rpm := range rpms {
		status := "ok"
		if rpm == 0 {
			status = "failed"
		}
		(*f)[i] = Fan{RPM: rpm, Status: status}
	}
	return nil
}

// Cooling contains cooling status information.
type Cooling struct {
	FanNum   int             `json:"fan_num"`
	Fans     FanData         `json:"fans"`
	Settings CoolingSettings `json:"settings"`
	FanDuty  int             `json:"fan_duty"`
}

// MinerSummary contains the miner summary data.
type MinerSummary struct {
	MinerStatus      MinerStatus `json:"miner_status"`
	MinerType        string      `json:"miner_type"`
	HRStock          float64     `json:"hr_stock"`
	AverageHashrate  float64     `json:"average_hashrate"`
	InstantHashrate  float64     `json:"instant_hashrate"`
	HRRealtime       float64     `json:"hr_realtime"`
	HRNominal        float64     `json:"hr_nominal"`
	HRAverage        float64     `json:"hr_average"`
	PCBTemp          TempRange   `json:"pcb_temp"`
	ChipTemp         TempRange   `json:"chip_temp"`
	PowerConsumption int         `json:"power_consumption"`
	PowerUsage       int         `json:"power_usage"`
	PowerEfficiency  float64     `json:"power_efficiency"`
	HWErrorsPercent  float64     `json:"hw_errors_percent"`
	HRError          float64     `json:"hr_error"`
	HWErrors         int         `json:"hw_errors"`
	DevFeePercent    float64     `json:"devfee_percent"`
	DevFee           float64     `json:"devfee"`
	Pools            []Pool      `json:"pools"`
	Cooling          Cooling     `json:"cooling"`
	Chains           []Chain     `json:"chains"`
	FoundBlocks      int         `json:"found_blocks"`
	BestShare        int         `json:"best_share"`
}

// Summary is the top-level summary response.
type Summary struct {
	Miner MinerSummary `json:"miner"`
}

// ChipStatuses contains chip health status counts.
type ChipStatuses struct {
	Red    int `json:"red"`
	Orange int `json:"orange"`
	Grey   int `json:"grey"`
}

// ChainStatus contains chain operational state.
type ChainStatus struct {
	State string `json:"state"`
}

// Chain contains mining chain (board) information.
type Chain struct {
	ID                 int          `json:"id"`
	Frequency          float64      `json:"frequency"`
	Voltage            float64      `json:"voltage"`
	PowerConsumption   int          `json:"power_consumption"`
	HashrateIdeal      float64      `json:"hashrate_ideal"`
	HashrateRT         float64      `json:"hashrate_rt"`
	HashratePercentage float64      `json:"hashrate_percentage"`
	HRError            float64      `json:"hr_error"`
	HWErrors           int          `json:"hw_errors"`
	PCBTemp            TempRange    `json:"pcb_temp"`
	ChipTemp           TempRange    `json:"chip_temp"`
	ChipStatuses       ChipStatuses `json:"chip_statuses"`
	Status             ChainStatus  `json:"status"`
	// Legacy fields (may not be present in all versions)
	ChipCount int   `json:"chip_count,omitempty"`
	ChipFreqs []int `json:"chip_freqs,omitempty"`
	ChipVolts []int `json:"chip_volts,omitempty"`
}

// ChainFactoryInfo contains factory information for chains.
type ChainFactoryInfo struct {
	HRStock   *float64 `json:"hr_stock"`
	HasPics   *bool    `json:"has_pics"`
	PSUModel  *string  `json:"psu_model"`
	PSUSerial *string  `json:"psu_serial"`
	Chains    []Chain  `json:"chains"`
}

// PresetGlobals contains global preset settings.
type PresetGlobals struct {
	Volt int `json:"volt"`
	Freq int `json:"freq"`
}

// CurrentPreset contains the current autotune preset.
type CurrentPreset struct {
	Name              string        `json:"name"`
	Pretty            string        `json:"pretty"`
	Status            string        `json:"status"`
	ModdedPSURequired bool          `json:"modded_psu_required"`
	Globals           PresetGlobals `json:"globals"`
}

// PresetSwitcher contains preset switcher configuration.
type PresetSwitcher struct {
	Enabled             bool    `json:"enabled"`
	TopPreset           *string `json:"top_preset"`
	DecreaseTemp        int     `json:"decrease_temp"`
	RiseTemp            int     `json:"rise_temp"`
	CheckTime           int     `json:"check_time"`
	AutoChangeTopPreset bool    `json:"autochange_top_preset"`
	IgnoreFanSpeed      bool    `json:"ignore_fan_speed"`
	MinPreset           *string `json:"min_preset"`
	PowerDelta          int     `json:"power_delta"`
}

// PerfSummary contains performance summary with autotune info.
type PerfSummary struct {
	CurrentPreset  CurrentPreset  `json:"current_preset"`
	PresetSwitcher PresetSwitcher `json:"preset_switcher"`
}

// TuneChainSettings contains per-chain tune settings.
type TuneChainSettings struct {
	Freq  int   `json:"freq"`
	Chips []int `json:"chips"`
}

// TuneSettings contains autotune settings.
type TuneSettings struct {
	Hashrate float64             `json:"hashrate"`
	Volt     int                 `json:"volt"`
	Freq     int                 `json:"freq"`
	Chains   []TuneChainSettings `json:"chains"`
	Modified bool                `json:"modified"`
}

// AutotunePreset contains an autotune preset configuration.
type AutotunePreset struct {
	Name              string        `json:"name"`
	Pretty            string        `json:"pretty"`
	Status            string        `json:"status"`
	ModdedPSURequired bool          `json:"modded_psu_required"`
	TuneSettings      *TuneSettings `json:"tune_settings,omitempty"`
}

// MetricData contains a single metric data point.
type MetricData struct {
	Hashrate         float64 `json:"hashrate"`
	PCBMaxTemp       int     `json:"pcb_max_temp"`
	ChipMaxTemp      int     `json:"chip_max_temp"`
	FanDuty          int     `json:"fan_duty"`
	PowerConsumption int     `json:"power_consumption"`
}

// MetricPoint contains a timestamped metric.
type MetricPoint struct {
	Time int64      `json:"time"`
	Data MetricData `json:"data"`
}

// AnnotationData contains annotation information.
type AnnotationData struct {
	Type string `json:"type"`
}

// Annotation contains a timestamped annotation.
type Annotation struct {
	Time int64          `json:"time"`
	Data AnnotationData `json:"data"`
}

// Metrics contains time-series metrics data.
type Metrics struct {
	Timezone    string        `json:"timezone"`
	Metrics     []MetricPoint `json:"metrics"`
	Annotations []Annotation  `json:"annotations"`
}

// Note contains a key-value note.
type Note struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value"`
}

// APIKey contains an API key entry.
type APIKey struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

// APIKeyRequest is the request for creating/deleting API keys.
type APIKeyRequest struct {
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
}

// APIKeyResponse is the response for API key operations.
type APIKeyResponse struct {
	Status string `json:"status"`
}

// MinerMiscSettings contains miscellaneous miner settings.
type MinerMiscSettings struct {
	QuietMode bool `json:"quiet_mode,omitempty"`
}

// MinerSettings contains miner-specific settings.
type MinerSettings struct {
	Misc MinerMiscSettings `json:"misc,omitempty"`
}

// SettingsUpdate is the request body for updating settings.
type SettingsUpdate struct {
	Miner MinerSettings `json:"miner,omitempty"`
}

// SwitchPoolRequest is the request for switching mining pools.
type SwitchPoolRequest struct {
	PoolID int64 `json:"pool_id"`
}

// FindMinerResponse is the response for find-miner LED toggle.
type FindMinerResponse struct {
	On bool `json:"on"`
}

// ErrorResponse contains an error message from the API.
type ErrorResponse struct {
	Err string `json:"err"`
}
