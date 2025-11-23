package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/discovery"
	"github.com/powerhive/powerhive-v2/pkg/miner"
)

// Balancer is the main power balancing daemon.
type Balancer struct {
	repo       *Repository
	aggregator *Aggregator
	controller *Controller
	strategy   *Strategy
	cooldowns  *CooldownManager
	scanner    *discovery.Scanner

	cfg   *Config
	state BalancerState
	mu    sync.RWMutex

	// Track when we entered recovery margin for hysteresis
	recoveryEnteredAt *time.Time

	// Current status for dashboard
	status *SystemStatus
}

// NewBalancer creates a new balancer instance.
func NewBalancer(
	repo *Repository,
	aggregator *Aggregator,
	controller *Controller,
	strategy *Strategy,
	probers []miner.FirmwareProber,
	cfg *Config,
) *Balancer {
	b := &Balancer{
		repo:       repo,
		aggregator: aggregator,
		controller: controller,
		strategy:   strategy,
		cfg:        cfg,
		state:      StateIdle,
		status:     &SystemStatus{State: StateIdle},
	}

	if repo != nil {
		b.cooldowns = NewCooldownManager(repo, cfg.CooldownDuration)
	}

	if len(probers) > 0 {
		b.scanner = discovery.NewScanner(probers)
	}

	return b
}

// Run starts the main balancer loop.
func (b *Balancer) Run(ctx context.Context) error {
	// Start discovery in background
	go b.runDiscoveryLoop(ctx)

	// Main balancing loop
	ticker := time.NewTicker(b.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := b.tick(ctx); err != nil {
				log.Printf("Balancer tick error: %v", err)
			}
		}
	}
}

// tick performs one iteration of the balancing loop.
func (b *Balancer) tick(ctx context.Context) error {
	// 1. Fetch latest energy data
	reading, err := b.aggregator.FetchLatest(ctx)
	if err != nil {
		log.Printf("Failed to fetch energy data: %v", err)
		// If we can't get data, be conservative
		return nil
	}

	// 2. Store the reading
	if err := b.repo.InsertEnergyReading(ctx, reading); err != nil {
		log.Printf("Failed to store reading: %v", err)
	}

	// 3. Clean up settled changes, expired cooldowns, and pending changes for offline miners
	if err := b.repo.ClearSettledChanges(ctx); err != nil {
		log.Printf("Failed to clear settled changes: %v", err)
	}
	if err := b.cooldowns.CleanupExpired(ctx); err != nil {
		log.Printf("Failed to clear cooldowns: %v", err)
	}
	// Remove pending changes for miners that went offline (their changes won't happen)
	if cleared, err := b.repo.ClearPendingChangesForOfflineMiners(ctx); err != nil {
		log.Printf("Failed to clear pending changes for offline miners: %v", err)
	} else if cleared > 0 {
		log.Printf("Cleared %d pending changes for offline miners", cleared)
	}

	// 4. Calculate effective margin (accounting for pending changes)
	pendingDelta, _ := b.repo.SumPendingDelta(ctx)
	pendingDeltaMW := float64(pendingDelta) / 1_000_000.0

	effectiveConsumption := reading.ConsumptionMW - pendingDeltaMW
	effectiveMargin := reading.GenerationMW - effectiveConsumption
	var effectiveMarginPercent float64
	if reading.GenerationMW > 0 {
		effectiveMarginPercent = (effectiveMargin / reading.GenerationMW) * 100
	}

	// 5. Update status
	b.updateStatus(reading, pendingDelta, effectiveMarginPercent)

	// 6. Log current state
	log.Printf("[%s] Gen=%.2fMW Con=%.2fMW Margin=%.1f%% (Eff=%.1f%%) Pending=%dW",
		b.state, reading.GenerationMW, reading.ConsumptionMW,
		reading.MarginPercent, effectiveMarginPercent, pendingDelta)

	// 7. State machine
	return b.runStateMachine(ctx, reading, effectiveMarginPercent, pendingDelta)
}

// runStateMachine executes the state machine logic.
func (b *Balancer) runStateMachine(ctx context.Context, reading *EnergyReading, effectiveMarginPercent float64, pendingDelta int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateIdle:
		return b.handleIdle(ctx, effectiveMarginPercent)

	case StateReducing:
		return b.handleReducing(ctx, reading, effectiveMarginPercent)

	case StateHolding:
		return b.handleHolding(ctx, effectiveMarginPercent, pendingDelta)

	case StateIncreasing:
		return b.handleIncreasing(ctx, reading, effectiveMarginPercent)

	case StateEmergency:
		return b.handleEmergency(ctx, reading, effectiveMarginPercent)
	}

	return nil
}

// handleIdle handles the IDLE state.
func (b *Balancer) handleIdle(ctx context.Context, effectiveMarginPercent float64) error {
	// Check if we need to start reducing
	if effectiveMarginPercent < b.cfg.CriticalMargin {
		log.Printf("Margin %.1f%% below critical (%.0f%%), switching to REDUCING",
			effectiveMarginPercent, b.cfg.CriticalMargin)
		b.state = StateReducing
		return nil
	}

	// Check if we can increase
	if effectiveMarginPercent > b.cfg.RecoveryMargin {
		if b.recoveryEnteredAt == nil {
			now := time.Now()
			b.recoveryEnteredAt = &now
			log.Printf("Margin %.1f%% above recovery (%.0f%%), starting hysteresis timer",
				effectiveMarginPercent, b.cfg.RecoveryMargin)
		} else if time.Since(*b.recoveryEnteredAt) > 2*time.Minute {
			log.Printf("Recovery threshold sustained for 2 minutes, switching to INCREASING")
			b.state = StateIncreasing
			b.recoveryEnteredAt = nil
		}
	} else {
		b.recoveryEnteredAt = nil
	}

	return nil
}

// handleReducing handles the REDUCING state.
func (b *Balancer) handleReducing(ctx context.Context, reading *EnergyReading, effectiveMarginPercent float64) error {
	// Check for emergency
	if effectiveMarginPercent < b.cfg.EmergencyMargin {
		log.Printf("EMERGENCY: Margin %.1f%% below emergency threshold (%.0f%%)",
			effectiveMarginPercent, b.cfg.EmergencyMargin)
		b.state = StateEmergency
		return nil
	}

	// Check if we've reached safe margin
	if effectiveMarginPercent >= b.cfg.SafeMargin {
		log.Printf("Reached safe margin (%.1f%% >= %.0f%%), switching to HOLDING",
			effectiveMarginPercent, b.cfg.SafeMargin)
		b.state = StateHolding
		return nil
	}

	// Calculate how much reduction we need
	targetConsumption := reading.GenerationMW * (1 - b.cfg.SafeMargin/100)
	reductionNeededMW := reading.ConsumptionMW - targetConsumption
	reductionNeededW := int(reductionNeededMW * 1_000_000)

	if reductionNeededW <= 0 {
		return nil
	}

	// Get one change to make
	changes, err := b.strategy.CalculateReduction(ctx, reductionNeededW)
	if err != nil {
		return fmt.Errorf("calculate reduction: %w", err)
	}

	if len(changes) == 0 {
		log.Printf("No miners available for reduction")
		return nil
	}

	// Execute first change only (sequential in normal mode)
	return b.executeChange(ctx, changes[0], "reduce", reading.MarginPercent)
}

// handleHolding handles the HOLDING state.
func (b *Balancer) handleHolding(ctx context.Context, effectiveMarginPercent float64, pendingDelta int) error {
	// Check for emergency
	if effectiveMarginPercent < b.cfg.EmergencyMargin {
		log.Printf("EMERGENCY while holding")
		b.state = StateEmergency
		return nil
	}

	// Check if we need to reduce more
	if effectiveMarginPercent < b.cfg.CriticalMargin {
		log.Printf("Margin dropped below critical while holding, switching to REDUCING")
		b.state = StateReducing
		return nil
	}

	// Check if we can go to IDLE (all changes settled and margin is good)
	if pendingDelta == 0 && effectiveMarginPercent >= b.cfg.SafeMargin {
		log.Printf("All changes settled and margin stable, switching to IDLE")
		b.state = StateIdle
		return nil
	}

	return nil
}

// handleIncreasing handles the INCREASING state.
func (b *Balancer) handleIncreasing(ctx context.Context, reading *EnergyReading, effectiveMarginPercent float64) error {
	// Check if margin dropped
	if effectiveMarginPercent < b.cfg.SafeMargin {
		log.Printf("Margin dropped below safe during increase, switching to REDUCING")
		b.state = StateReducing
		return nil
	}

	// Calculate how much we can increase while maintaining safe margin
	targetConsumption := reading.GenerationMW * (1 - b.cfg.SafeMargin/100)
	increaseRoomMW := targetConsumption - reading.ConsumptionMW
	increaseRoomW := int(increaseRoomMW * 1_000_000)

	if increaseRoomW <= 0 {
		log.Printf("No room to increase, switching to IDLE")
		b.state = StateIdle
		return nil
	}

	// Get one change to make (conservative increase)
	changes, err := b.strategy.CalculateIncrease(ctx, increaseRoomW)
	if err != nil {
		return fmt.Errorf("calculate increase: %w", err)
	}

	if len(changes) == 0 {
		log.Printf("No miners available for increase, switching to IDLE")
		b.state = StateIdle
		return nil
	}

	// Execute first change only with longer spacing
	return b.executeChange(ctx, changes[0], "increase", reading.MarginPercent)
}

// handleEmergency handles the EMERGENCY state.
func (b *Balancer) handleEmergency(ctx context.Context, reading *EnergyReading, effectiveMarginPercent float64) error {
	// Check if we've recovered from emergency
	if effectiveMarginPercent >= b.cfg.CriticalMargin {
		log.Printf("Recovered from emergency (margin %.1f%%), switching to REDUCING",
			effectiveMarginPercent)
		b.state = StateReducing
		return nil
	}

	// Calculate aggressive reduction
	targetConsumption := reading.GenerationMW * (1 - b.cfg.SafeMargin/100)
	reductionNeededMW := reading.ConsumptionMW - targetConsumption
	reductionNeededW := int(reductionNeededMW * 1_000_000)

	if reductionNeededW <= 0 {
		return nil
	}

	// Get multiple changes in emergency mode
	changes, err := b.strategy.CalculateReduction(ctx, reductionNeededW)
	if err != nil {
		return fmt.Errorf("calculate emergency reduction: %w", err)
	}

	if len(changes) == 0 {
		log.Printf("EMERGENCY: No miners available for reduction!")
		return nil
	}

	// Execute multiple changes in parallel (up to MaxParallelEmergency)
	count := len(changes)
	if count > b.cfg.MaxParallelEmergency {
		count = b.cfg.MaxParallelEmergency
	}

	log.Printf("EMERGENCY: Executing %d parallel changes", count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(change *PresetChange) {
			defer wg.Done()
			if err := b.executeChange(ctx, change, "emergency", reading.MarginPercent); err != nil {
				log.Printf("Emergency change failed: %v", err)
			}
		}(changes[i])
	}
	wg.Wait()

	return nil
}

// executeChange executes a single preset change.
func (b *Balancer) executeChange(ctx context.Context, change *PresetChange, reason string, margin float64) error {
	miner := change.Miner
	log.Printf("Changing %s from %s to %s (delta: %dW, reason: %s)",
		miner.Miner.IPAddress, change.FromPreset.Name, change.ToPreset.Name,
		change.ExpectedDeltaW, reason)

	// Execute the change
	err := b.controller.SetPreset(ctx, miner.Miner.IPAddress, change.ToPreset.Name)

	// Log the change
	logEntry := &ChangeLog{
		MinerID:        &miner.Miner.ID,
		MinerIP:        miner.Miner.IPAddress,
		ModelName:      miner.Model.Name,
		FromPreset:     change.FromPreset.Name,
		ToPreset:       change.ToPreset.Name,
		ExpectedDeltaW: change.ExpectedDeltaW,
		Reason:         reason,
		MarginAtTime:   margin,
		IssuedAt:       time.Now(),
		Success:        err == nil,
	}
	if err != nil {
		logEntry.ErrorMessage = err.Error()
		// Mark miner as offline since we couldn't reach it
		if markErr := b.repo.SetMinerOnlineStatus(ctx, miner.Miner.ID, false); markErr != nil {
			log.Printf("Failed to mark miner %s offline: %v", miner.Miner.IPAddress, markErr)
		} else {
			log.Printf("Marked miner %s as offline due to connection failure", miner.Miner.IPAddress)
		}
	}
	if logErr := b.repo.InsertChangeLog(ctx, logEntry); logErr != nil {
		log.Printf("Failed to log change: %v", logErr)
	}

	if err != nil {
		return fmt.Errorf("set preset on %s: %w", miner.Miner.IPAddress, err)
	}

	// Set cooldown
	if err := b.cooldowns.SetCooldown(ctx, miner.Miner.ID); err != nil {
		log.Printf("Failed to set cooldown: %v", err)
	}

	// Record pending change
	pending := &PendingChange{
		MinerID:        miner.Miner.ID,
		FromPresetID:   &change.FromPreset.ID,
		ToPresetID:     &change.ToPreset.ID,
		ExpectedDeltaW: change.ExpectedDeltaW,
		IssuedAt:       time.Now(),
		SettlesAt:      time.Now().Add(b.cfg.SettleTime),
	}
	if err := b.repo.CreatePendingChange(ctx, pending); err != nil {
		log.Printf("Failed to record pending change: %v", err)
	}

	// Update miner's current preset in database
	if err := b.repo.UpdateMinerPreset(ctx, miner.Miner.ID, change.ToPreset.ID); err != nil {
		log.Printf("Failed to update miner preset: %v", err)
	}

	return nil
}

// updateStatus updates the current system status.
func (b *Balancer) updateStatus(reading *EnergyReading, pendingDelta int, effectiveMarginPercent float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	count, _ := b.repo.CountManagedMiners(context.Background())
	cooldownCount, _ := b.cooldowns.CountActive(context.Background())

	b.status = &SystemStatus{
		State:                  b.state,
		GenerationMW:           reading.GenerationMW,
		ConsumptionMW:          reading.ConsumptionMW,
		MarginMW:               reading.MarginMW,
		MarginPercent:          reading.MarginPercent,
		PendingDeltaW:          pendingDelta,
		EffectiveMarginPercent: effectiveMarginPercent,
		ManagedMinersCount:     count,
		MinersOnCooldown:       cooldownCount,
		GenerosoStatus:         reading.GenerosoStatus,
		NogueiraStatus:         reading.NogueiraStatus,
		LastUpdated:            time.Now(),
	}
}

// GetStatus returns the current system status.
func (b *Balancer) GetStatus() *SystemStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

// GetState returns the current balancer state.
func (b *Balancer) GetState() BalancerState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// runDiscoveryLoop periodically discovers miners on the network.
func (b *Balancer) runDiscoveryLoop(ctx context.Context) {
	// Initial discovery
	if _, err := b.DiscoverMinersOnNetworks(ctx, b.cfg.NetworkCIDRs); err != nil {
		log.Printf("Initial discovery failed: %v", err)
	}

	ticker := time.NewTicker(b.cfg.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := b.DiscoverMinersOnNetworks(ctx, b.cfg.NetworkCIDRs); err != nil {
				log.Printf("Discovery failed: %v", err)
			}
		}
	}
}

// DiscoverMinersOnNetworks scans multiple networks and discovers VNish miners.
func (b *Balancer) DiscoverMinersOnNetworks(ctx context.Context, networks []string) (int, error) {
	if b.scanner == nil {
		return 0, fmt.Errorf("scanner not initialized")
	}

	if len(networks) == 0 {
		return 0, fmt.Errorf("no networks configured")
	}

	// Mark all existing miners as offline before scan
	// Miners found in any network will be marked online again
	if err := b.repo.MarkAllMinersOffline(ctx); err != nil {
		log.Printf("Warning: failed to mark miners offline before scan: %v", err)
	}

	var totalCount int
	for _, network := range networks {
		count, err := b.discoverMinersOnNetwork(ctx, network)
		if err != nil {
			log.Printf("Discovery on %s failed: %v", network, err)
			continue
		}
		totalCount += count
	}

	log.Printf("Total discovered: %d VNish miners across %d networks", totalCount, len(networks))
	return totalCount, nil
}

// discoverMinersOnNetwork scans a single network and discovers VNish miners.
// This is an internal helper - use DiscoverMinersOnNetworks for multi-network scanning.
func (b *Balancer) discoverMinersOnNetwork(ctx context.Context, network string) (int, error) {
	log.Printf("Discovering miners on %s...", network)

	result, err := b.scanner.ScanNetwork(ctx, network)
	if err != nil {
		return 0, fmt.Errorf("scan: %w", err)
	}

	var count int
	for _, m := range result.Miners {
		if m.FirmwareType != miner.FirmwareVNish {
			continue // Only manage VNish miners
		}

		if err := b.processDiscoveredMiner(ctx, m); err != nil {
			log.Printf("Failed to process miner %s: %v", m.IP, err)
			continue
		}
		count++
	}

	log.Printf("Discovered %d VNish miners", count)
	return count, nil
}

// processDiscoveredMiner processes a discovered miner.
func (b *Balancer) processDiscoveredMiner(ctx context.Context, dm discovery.DiscoveredMiner) error {
	// Get miner info for more details
	info, err := b.controller.GetMinerInfo(ctx, dm.IP)
	if err != nil {
		return fmt.Errorf("get info: %w", err)
	}

	// Get or create model
	model, err := b.repo.GetOrCreateModel(ctx, info.Model)
	if err != nil {
		return fmt.Errorf("get/create model: %w", err)
	}

	// Check if we need to fetch presets for this model
	presets, err := b.repo.GetModelPresets(ctx, model.ID)
	if err != nil {
		return fmt.Errorf("get model presets: %w", err)
	}

	if len(presets) == 0 {
		// Fetch presets from miner
		vnishPresets, err := b.controller.GetAvailablePresets(ctx, dm.IP)
		if err != nil {
			log.Printf("Failed to get presets for model %s: %v", model.Name, err)
		} else {
			for i, p := range vnishPresets {
				watts, hashrate := parsePresetPretty(p.Pretty)
				preset := &ModelPreset{
					ModelID:           model.ID,
					Name:              p.Name,
					Watts:             watts,
					HashrateTH:        hashrate,
					DisplayName:       p.Pretty,
					RequiresModdedPSU: p.ModdedPSURequired,
					SortOrder:         i,
				}
				if err := b.repo.UpsertModelPreset(ctx, preset); err != nil {
					log.Printf("Failed to save preset %s: %v", p.Name, err)
				}
			}
		}
	}

	// Get current preset
	currentPresetName, err := b.controller.GetCurrentPreset(ctx, dm.IP)
	if err != nil {
		return fmt.Errorf("get current preset: %w", err)
	}

	// Look up preset ID
	var currentPresetID *int64
	currentPreset, err := b.repo.GetPresetByModelAndName(ctx, model.ID, currentPresetName)
	if err == nil {
		currentPresetID = &currentPreset.ID
	}

	// Upsert miner - use MAC from discovery or from info
	macAddr := dm.MAC
	if macAddr == "" {
		macAddr = info.System.NetworkStatus.MAC
	}

	m := &Miner{
		MACAddress:      macAddr,
		IPAddress:       dm.IP,
		ModelID:         &model.ID,
		FirmwareType:    "vnish",
		CurrentPresetID: currentPresetID,
		IsOnline:        true, // Miner was just discovered/reached
		LastSeen:        func() *time.Time { t := time.Now(); return &t }(),
	}
	if err := b.repo.UpsertMiner(ctx, m); err != nil {
		return fmt.Errorf("upsert miner: %w", err)
	}

	// Ensure balance config exists
	if _, err := b.repo.GetOrCreateBalanceConfig(ctx, m.ID); err != nil {
		return fmt.Errorf("create balance config: %w", err)
	}

	return nil
}

// parsePresetPretty extracts watts and hashrate from a pretty string like "1100 watt ~ 53 TH".
func parsePresetPretty(pretty string) (int, float64) {
	// Try to extract watts from preset name pattern "XXXX watt ~ YY TH"
	re := regexp.MustCompile(`(\d+)\s*watt.*?~\s*([\d.]+)\s*TH`)
	matches := re.FindStringSubmatch(pretty)
	if len(matches) >= 3 {
		watts, _ := strconv.Atoi(matches[1])
		hashrate, _ := strconv.ParseFloat(matches[2], 64)
		return watts, hashrate
	}

	// Fallback: try to get just the number at the start
	re2 := regexp.MustCompile(`^(\d+)`)
	matches2 := re2.FindStringSubmatch(strings.TrimSpace(pretty))
	if len(matches2) >= 2 {
		watts, _ := strconv.Atoi(matches2[1])
		return watts, 0
	}

	return 0, 0
}
