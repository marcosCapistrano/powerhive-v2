package database

import (
	"strconv"
	"strings"

	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/stock"
)

// StockMapper converts Stock firmware API responses to database models.
type StockMapper struct{}

// NewStockMapper creates a new Stock mapper.
func NewStockMapper() *StockMapper {
	return &StockMapper{}
}

// MapMinerInfo converts Stock SystemInfo to database Miner.
func (m *StockMapper) MapMinerInfo(info *stock.SystemInfo, ipAddress string) *Miner {
	return &Miner{
		IPAddress:       ipAddress,
		MACAddress:      info.MACAddr,
		Hostname:        info.Hostname,
		SerialNumber:    info.Serinum,
		FirmwareType:    miner.FirmwareStock,
		FirmwareVersion: info.SystemFilesystemVersion,
		Model:           extractModelCode(info.MinerType),
		MinerType:       info.MinerType,
		Algorithm:       info.Algorithm,
		HRMeasure:       getHashrateUnitForAlgorithm(info.Algorithm),
	}
}

// extractModelCode extracts model code from miner type string.
// Example: "Antminer KS5 Pro" -> "ks5pro", "Antminer S19" -> "s19"
func extractModelCode(minerType string) string {
	// Remove "Antminer " prefix and convert to lowercase
	model := strings.ToLower(strings.TrimPrefix(minerType, "Antminer "))
	// Remove spaces
	model = strings.ReplaceAll(model, " ", "")
	return model
}

// getHashrateUnitForAlgorithm returns the appropriate hashrate unit for an algorithm.
func getHashrateUnitForAlgorithm(algorithm string) string {
	switch strings.ToLower(algorithm) {
	case "kheavyhash":
		return "H/s" // Kaspa uses H/s (raw hashrate)
	case "sha256d":
		return "TH/s" // Bitcoin uses TH/s
	default:
		return "H/s"
	}
}

// MapNetwork converts Stock SystemInfo/NetworkInfo to database MinerNetwork.
func (m *StockMapper) MapNetwork(info *stock.SystemInfo, netInfo *stock.NetworkInfo, minerID int64) *MinerNetwork {
	n := &MinerNetwork{
		MinerID:   minerID,
		DHCP:      info.NetType == "DHCP",
		IPAddress: info.IPAddress,
		Netmask:   info.Netmask,
		Gateway:   info.Gateway,
		NetDevice: info.NetDevice,
	}

	if info.DNSServers != "" {
		n.DNSServers = info.DNSServers
	}

	// Override with network info if available
	if netInfo != nil {
		n.DHCP = netInfo.NetType == "DHCP"
		n.IPAddress = netInfo.IPAddress
		n.Netmask = netInfo.Netmask
		n.NetDevice = netInfo.NetDevice
	}

	return n
}

// MapHardwareFromStats converts Stock StatsData to database MinerHardware.
// Stock firmware doesn't provide detailed hardware specs like VNish, so we extract what we can.
func (m *StockMapper) MapHardwareFromStats(stats *stock.StatsData, minerID int64) *MinerHardware {
	hw := &MinerHardware{
		MinerID:   minerID,
		NumChains: stats.ChainNum,
		FanCount:  stats.FanNum,
	}

	// Calculate total ASIC count from chains
	totalAsics := 0
	for _, chain := range stats.Chain {
		totalAsics += chain.AsicNum
		if hw.ChipsPerChain == 0 && chain.AsicNum > 0 {
			hw.ChipsPerChain = chain.AsicNum
		}
	}
	hw.TotalAsicCount = totalAsics

	return hw
}

// MapHardwareFromConfig adds config data to hardware.
func (m *StockMapper) MapHardwareFromConfig(hw *MinerHardware, config *stock.MinerConfig) *MinerHardware {
	if config.BitmainVoltage != "" {
		hw.DefaultVoltage, _ = strconv.Atoi(config.BitmainVoltage)
	}
	if config.BitmainFreq != "" {
		hw.DefaultFreq, _ = strconv.Atoi(config.BitmainFreq)
	}
	return hw
}

// MapStatusFromSummary converts Stock SummaryResponse to database MinerStatus.
func (m *StockMapper) MapStatusFromSummary(summary *stock.SummaryResponse, minerID int64) *MinerStatus {
	s := &MinerStatus{
		MinerID: minerID,
		State:   "running", // Assume running if we can query the API
	}

	if len(summary.Summary) > 0 {
		data := summary.Summary[0]
		s.UptimeSeconds = data.Elapsed

		// Map status items
		for _, status := range data.Status {
			switch status.Type {
			case "rate":
				s.RateStatus = status.Status
				if status.Status == "e" {
					s.State = "failure"
					s.Description = status.Msg
				}
			case "network":
				s.NetworkStatus = status.Status
			case "fans":
				s.FansStatus = status.Status
			case "temp":
				s.TempStatus = status.Status
			}
		}
	}

	return s
}

// MapStatusFromOldAPI converts Stock MinerStatus (old API) to database MinerStatus.
func (m *StockMapper) MapStatusFromOldAPI(status *stock.MinerStatus, minerID int64) *MinerStatus {
	s := &MinerStatus{
		MinerID:       minerID,
		State:         "running",
		UptimeSeconds: status.Summary.Elapsed,
	}
	return s
}

// MapSummaryFromStats converts Stock StatsResponse to database MinerSummary.
func (m *StockMapper) MapSummaryFromStats(stats *stock.StatsResponse, minerID int64) *MinerSummary {
	if len(stats.Stats) == 0 {
		return &MinerSummary{MinerID: minerID}
	}

	data := stats.Stats[0]
	s := &MinerSummary{
		MinerID:       minerID,
		Hashrate5s:    data.Rate5s,
		HashrateAvg:   data.RateAvg,
		Hashrate30m:   data.Rate30m,
		HashrateIdeal: data.RateIdeal,
		HWErrorPercent: data.HWPTotal,
		FanCount:      data.FanNum,
	}

	// Calculate temps from chains
	var maxPCBTemp, maxChipTemp int
	var totalHWErrors int
	for _, chain := range data.Chain {
		totalHWErrors += chain.HW
		if len(chain.TempPCB) > 0 {
			for _, t := range chain.TempPCB {
				if t > maxPCBTemp {
					maxPCBTemp = t
				}
			}
		}
		if len(chain.TempChip) > 0 {
			for _, t := range chain.TempChip {
				if t > maxChipTemp {
					maxChipTemp = t
				}
			}
		}
	}
	s.PCBTempMax = maxPCBTemp
	s.ChipTempMax = maxChipTemp
	s.HWErrors = totalHWErrors

	return s
}

// MapSummaryFromSummaryAPI converts Stock SummaryResponse to database MinerSummary.
func (m *StockMapper) MapSummaryFromSummaryAPI(summary *stock.SummaryResponse, minerID int64) *MinerSummary {
	if len(summary.Summary) == 0 {
		return &MinerSummary{MinerID: minerID}
	}

	data := summary.Summary[0]
	return &MinerSummary{
		MinerID:       minerID,
		Hashrate5s:    data.Rate5s,
		HashrateAvg:   data.RateAvg,
		Hashrate30m:   data.Rate30m,
		HashrateIdeal: data.RateIdeal,
		HWErrors:      data.HWAll,
		BestShare:     data.BestShare,
	}
}

// MapSummaryFromOldAPI converts Stock MinerStatus (old API) to database MinerSummary.
func (m *StockMapper) MapSummaryFromOldAPI(status *stock.MinerStatus, minerID int64) *MinerSummary {
	sum := status.Summary
	s := &MinerSummary{
		MinerID:     minerID,
		Hashrate5s:  sum.GHS5s,
		HashrateAvg: sum.GHSav,
		HWErrors:    sum.HWErrors,
		Accepted:    sum.Accepted,
		Rejected:    sum.Rejected,
		Stale:       sum.Stale,
		BestShare:   sum.BestShare,
		FoundBlocks: sum.FoundBlocks,
	}
	return s
}

// MergeSummary merges summary data from multiple sources.
func (m *StockMapper) MergeSummary(base *MinerSummary, stats *MinerSummary) *MinerSummary {
	if base == nil {
		return stats
	}
	if stats == nil {
		return base
	}

	// Prefer stats data for hashrate/temp, but keep share data from summary
	result := &MinerSummary{
		MinerID:          base.MinerID,
		HashrateInstant:  nonZeroFloat(stats.HashrateInstant, base.HashrateInstant),
		HashrateAvg:      nonZeroFloat(stats.HashrateAvg, base.HashrateAvg),
		Hashrate5s:       nonZeroFloat(stats.Hashrate5s, base.Hashrate5s),
		Hashrate30m:      nonZeroFloat(stats.Hashrate30m, base.Hashrate30m),
		HashrateIdeal:    nonZeroFloat(stats.HashrateIdeal, base.HashrateIdeal),
		HashrateNominal:  nonZeroFloat(stats.HashrateNominal, base.HashrateNominal),
		PowerConsumption: nonZeroInt(stats.PowerConsumption, base.PowerConsumption),
		PowerEfficiency:  nonZeroFloat(stats.PowerEfficiency, base.PowerEfficiency),
		PCBTempMin:       nonZeroInt(stats.PCBTempMin, base.PCBTempMin),
		PCBTempMax:       nonZeroInt(stats.PCBTempMax, base.PCBTempMax),
		ChipTempMin:      nonZeroInt(stats.ChipTempMin, base.ChipTempMin),
		ChipTempMax:      nonZeroInt(stats.ChipTempMax, base.ChipTempMax),
		HWErrors:         nonZeroInt(stats.HWErrors, base.HWErrors),
		HWErrorPercent:   nonZeroFloat(stats.HWErrorPercent, base.HWErrorPercent),
		Accepted:         nonZeroInt(stats.Accepted, base.Accepted),
		Rejected:         nonZeroInt(stats.Rejected, base.Rejected),
		Stale:            nonZeroInt(stats.Stale, base.Stale),
		BestShare:        nonZeroInt64(stats.BestShare, base.BestShare),
		FoundBlocks:      nonZeroInt(stats.FoundBlocks, base.FoundBlocks),
		FanCount:         nonZeroInt(stats.FanCount, base.FanCount),
		FanDuty:          nonZeroInt(stats.FanDuty, base.FanDuty),
	}
	return result
}

func nonZeroFloat(a, b float64) float64 {
	if a != 0 {
		return a
	}
	return b
}

func nonZeroInt(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}

func nonZeroInt64(a, b int64) int64 {
	if a != 0 {
		return a
	}
	return b
}

// MapChains converts Stock Chain array to database MinerChain array.
func (m *StockMapper) MapChains(chains []stock.Chain, minerID int64) []*MinerChain {
	result := make([]*MinerChain, len(chains))
	for i, c := range chains {
		chain := &MinerChain{
			MinerID:      minerID,
			ChainIndex:   c.Index,
			SerialNumber: c.SN,
			FreqAvg:      c.FreqAvg,
			HashrateIdeal: c.RateIdeal,
			HashrateReal:  c.RateReal,
			AsicNum:      c.AsicNum,
			HWErrors:     c.HW,
			EepromLoaded: c.EepromLoaded,
		}

		// Get max temp from arrays
		if len(c.TempPCB) > 0 {
			chain.TempPCB = maxInt(c.TempPCB)
		}
		if len(c.TempChip) > 0 {
			chain.TempChip = maxInt(c.TempChip)
		}
		if len(c.TempPIC) > 0 {
			chain.TempPIC = maxInt(c.TempPIC)
		}

		result[i] = chain
	}
	return result
}

func maxInt(arr []int) int {
	if len(arr) == 0 {
		return 0
	}
	max := arr[0]
	for _, v := range arr[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// MapPoolsFromConfig converts Stock PoolConfig array to database MinerPool array.
func (m *StockMapper) MapPoolsFromConfig(pools []stock.PoolConfig, minerID int64) []*MinerPool {
	result := make([]*MinerPool, len(pools))
	for i, p := range pools {
		result[i] = &MinerPool{
			MinerID:   minerID,
			PoolIndex: i,
			URL:       p.URL,
			User:      p.User,
			Password:  p.Pass,
		}
	}
	return result
}

// MapPoolsFromPoolsAPI converts Stock PoolData array to database MinerPool array.
func (m *StockMapper) MapPoolsFromPoolsAPI(pools []stock.PoolData, minerID int64) []*MinerPool {
	result := make([]*MinerPool, len(pools))
	for i, p := range pools {
		result[i] = &MinerPool{
			MinerID:    minerID,
			PoolIndex:  p.Index,
			URL:        p.URL,
			User:       p.User,
			Status:     p.Status,
			Priority:   p.Priority,
			Accepted:   p.Accepted,
			Rejected:   p.Rejected,
			Stale:      p.Stale,
			Discarded:  p.Discarded,
			Difficulty: p.Diff,
			DiffA:      p.DiffA,
		}
	}
	return result
}

// MergePool merges pool config with pool status data.
func (m *StockMapper) MergePool(config *MinerPool, status *MinerPool) *MinerPool {
	if config == nil {
		return status
	}
	if status == nil {
		return config
	}

	return &MinerPool{
		MinerID:    config.MinerID,
		PoolIndex:  config.PoolIndex,
		URL:        config.URL,
		User:       config.User,
		Password:   config.Password,
		Status:     status.Status,
		Priority:   status.Priority,
		Accepted:   status.Accepted,
		Rejected:   status.Rejected,
		Stale:      status.Stale,
		Discarded:  status.Discarded,
		Difficulty: status.Difficulty,
		DiffA:      status.DiffA,
	}
}

// MapFans converts Stock fan data to database MinerFan array.
func (m *StockMapper) MapFans(fanRPMs []int, minerID int64) []*MinerFan {
	result := make([]*MinerFan, len(fanRPMs))
	for i, rpm := range fanRPMs {
		status := "ok"
		if rpm == 0 {
			status = "failed"
		}
		result[i] = &MinerFan{
			MinerID:  minerID,
			FanIndex: i,
			RPM:      rpm,
			Status:   status,
		}
	}
	return result
}

// MapFansFromDevs converts Stock Dev array (old API) to database MinerFan array.
func (m *StockMapper) MapFansFromDevs(devs []stock.Dev, minerID int64) []*MinerFan {
	var fans []*MinerFan
	seenFans := make(map[int]bool)

	for _, dev := range devs {
		if dev.FanSpeed > 0 && !seenFans[dev.Index] {
			status := "ok"
			if dev.FanSpeed == 0 {
				status = "failed"
			}
			fans = append(fans, &MinerFan{
				MinerID:  minerID,
				FanIndex: dev.Index,
				RPM:      dev.FanSpeed,
				Status:   status,
			})
			seenFans[dev.Index] = true
		}
	}
	return fans
}
