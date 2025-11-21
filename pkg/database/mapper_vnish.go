package database

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

// VNishMapper converts VNish API responses to database models.
type VNishMapper struct{}

// NewVNishMapper creates a new VNish mapper.
func NewVNishMapper() *VNishMapper {
	return &VNishMapper{}
}

// MapMinerInfo converts VNish MinerInfo to database Miner.
func (m *VNishMapper) MapMinerInfo(info *vnish.MinerInfo, ipAddress string) *Miner {
	return &Miner{
		IPAddress:       ipAddress,
		MACAddress:      info.System.NetworkStatus.MAC,
		Hostname:        info.System.NetworkStatus.Hostname,
		SerialNumber:    info.Serial,
		FirmwareType:    miner.FirmwareVNish,
		FirmwareVersion: info.FWVersion,
		Model:           info.Model,
		MinerType:       info.Miner,
		Algorithm:       info.Algorithm,
		Platform:        info.Platform,
		HRMeasure:       info.HRMeasure,
	}
}

// MapNetwork converts VNish NetworkStatus to database MinerNetwork.
func (m *VNishMapper) MapNetwork(info *vnish.MinerInfo, minerID int64) *MinerNetwork {
	ns := info.System.NetworkStatus
	return &MinerNetwork{
		MinerID:    minerID,
		DHCP:       ns.DHCP,
		IPAddress:  ns.IP,
		Netmask:    ns.Netmask,
		Gateway:    ns.Gateway,
		DNSServers: strings.Join(ns.DNS, ","),
		NetDevice:  "eth0", // VNish doesn't expose this, assume eth0
	}
}

// MapHardware converts VNish ModelInfo to database MinerHardware.
func (m *VNishMapper) MapHardware(model *vnish.ModelInfo, factoryInfo *vnish.ChainFactoryInfo, minerID int64) *MinerHardware {
	hw := &MinerHardware{
		MinerID:        minerID,
		NumChains:      model.Chain.NumChains,
		ChipsPerChain:  model.Chain.ChipsPerChain,
		TotalAsicCount: model.Chain.NumChains * model.Chain.ChipsPerChain,
		MinVoltage:     model.Overclock.MinVoltage,
		MaxVoltage:     model.Overclock.MaxVoltage,
		DefaultVoltage: model.Overclock.DefaultVoltage,
		MinFreq:        model.Overclock.MinFreq,
		MaxFreq:        model.Overclock.MaxFreq,
		DefaultFreq:    model.Overclock.DefaultFreq,
		MinFanPWM:      model.Cooling.MinFanPWM,
		MinTargetTemp:  model.Cooling.MinTargetTemp,
		MaxTargetTemp:  model.Cooling.MaxTargetTemp,
		FanCount:       model.Cooling.FanMinCount.Default,
	}

	if factoryInfo != nil {
		if factoryInfo.PSUModel != nil {
			hw.PSUModel = *factoryInfo.PSUModel
		}
		if factoryInfo.PSUSerial != nil {
			hw.PSUSerial = *factoryInfo.PSUSerial
		}
	}

	return hw
}

// MapStatus converts VNish MinerStatus to database MinerStatus.
func (m *VNishMapper) MapStatus(status *vnish.MinerStatus, info *vnish.MinerInfo, minerID int64) *MinerStatus {
	s := &MinerStatus{
		MinerID:         minerID,
		State:           status.MinerState,
		StateTime:       status.MinerStateTime,
		Description:     status.Description,
		FailureCode:     status.FailureCode,
		Unlocked:        status.Unlocked,
		RestartRequired: status.RestartRequired,
		RebootRequired:  status.RebootRequired,
		FindMiner:       status.FindMiner,
	}

	// Parse uptime from info if available (format: "H:MM" or "D:HH:MM")
	if info != nil {
		s.UptimeSeconds = parseVNishUptime(info.System.Uptime)
	}

	return s
}

// parseVNishUptime converts VNish uptime string to seconds.
// Format can be "H:MM", "HH:MM", "D:HH:MM", etc.
func parseVNishUptime(uptime string) int {
	parts := strings.Split(uptime, ":")
	switch len(parts) {
	case 2: // H:MM
		hours, _ := strconv.Atoi(parts[0])
		mins, _ := strconv.Atoi(parts[1])
		return hours*3600 + mins*60
	case 3: // D:HH:MM
		days, _ := strconv.Atoi(parts[0])
		hours, _ := strconv.Atoi(parts[1])
		mins, _ := strconv.Atoi(parts[2])
		return days*86400 + hours*3600 + mins*60
	default:
		return 0
	}
}

// MapSummary converts VNish Summary to database MinerSummary.
func (m *VNishMapper) MapSummary(summary *vnish.Summary, minerID int64) *MinerSummary {
	ms := summary.Miner
	return &MinerSummary{
		MinerID:          minerID,
		HashrateInstant:  ms.InstantHashrate,
		HashrateAvg:      ms.AverageHashrate,
		Hashrate5s:       ms.HRRealtime,
		HashrateIdeal:    ms.HRStock,
		HashrateNominal:  ms.HRNominal,
		PowerConsumption: ms.PowerConsumption,
		PowerEfficiency:  ms.PowerEfficiency,
		PCBTempMin:       ms.PCBTemp.Min,
		PCBTempMax:       ms.PCBTemp.Max,
		ChipTempMin:      ms.ChipTemp.Min,
		ChipTempMax:      ms.ChipTemp.Max,
		HWErrors:         ms.HWErrors,
		HWErrorPercent:   ms.HWErrorsPercent,
		BestShare:        int64(ms.BestShare),
		FoundBlocks:      ms.FoundBlocks,
		DevFeePercent:    ms.DevFeePercent,
		FanCount:         ms.Cooling.FanNum,
		FanDuty:          ms.Cooling.FanDuty,
		FanMode:          ms.Cooling.Settings.Mode.Name,
	}
}

// MapChains converts VNish Chain array to database MinerChain array.
func (m *VNishMapper) MapChains(chains []vnish.Chain, minerID int64) []*MinerChain {
	result := make([]*MinerChain, len(chains))
	for i, c := range chains {
		result[i] = &MinerChain{
			MinerID:       minerID,
			ChainIndex:    c.ID,
			FreqAvg:       c.Frequency,
			HashrateReal:  c.Hashrate,
			AsicNum:       c.ChipCount,
			Voltage:       c.Voltage,
			TempPCB:       c.PCBTemp,
			TempChip:      c.ChipTemp,
			HWErrors:      c.HWErrors,
		}
	}
	return result
}

// MapPools converts VNish Pool array to database MinerPool array.
func (m *VNishMapper) MapPools(pools []vnish.Pool, minerID int64) []*MinerPool {
	result := make([]*MinerPool, len(pools))
	for i, p := range pools {
		result[i] = &MinerPool{
			MinerID:    minerID,
			PoolIndex:  p.ID,
			URL:        p.URL,
			User:       p.User,
			Status:     p.Status,
			Accepted:   p.Accepted,
			Rejected:   p.Rejected,
			Stale:      p.Stale,
			Difficulty: p.Diff,
			DiffA:      p.DiffA,
			ASICBoost:  p.ASICBoost,
			Ping:       p.Ping,
			PoolType:   p.PoolType,
		}
	}
	return result
}

// MapFans converts VNish Cooling data to database MinerFan array.
func (m *VNishMapper) MapFans(cooling vnish.Cooling, minerID int64) []*MinerFan {
	result := make([]*MinerFan, len(cooling.Fans))
	for i, rpm := range cooling.Fans {
		status := "ok"
		if rpm == 0 {
			status = "failed"
		}
		result[i] = &MinerFan{
			MinerID:   minerID,
			FanIndex:  i,
			RPM:       rpm,
			DutyCycle: cooling.FanDuty,
			Status:    status,
		}
	}
	return result
}

// MapMetric converts a VNish MetricPoint to database MinerMetric.
func (m *VNishMapper) MapMetric(metric vnish.MetricPoint, minerID int64) *MinerMetric {
	return &MinerMetric{
		MinerID:          minerID,
		Timestamp:        time.Unix(metric.Time, 0),
		Hashrate:         metric.Data.Hashrate,
		PowerConsumption: metric.Data.PowerConsumption,
		PCBTempMax:       metric.Data.PCBMaxTemp,
		ChipTempMax:      metric.Data.ChipMaxTemp,
		FanDuty:          metric.Data.FanDuty,
	}
}

// MapMetrics converts VNish Metrics to database MinerMetric array.
func (m *VNishMapper) MapMetrics(metrics *vnish.Metrics, minerID int64) []*MinerMetric {
	result := make([]*MinerMetric, len(metrics.Metrics))
	for i, mp := range metrics.Metrics {
		result[i] = m.MapMetric(mp, minerID)
	}
	return result
}

// MapAutotunePresets converts VNish AutotunePreset array to database AutotunePreset array.
func (m *VNishMapper) MapAutotunePresets(presets []vnish.AutotunePreset, currentPresetName string, minerID int64) []*AutotunePreset {
	result := make([]*AutotunePreset, len(presets))
	for i, p := range presets {
		preset := &AutotunePreset{
			MinerID:           minerID,
			Name:              p.Name,
			PrettyName:        p.Pretty,
			Status:            p.Status,
			ModdedPSURequired: p.ModdedPSURequired,
			IsCurrent:         p.Name == currentPresetName,
		}

		// Parse power and hashrate from pretty name if available
		// Format: "1100 watt ~ 53 TH" or similar
		preset.TargetPower, preset.TargetHashrate = parsePrettyPreset(p.Pretty)

		// Get voltage/frequency from tune settings if available
		if p.TuneSettings != nil {
			preset.Voltage = p.TuneSettings.Volt
			preset.Frequency = p.TuneSettings.Freq
		}

		result[i] = preset
	}
	return result
}

// parsePrettyPreset extracts power (watts) and hashrate (TH) from preset pretty name.
// Example: "1100 watt ~ 53 TH" -> (1100, 53.0)
func parsePrettyPreset(pretty string) (int, float64) {
	// Match patterns like "1100 watt ~ 53 TH"
	re := regexp.MustCompile(`(\d+)\s*watt\s*~\s*(\d+)\s*TH`)
	matches := re.FindStringSubmatch(pretty)
	if len(matches) == 3 {
		power, _ := strconv.Atoi(matches[1])
		hashrate, _ := strconv.ParseFloat(matches[2], 64)
		return power, hashrate
	}
	return 0, 0
}

// MapNote converts VNish Note to database MinerNote.
func (m *VNishMapper) MapNote(note vnish.Note, minerID int64) *MinerNote {
	return &MinerNote{
		MinerID: minerID,
		Key:     note.Key,
		Value:   note.Value,
	}
}
