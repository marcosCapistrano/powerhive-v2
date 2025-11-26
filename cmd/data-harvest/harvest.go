package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/database"
	"github.com/powerhive/powerhive-v2/pkg/discovery"
	"github.com/powerhive/powerhive-v2/pkg/miner"
	"github.com/powerhive/powerhive-v2/pkg/stock"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

// Harvester orchestrates data collection from miners.
type Harvester struct {
	repo         database.Repository
	detector     *discovery.Detector
	scanner      *discovery.Scanner
	collector    *Collector
	logCollector *LogCollector
	config       *Config
}

// NewHarvester creates a new harvester.
func NewHarvester(repo database.Repository, probers []miner.FirmwareProber, cfg *Config) *Harvester {
	return &Harvester{
		repo:         repo,
		detector:     discovery.NewDetector(probers, discovery.WithDetectorTimeout(cfg.Timeout)),
		scanner:      discovery.NewScanner(probers, discovery.WithTimeout(cfg.Timeout), discovery.WithConcurrency(cfg.Concurrency)),
		collector:    NewCollector(),
		logCollector: NewLogCollector(repo),
		config:       cfg,
	}
}

// HarvestNetwork scans a network and harvests all discovered miners.
// Returns a set of miner IDs that were successfully contacted.
func (h *Harvester) HarvestNetwork(ctx context.Context, cidr string) (map[int64]bool, error) {
	log.Printf("Scanning network %s...", cidr)

	result, err := h.scanner.ScanNetwork(ctx, cidr)
	if err != nil {
		return nil, fmt.Errorf("failed to scan network: %w", err)
	}

	log.Printf("Found %d miners", len(result.Miners))

	// Collect IPs to harvest
	ips := make([]string, len(result.Miners))
	for i, m := range result.Miners {
		ips[i] = m.IP
		log.Printf("  - %s (%s %s)", m.IP, m.Firmware, m.Model)
	}

	return h.HarvestMiners(ctx, ips)
}

// HarvestMiners harvests data from specific miners.
// Returns a set of miner IDs that were successfully contacted.
func (h *Harvester) HarvestMiners(ctx context.Context, ips []string) (map[int64]bool, error) {
	log.Printf("Harvesting %d miners...", len(ips))

	var wg sync.WaitGroup
	sem := make(chan struct{}, h.config.Concurrency)
	var mu sync.Mutex
	var successCount, failCount int
	successIDs := make(map[int64]bool)

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			minerID, err := h.harvestOne(ctx, ip)
			if err != nil {
				log.Printf("[%s] ERROR: %v", ip, err)
				mu.Lock()
				failCount++
				mu.Unlock()
			} else {
				log.Printf("[%s] OK", ip)
				mu.Lock()
				successCount++
				successIDs[minerID] = true
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	log.Printf("Harvest complete: %d succeeded, %d failed", successCount, failCount)
	return successIDs, nil
}

// harvestOne harvests data from a single miner.
// Returns the miner ID on success (for tracking which miners responded).
func (h *Harvester) harvestOne(ctx context.Context, ip string) (int64, error) {
	// Detect firmware and get client
	client, fwType, err := h.detector.GetClient(ctx, ip)
	if err != nil {
		return 0, fmt.Errorf("failed to detect miner: %w", err)
	}

	// Collect data
	data, err := h.collector.Collect(ctx, client, fwType)
	if err != nil {
		return 0, fmt.Errorf("failed to collect data: %w", err)
	}

	// Validate MAC address - required for unique identification
	if data.Miner.MACAddress == "" {
		return 0, fmt.Errorf("miner did not return MAC address, cannot uniquely identify")
	}

	// Check if miner was previously offline (for transition tracking)
	existingMiner, _ := h.repo.GetMinerByMAC(ctx, data.Miner.MACAddress)
	wasOffline := existingMiner != nil && !existingMiner.IsOnline

	// Upsert miner record by MAC address (not IP - IPs can change)
	data.Miner.IsOnline = true // Successfully contacted = online
	if err := h.repo.UpsertMinerByMAC(ctx, data.Miner); err != nil {
		return 0, fmt.Errorf("failed to upsert miner: %w", err)
	}
	minerID := data.Miner.ID

	// Update all collected data with miner ID
	data.SetMinerID(minerID)

	// Update last seen and ensure online status
	data.Miner.LastSeenAt = time.Now()
	data.Miner.IsOnline = true
	if err := h.repo.UpdateMiner(ctx, data.Miner); err != nil {
		log.Printf("[%s] warning: failed to update miner: %v", ip, err)
	}

	// Upsert all related data
	if data.Network != nil {
		if err := h.repo.UpsertMinerNetwork(ctx, data.Network); err != nil {
			log.Printf("[%s] warning: failed to upsert network: %v", ip, err)
		}
	}

	if data.Hardware != nil {
		if err := h.repo.UpsertMinerHardware(ctx, data.Hardware); err != nil {
			log.Printf("[%s] warning: failed to upsert hardware: %v", ip, err)
		}
	}

	if data.Status != nil {
		if err := h.repo.UpsertMinerStatus(ctx, data.Status); err != nil {
			log.Printf("[%s] warning: failed to upsert status: %v", ip, err)
		}
	}

	if data.Summary != nil {
		if err := h.repo.UpsertMinerSummary(ctx, data.Summary); err != nil {
			log.Printf("[%s] warning: failed to upsert summary: %v", ip, err)
		}
	}

	// Upsert chains
	for _, chain := range data.Chains {
		if err := h.repo.UpsertMinerChain(ctx, chain); err != nil {
			log.Printf("[%s] warning: failed to upsert chain %d: %v", ip, chain.ChainIndex, err)
		}
	}

	// Upsert pools
	for _, pool := range data.Pools {
		if err := h.repo.UpsertMinerPool(ctx, pool); err != nil {
			log.Printf("[%s] warning: failed to upsert pool %d: %v", ip, pool.PoolIndex, err)
		}
	}

	// Upsert fans
	for _, fan := range data.Fans {
		if err := h.repo.UpsertMinerFan(ctx, fan); err != nil {
			log.Printf("[%s] warning: failed to upsert fan %d: %v", ip, fan.FanIndex, err)
		}
	}

	// Insert metric (time-series)
	if data.Metric != nil {
		// If miner was offline and is now back online, insert a zero metric first
		// This creates a clear transition point in charts (0 → actual hashrate)
		if wasOffline {
			zeroMetric := &database.MinerMetric{
				MinerID:          minerID,
				Timestamp:        data.Metric.Timestamp.Add(-time.Second), // Just before the real metric
				Hashrate:         0,
				PowerConsumption: 0,
				PCBTempMax:       0,
				ChipTempMax:      0,
				FanDuty:          0,
			}
			if err := h.repo.InsertMinerMetric(ctx, zeroMetric); err != nil {
				log.Printf("[%s] warning: failed to insert transition zero metric: %v", ip, err)
			}
		}
		if err := h.repo.InsertMinerMetric(ctx, data.Metric); err != nil {
			log.Printf("[%s] warning: failed to insert metric: %v", ip, err)
		}
	}

	// Insert fan metrics (time-series per fan)
	if len(data.FanMetrics) > 0 {
		if err := h.repo.InsertFanMetrics(ctx, data.FanMetrics); err != nil {
			log.Printf("[%s] warning: failed to insert fan metrics: %v", ip, err)
		}
	}

	// Upsert autotune presets (VNish only)
	for _, preset := range data.Presets {
		if err := h.repo.UpsertAutotunePreset(ctx, preset); err != nil {
			log.Printf("[%s] warning: failed to upsert preset %s: %v", ip, preset.Name, err)
		}
	}

	// Collect logs (uses uptime to track boot sessions)
	uptimeSeconds := 0
	if data.Status != nil {
		uptimeSeconds = data.Status.UptimeSeconds
	}

	if uptimeSeconds > 0 {
		switch fwType {
		case miner.FirmwareVNish:
			if vnishClient, ok := client.(*vnish.HTTPClient); ok {
				if err := h.logCollector.CollectVNishLogs(ctx, vnishClient, minerID, uptimeSeconds); err != nil {
					log.Printf("[%s] warning: failed to collect logs: %v", ip, err)
				}
			}
		case miner.FirmwareStock:
			if stockClient, ok := client.(*stock.HTTPClient); ok {
				if err := h.logCollector.CollectStockLogs(ctx, stockClient, minerID, uptimeSeconds); err != nil {
					log.Printf("[%s] warning: failed to collect logs: %v", ip, err)
				}
			}
		}
	}

	return minerID, nil
}

// RunDaemon runs continuous harvesting.
func (h *Harvester) RunDaemon(ctx context.Context, networks []string) error {
	log.Printf("Starting daemon mode (interval: %s)", h.config.HarvestInterval)

	ticker := time.NewTicker(h.config.HarvestInterval)
	defer ticker.Stop()

	// Initial harvest
	if err := h.harvestAll(ctx, networks); err != nil {
		log.Printf("Initial harvest error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Daemon stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := h.harvestAll(ctx, networks); err != nil {
				log.Printf("Harvest cycle error: %v", err)
			}
		}
	}
}

// harvestAll harvests from networks and known miners.
// After harvesting, marks miners that didn't respond as offline.
func (h *Harvester) harvestAll(ctx context.Context, networks []string) error {
	log.Printf("Starting harvest cycle at %s", time.Now().Format(time.RFC3339))

	// Track all miners that successfully responded
	successfulMiners := make(map[int64]bool)

	// Scan networks if provided
	for _, network := range networks {
		ids, err := h.HarvestNetwork(ctx, network)
		if err != nil {
			log.Printf("Error scanning %s: %v", network, err)
		} else {
			// Merge successful IDs
			for id := range ids {
				successfulMiners[id] = true
			}
		}
	}

	// Also harvest known miners from database that weren't in the scan
	miners, err := h.repo.ListMiners(ctx)
	if err != nil {
		log.Printf("Error listing miners: %v", err)
		return nil
	}

	// Filter miners not recently seen (might have moved to different network)
	var staleMiners []string
	staleThreshold := time.Now().Add(-24 * time.Hour)
	for _, m := range miners {
		if m.LastSeenAt.Before(staleThreshold) && !successfulMiners[m.ID] {
			staleMiners = append(staleMiners, m.IPAddress)
		}
	}

	if len(staleMiners) > 0 {
		log.Printf("Re-checking %d stale miners...", len(staleMiners))
		ids, _ := h.HarvestMiners(ctx, staleMiners)
		for id := range ids {
			successfulMiners[id] = true
		}
	}

	// Mark miners that didn't respond as offline
	offlineCount := 0
	for _, m := range miners {
		if !successfulMiners[m.ID] && m.IsOnline {
			if err := h.repo.SetMinerOnlineStatus(ctx, m.ID, false); err != nil {
				log.Printf("Warning: failed to mark miner %d offline: %v", m.ID, err)
			} else {
				offlineCount++
				// Zero summary data so totals reflect reality
				if err := h.repo.ZeroMinerSummary(ctx, m.ID); err != nil {
					log.Printf("Warning: failed to zero summary for miner %d: %v", m.ID, err)
				}
				// Insert zero metric so charts show offline period
				zeroMetric := &database.MinerMetric{
					MinerID:          m.ID,
					Timestamp:        time.Now(),
					Hashrate:         0,
					PowerConsumption: 0,
					PCBTempMax:       0,
					ChipTempMax:      0,
					FanDuty:          0,
				}
				if err := h.repo.InsertMinerMetric(ctx, zeroMetric); err != nil {
					log.Printf("Warning: failed to insert zero metric for miner %d: %v", m.ID, err)
				}
			}
		}
	}
	if offlineCount > 0 {
		log.Printf("Marked %d miners as offline", offlineCount)
	}

	return nil
}

// ListMiners lists all known miners from the database.
func (h *Harvester) ListMiners(ctx context.Context) error {
	miners, err := h.repo.ListMiners(ctx)
	if err != nil {
		return fmt.Errorf("failed to list miners: %w", err)
	}

	if len(miners) == 0 {
		fmt.Println("No miners in database")
		return nil
	}

	fmt.Printf("%-8s %-16s %-12s %-20s %-10s %s\n", "STATUS", "IP", "FIRMWARE", "MODEL", "STATE", "LAST SEEN")
	fmt.Println("------------------------------------------------------------------------------------------")

	for _, m := range miners {
		// Get status for state
		status, _ := h.repo.GetMinerStatus(ctx, m.ID)
		state := "unknown"
		if status != nil {
			state = status.State
		}

		onlineStatus := "OFFLINE"
		if m.IsOnline {
			onlineStatus = "ONLINE"
		}

		fmt.Printf("%-8s %-16s %-12s %-20s %-10s %s\n",
			onlineStatus,
			m.IPAddress,
			m.FirmwareType,
			truncate(m.MinerType, 20),
			state,
			m.LastSeenAt.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

// ShowMiner shows detailed information for a specific miner.
func (h *Harvester) ShowMiner(ctx context.Context, ip string) error {
	m, err := h.repo.GetMinerByIP(ctx, ip)
	if err != nil {
		return fmt.Errorf("failed to get miner: %w", err)
	}
	if m == nil {
		return fmt.Errorf("miner not found: %s", ip)
	}

	// Get all related data
	details, err := database.GetMinerWithDetails(ctx, h.repo, m.ID)
	if err != nil {
		return fmt.Errorf("failed to get miner details: %w", err)
	}

	// Print miner info
	fmt.Printf("=== Miner: %s ===\n", m.IPAddress)
	fmt.Printf("Type:       %s\n", m.MinerType)
	fmt.Printf("Model:      %s\n", m.Model)
	fmt.Printf("Firmware:   %s %s\n", m.FirmwareType, m.FirmwareVersion)
	fmt.Printf("Algorithm:  %s\n", m.Algorithm)
	fmt.Printf("MAC:        %s\n", m.MACAddress)
	fmt.Printf("Hostname:   %s\n", m.Hostname)
	fmt.Printf("Serial:     %s\n", m.SerialNumber)
	fmt.Printf("Last Seen:  %s\n", m.LastSeenAt.Format(time.RFC3339))

	// Print status
	if details.Status != nil {
		s := details.Status
		fmt.Printf("\n=== Status ===\n")
		fmt.Printf("State:      %s\n", s.State)
		fmt.Printf("Uptime:     %s\n", formatDuration(time.Duration(s.UptimeSeconds)*time.Second))
		if s.Description != "" {
			fmt.Printf("Info:       %s\n", s.Description)
		}
	}

	// Print summary
	if details.Summary != nil {
		s := details.Summary
		fmt.Printf("\n=== Performance ===\n")
		fmt.Printf("Hashrate:   %.2f %s (avg)\n", s.HashrateAvg, m.HRMeasure)
		fmt.Printf("Power:      %d W\n", s.PowerConsumption)
		if s.PowerEfficiency > 0 {
			fmt.Printf("Efficiency: %.2f J/TH\n", s.PowerEfficiency)
		}
		fmt.Printf("Temp (PCB): %d - %d °C\n", s.PCBTempMin, s.PCBTempMax)
		fmt.Printf("Temp (Chip): %d - %d °C\n", s.ChipTempMin, s.ChipTempMax)
		fmt.Printf("HW Errors:  %d (%.2f%%)\n", s.HWErrors, s.HWErrorPercent)
		fmt.Printf("Fans:       %d @ %d%% duty\n", s.FanCount, s.FanDuty)
	}

	// Print chains
	if len(details.Chains) > 0 {
		fmt.Printf("\n=== Chains ===\n")
		for _, c := range details.Chains {
			fmt.Printf("Chain %d: %.2f %s, %d ASICs, %d MHz, PCB %d°C, Chip %d°C, %d HW errors\n",
				c.ChainIndex, c.HashrateReal, m.HRMeasure, c.AsicNum, c.FreqAvg,
				c.TempPCB, c.TempChip, c.HWErrors)
		}
	}

	// Print pools
	if len(details.Pools) > 0 {
		fmt.Printf("\n=== Pools ===\n")
		for _, p := range details.Pools {
			fmt.Printf("Pool %d: %s (%s) - %s\n", p.PoolIndex, p.URL, p.User, p.Status)
			fmt.Printf("         Accepted: %d, Rejected: %d, Stale: %d\n",
				p.Accepted, p.Rejected, p.Stale)
		}
	}

	return nil
}

// Helper functions

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
