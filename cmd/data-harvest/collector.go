package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/database"
	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/stock"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

// CollectedData holds all data collected from a miner.
type CollectedData struct {
	Miner      *database.Miner
	Network    *database.MinerNetwork
	Hardware   *database.MinerHardware
	Status     *database.MinerStatus
	Summary    *database.MinerSummary
	Chains     []*database.MinerChain
	Pools      []*database.MinerPool
	Fans       []*database.MinerFan
	Metric     *database.MinerMetric
	FanMetrics []*database.FanMetric // Per-fan time-series data
	Presets    []*database.AutotunePreset // VNish only
}

// Collector handles data collection from miners.
type Collector struct {
	vnishMapper *database.VNishMapper
	stockMapper *database.StockMapper
}

// NewCollector creates a new data collector.
func NewCollector() *Collector {
	return &Collector{
		vnishMapper: database.NewVNishMapper(),
		stockMapper: database.NewStockMapper(),
	}
}

// Collect fetches all available data from a miner.
func (c *Collector) Collect(ctx context.Context, client miner.Client, fwType miner.FirmwareType) (*CollectedData, error) {
	switch fwType {
	case miner.FirmwareVNish:
		vnishClient, ok := client.(*vnish.HTTPClient)
		if !ok {
			return nil, fmt.Errorf("expected VNish client, got %T", client)
		}
		return c.collectVNish(ctx, vnishClient)
	case miner.FirmwareStock:
		stockClient, ok := client.(*stock.HTTPClient)
		if !ok {
			return nil, fmt.Errorf("expected Stock client, got %T", client)
		}
		return c.collectStock(ctx, stockClient)
	default:
		return nil, fmt.Errorf("unsupported firmware type: %s", fwType)
	}
}

// collectVNish fetches all data from a VNish miner.
func (c *Collector) collectVNish(ctx context.Context, client *vnish.HTTPClient) (*CollectedData, error) {
	data := &CollectedData{}
	ip := client.Host()

	// Get basic info (public endpoint)
	info, err := client.GetInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}
	data.Miner = c.vnishMapper.MapMinerInfo(info, ip)
	data.Network = c.vnishMapper.MapNetwork(info, 0) // minerID set later

	// Get model info for hardware specs
	model, err := client.GetModel(ctx)
	if err != nil {
		log.Printf("[%s] warning: failed to get model info: %v", ip, err)
	} else {
		// Get factory info for PSU details
		factoryInfo, _ := client.GetChainsFactoryInfo(ctx)
		data.Hardware = c.vnishMapper.MapHardware(model, factoryInfo, 0)
	}

	// Get status (public endpoint)
	status, err := client.GetStatus(ctx)
	if err != nil {
		log.Printf("[%s] warning: failed to get status: %v", ip, err)
	} else {
		data.Status = c.vnishMapper.MapStatus(status, info, 0)
	}

	// Get summary (requires auth)
	summary, err := client.GetSummary(ctx)
	if err != nil {
		log.Printf("[%s] warning: failed to get summary: %v", ip, err)
	} else {
		data.Summary = c.vnishMapper.MapSummary(summary, 0)
		data.Chains = c.vnishMapper.MapChains(summary.Miner.Chains, 0)
		data.Pools = c.vnishMapper.MapPools(summary.Miner.Pools, 0)
		data.Fans = c.vnishMapper.MapFans(summary.Miner.Cooling, 0)

		// Create metric from summary
		now := time.Now()
		data.Metric = &database.MinerMetric{
			Timestamp:        now,
			Hashrate:         summary.Miner.InstantHashrate,
			PowerConsumption: summary.Miner.PowerConsumption,
			PCBTempMax:       summary.Miner.PCBTemp.Max,
			ChipTempMax:      summary.Miner.ChipTemp.Max,
			FanDuty:          summary.Miner.Cooling.FanDuty,
		}

		// Create per-fan metrics from fans
		for _, fan := range data.Fans {
			rpm := fan.RPM
			if fan.Status == "failed" || fan.Status == "error" {
				rpm = -1 // Mark failed fan with -1 RPM
			}
			data.FanMetrics = append(data.FanMetrics, &database.FanMetric{
				FanIndex:  fan.FanIndex,
				Timestamp: now,
				RPM:       rpm,
			})
		}
	}

	// Get autotune presets (requires auth)
	presets, err := client.GetAutotunePresets(ctx)
	if err != nil {
		log.Printf("[%s] warning: failed to get autotune presets: %v", ip, err)
	} else {
		// Get current preset name from perf summary
		currentPreset := ""
		if perfSummary, err := client.GetPerfSummary(ctx); err == nil {
			currentPreset = perfSummary.CurrentPreset.Name
		}
		data.Presets = c.vnishMapper.MapAutotunePresets(presets, currentPreset, 0)
	}

	return data, nil
}

// collectStock fetches all data from a Stock firmware miner.
func (c *Collector) collectStock(ctx context.Context, client *stock.HTTPClient) (*CollectedData, error) {
	data := &CollectedData{}
	ip := client.Host()

	// Get system info
	sysInfo, err := client.GetSystemInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}
	data.Miner = c.stockMapper.MapMinerInfo(sysInfo, ip)

	// Get network info
	netInfo, _ := client.GetNetworkInfo(ctx)
	data.Network = c.stockMapper.MapNetwork(sysInfo, netInfo, 0)

	// Get miner config for pool/hardware info
	config, err := client.GetMinerConfig(ctx)
	if err != nil {
		log.Printf("[%s] warning: failed to get miner config: %v", ip, err)
	}

	// Try new API first (KS5/newer models), fallback to old API (S19)
	statsResp, statsErr := client.GetStats(ctx)
	summaryResp, summaryErr := client.GetSummary(ctx)

	if statsErr == nil && len(statsResp.Stats) > 0 {
		// New API available (KS5/newer)
		statsData := statsResp.Stats[0]

		data.Hardware = c.stockMapper.MapHardwareFromStats(&statsData, 0)
		if config != nil {
			data.Hardware = c.stockMapper.MapHardwareFromConfig(data.Hardware, config)
		}

		data.Summary = c.stockMapper.MapSummaryFromStats(statsResp, 0)
		data.Chains = c.stockMapper.MapChains(statsData.Chain, 0)
		data.Fans = c.stockMapper.MapFans(statsData.Fan, 0)

		// Create metric
		data.Metric = &database.MinerMetric{
			Timestamp:        time.Now(),
			Hashrate:         statsData.RateAvg,
			PCBTempMax:       data.Summary.PCBTempMax,
			ChipTempMax:      data.Summary.ChipTempMax,
		}

		// Merge with summary data if available
		if summaryErr == nil {
			summaryData := c.stockMapper.MapSummaryFromSummaryAPI(summaryResp, 0)
			data.Summary = c.stockMapper.MergeSummary(data.Summary, summaryData)

			// Get status from summary
			data.Status = c.stockMapper.MapStatusFromSummary(summaryResp, 0)
		}

		// Fallback: get status from stats if summary failed
		if data.Status == nil {
			data.Status = &database.MinerStatus{
				State:         "running",
				UptimeSeconds: statsData.Elapsed,
			}
		}
	} else {
		// Try old API (S19/older models)
		minerStatus, err := client.GetMinerStatusFull(ctx)
		if err != nil {
			log.Printf("[%s] warning: failed to get miner status (old API): %v", ip, err)
		} else {
			data.Status = c.stockMapper.MapStatusFromOldAPI(minerStatus, 0)
			data.Summary = c.stockMapper.MapSummaryFromOldAPI(minerStatus, 0)

			// Hardware from devs
			if data.Hardware == nil {
				data.Hardware = &database.MinerHardware{NumChains: len(minerStatus.Devs)}
			}
			if config != nil {
				data.Hardware = c.stockMapper.MapHardwareFromConfig(data.Hardware, config)
			}

			data.Fans = c.stockMapper.MapFansFromDevs(minerStatus.Devs, 0)

			// Create metric
			if data.Summary != nil {
				data.Metric = &database.MinerMetric{
					Timestamp: time.Now(),
					Hashrate:  data.Summary.Hashrate5s,
				}
			}
		}
	}

	// Get pools from pools.cgi
	poolsResp, poolsErr := client.GetPools(ctx)
	if poolsErr == nil && len(poolsResp.Pools) > 0 {
		poolsFromAPI := c.stockMapper.MapPoolsFromPoolsAPI(poolsResp.Pools, 0)

		// Merge with config pools to get passwords
		if config != nil {
			configPools := c.stockMapper.MapPoolsFromConfig(config.Pools, 0)
			data.Pools = make([]*database.MinerPool, len(poolsFromAPI))
			for i, p := range poolsFromAPI {
				var configPool *database.MinerPool
				if i < len(configPools) {
					configPool = configPools[i]
				}
				data.Pools[i] = c.stockMapper.MergePool(configPool, p)
			}
		} else {
			data.Pools = poolsFromAPI
		}
	} else if config != nil {
		// Fallback to config pools only
		data.Pools = c.stockMapper.MapPoolsFromConfig(config.Pools, 0)
	}

	// Set status if not already set
	if data.Status == nil {
		data.Status = &database.MinerStatus{State: "running"}
	}

	// Create per-fan metrics from fans
	now := time.Now()
	for _, fan := range data.Fans {
		rpm := fan.RPM
		if fan.Status == "failed" || fan.Status == "error" {
			rpm = -1 // Mark failed fan with -1 RPM
		}
		data.FanMetrics = append(data.FanMetrics, &database.FanMetric{
			FanIndex:  fan.FanIndex,
			Timestamp: now,
			RPM:       rpm,
		})
	}

	return data, nil
}

// SetMinerID updates all collected data with the miner ID.
func (data *CollectedData) SetMinerID(minerID int64) {
	if data.Miner != nil {
		data.Miner.ID = minerID
	}
	if data.Network != nil {
		data.Network.MinerID = minerID
	}
	if data.Hardware != nil {
		data.Hardware.MinerID = minerID
	}
	if data.Status != nil {
		data.Status.MinerID = minerID
	}
	if data.Summary != nil {
		data.Summary.MinerID = minerID
	}
	for _, c := range data.Chains {
		c.MinerID = minerID
	}
	for _, p := range data.Pools {
		p.MinerID = minerID
	}
	for _, f := range data.Fans {
		f.MinerID = minerID
	}
	if data.Metric != nil {
		data.Metric.MinerID = minerID
	}
	for _, fm := range data.FanMetrics {
		fm.MinerID = minerID
	}
	for _, p := range data.Presets {
		p.MinerID = minerID
	}
}
