// Package stock provides a client for Bitmain Antminer stock firmware API.
package stock

// SystemInfo contains system information from get_system_info.cgi.
type SystemInfo struct {
	// Network info
	MinerType  string `json:"minertype"`
	NetType    string `json:"nettype"`
	NetDevice  string `json:"netdevice"`
	MACAddr    string `json:"macaddr"`
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"ipaddress"`
	Netmask    string `json:"netmask"`
	Gateway    string `json:"gateway"`
	DNSServers string `json:"dnsservers"`

	// Time info (S19/older models)
	CurTime     string `json:"curtime"`
	Uptime      string `json:"uptime"`
	LoadAverage string `json:"loadaverage"`

	// Memory info (S19/older models)
	MemTotal   int `json:"mem_total"`
	MemUsed    int `json:"mem_used"`
	MemFree    int `json:"mem_free"`
	MemBuffers int `json:"mem_buffers"`
	MemCached  int `json:"mem_cached"`

	// System info
	SystemMode              string `json:"system_mode"`
	AntHWV1                 string `json:"ant_hwv1"`
	AntHWV2                 string `json:"ant_hwv2"`
	AntHWV3                 string `json:"ant_hwv3"`
	AntHWV4                 string `json:"ant_hwv4"`
	SystemKernelVersion     string `json:"system_kernel_version"`
	SystemFilesystemVersion string `json:"system_filesystem_version"`
	CGMinerVersion          string `json:"cgminer_version"`

	// KS5/newer model fields
	FirmwareType string `json:"firmware_type"` // e.g., "Release"
	Algorithm    string `json:"Algorithm"`     // e.g., "KHeavyHash", "sha256d" (capital A in API)
	Serinum      string `json:"serinum"`       // Hardware serial number (misspelled in API)
}

// MinerStatus contains mining status from get_miner_status.cgi.
type MinerStatus struct {
	Summary Summary `json:"summary"`
	Pools   []Pool  `json:"pools"`
	Devs    []Dev   `json:"devs"`
}

// Summary contains aggregated mining metrics.
type Summary struct {
	Elapsed     int     `json:"elapsed"`
	GHS5s       float64 `json:"ghs5s"`
	GHSav       float64 `json:"ghsav"`
	FoundBlocks int     `json:"foundblocks"`
	GetWorks    int     `json:"getworks"`
	Accepted    int     `json:"accepted"`
	Rejected    int     `json:"rejected"`
	HWErrors    int     `json:"hw"`
	Utility     float64 `json:"utility"`
	Discarded   int     `json:"discarded"`
	Stale       int     `json:"stale"`
	LocalWork   int     `json:"localwork"`
	WU          float64 `json:"wu"`
	DiffA       float64 `json:"diffa"`
	DiffR       float64 `json:"diffr"`
	DiffS       float64 `json:"diffs"`
	BestShare   int64   `json:"bestshare"`
}

// Pool contains pool configuration and statistics.
type Pool struct {
	URL         string  `json:"url"`
	User        string  `json:"user"`
	Status      string  `json:"status"`
	Priority    int     `json:"priority"`
	Quota       int     `json:"quota"`
	LongPoll    string  `json:"longpoll"`
	GetWorks    int     `json:"getworks"`
	Accepted    int     `json:"accepted"`
	Rejected    int     `json:"rejected"`
	Discarded   int     `json:"discarded"`
	Stale       int     `json:"stale"`
	DiffA       float64 `json:"diffa"`
	DiffR       float64 `json:"diffr"`
	DiffS       float64 `json:"diffs"`
	LastShareTime int64 `json:"lastsharetime"`
	Diff        string  `json:"diff"`
	Diff1Shares int64   `json:"diff1shares"`
}

// Dev contains device/chain information.
type Dev struct {
	Index       int     `json:"index"`
	Enabled     string  `json:"enabled"`
	Status      string  `json:"status"`
	Temperature float64 `json:"temperature"`
	ChipFreq    int     `json:"chip_freq"`
	FanSpeed    int     `json:"fan_speed"`
	Accepted    int     `json:"accepted"`
	Rejected    int     `json:"rejected"`
	HWErrors    int     `json:"hw"`
	Hashrate    float64 `json:"hashrate"`
}

// MinerConfig contains miner configuration from get_miner_conf.cgi.
type MinerConfig struct {
	Pools []PoolConfig `json:"pools"`

	// Algorithm preset (KS5/newer models)
	Algo string `json:"algo"` // e.g., "ks5_2382"

	// Miner settings
	BitmainFanCtrl   bool   `json:"bitmain-fan-ctrl"`
	BitmainFanPWM    string `json:"bitmain-fan-pwm"`
	BitmainNoBeeper  bool   `json:"bitmain-nobeeper"`  // S19/older models
	BitmainFreq      string `json:"bitmain-freq"`
	BitmainVoltage   string `json:"bitmain-voltage"`
	BitmainCCDelay   string `json:"bitmain-ccdelay"`   // S19/older models
	BitmainPWTH      string `json:"bitmain-pwth"`      // S19/older models
	BitmainWorkMode  string `json:"bitmain-work-mode"`
	BitmainFreqLevel string `json:"bitmain-freq-level"`
}

// PoolConfig contains pool configuration.
type PoolConfig struct {
	URL  string `json:"url"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

// =============================================================================
// New API types for stats.cgi and summary.cgi (KS5, newer models)
// =============================================================================

// APIStatus contains the status header from stats.cgi/summary.cgi responses.
type APIStatus struct {
	Status     string `json:"STATUS"`
	When       int64  `json:"when"`
	Msg        string `json:"Msg"`
	APIVersion string `json:"api_version"`
}

// APIInfo contains miner info from stats.cgi/summary.cgi responses.
type APIInfo struct {
	MinerVersion string `json:"miner_version"`
	CompileTime  string `json:"CompileTime"`
	Type         string `json:"type"`
}

// StatsResponse is the response from stats.cgi endpoint.
type StatsResponse struct {
	Status APIStatus   `json:"STATUS"`
	Info   APIInfo     `json:"INFO"`
	Stats  []StatsData `json:"STATS"`
}

// StatsData contains detailed mining statistics.
type StatsData struct {
	Elapsed   int       `json:"elapsed"`
	Rate5s    float64   `json:"rate_5s"`
	RateAvg   float64   `json:"rate_avg"`
	Rate30m   float64   `json:"rate_30m"`
	RateIdeal float64   `json:"rate_ideal"`
	RateUnit  string    `json:"rate_unit"`
	ChainNum  int       `json:"chain_num"`
	FanNum    int       `json:"fan_num"`
	Fan       []int     `json:"fan"`
	HWPTotal  float64   `json:"hwp_total"`
	Chain     []Chain   `json:"chain"`
}

// Chain contains per-chain information from stats.cgi.
type Chain struct {
	Index       int       `json:"index"`
	FreqAvg     int       `json:"freq_avg"`
	RateIdeal   float64   `json:"rate_ideal"`
	RateReal    float64   `json:"rate_real"`
	AsicNum     int       `json:"asic_num"`
	Asic        string    `json:"asic"`
	TempPIC     []int     `json:"temp_pic"`
	TempPCB     []int     `json:"temp_pcb"`
	TempChip    []int     `json:"temp_chip"`
	HW          int       `json:"hw"`
	EepromLoaded bool     `json:"eeprom_loaded"`
	SN          string    `json:"sn"`
}

// SummaryResponse is the response from summary.cgi endpoint.
type SummaryResponse struct {
	Status  APIStatus     `json:"STATUS"`
	Info    APIInfo       `json:"INFO"`
	Summary []SummaryData `json:"SUMMARY"`
}

// SummaryData contains summary mining data.
type SummaryData struct {
	Elapsed   int            `json:"elapsed"`
	Rate5s    float64        `json:"rate_5s"`
	RateAvg   float64        `json:"rate_avg"`
	Rate30m   float64        `json:"rate_30m"`
	RateIdeal float64        `json:"rate_ideal"`
	RateUnit  string         `json:"rate_unit"`
	HWAll     int            `json:"hw_all"`
	BestShare int64          `json:"bestshare"`
	Status    []StatusItem   `json:"status"`
}

// StatusItem contains individual status checks.
type StatusItem struct {
	Type   string `json:"type"`   // "rate", "network", "fans", "temp"
	Status string `json:"status"` // "s" = success, "w" = warning, "e" = error
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
}

// =============================================================================
// pools.cgi response types
// =============================================================================

// PoolsResponse is the response from pools.cgi endpoint.
type PoolsResponse struct {
	Status APIStatus  `json:"STATUS"`
	Info   APIInfo    `json:"INFO"`
	Pools  []PoolData `json:"POOLS"`
}

// PoolData contains pool status from pools.cgi (different from PoolConfig).
type PoolData struct {
	Index     int     `json:"index"`
	URL       string  `json:"url"`
	User      string  `json:"user"`
	Status    string  `json:"status"`   // "Alive", "Dead"
	Priority  int     `json:"priority"`
	GetWorks  int     `json:"getworks"`
	Accepted  int     `json:"accepted"`
	Rejected  int     `json:"rejected"`
	Discarded int     `json:"discarded"`
	Stale     int     `json:"stale"`
	Diff      string  `json:"diff"`    // Current difficulty as string
	Diff1     int     `json:"diff1"`   // Difficulty 1 shares
	DiffA     float64 `json:"diffa"`   // Accepted difficulty
	DiffR     int     `json:"diffr"`   // Rejected difficulty
	DiffS     int     `json:"diffs"`   // Stale difficulty
	LSDiff    int     `json:"lsdiff"`  // Last share difficulty
	LSTime    string  `json:"lstime"`  // Last share time
}

// =============================================================================
// Network info types
// =============================================================================

// NetworkInfo contains network configuration from get_network_info.cgi.
type NetworkInfo struct {
	NetType        string `json:"nettype"`
	NetDevice      string `json:"netdevice"`
	MACAddr        string `json:"macaddr"`
	IPAddress      string `json:"ipaddress"`
	Netmask        string `json:"netmask"`
	ConfNetType    string `json:"conf_nettype"`
	ConfHostname   string `json:"conf_hostname"`
	ConfIPAddress  string `json:"conf_ipaddress"`
	ConfNetmask    string `json:"conf_netmask"`
	ConfGateway    string `json:"conf_gateway"`
	ConfDNSServers string `json:"conf_dnsservers"`
}

// =============================================================================
// Control endpoint types
// =============================================================================

// BlinkStatus contains LED blink status from get_blink_status.cgi.
type BlinkStatus struct {
	Blink bool `json:"blink"`
}

// ConfigResponse is the response from POST configuration endpoints.
type ConfigResponse struct {
	Stats string `json:"stats"` // "success" or "error"
	Code  string `json:"code"`  // e.g., "M000", "N001"
	Msg   string `json:"msg"`   // e.g., "OK!", "Hostname invalid!"
}
