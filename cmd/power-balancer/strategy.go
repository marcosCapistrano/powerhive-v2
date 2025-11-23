package main

import (
	"context"
	"sort"
)

// Strategy handles miner selection and preset targeting logic.
type Strategy struct {
	repo *Repository
	cfg  *Config
}

// PresetChange represents a planned preset change.
type PresetChange struct {
	Miner        *MinerWithContext
	FromPreset   *ModelPreset
	ToPreset     *ModelPreset
	ExpectedDeltaW int // Positive = reduction
}

// NewStrategy creates a new strategy instance.
func NewStrategy(repo *Repository, cfg *Config) *Strategy {
	return &Strategy{
		repo: repo,
		cfg:  cfg,
	}
}

// CalculateReduction calculates which miners to reduce and by how much.
// Returns a list of preset changes that would achieve the target reduction.
func (s *Strategy) CalculateReduction(ctx context.Context, targetReductionW int) ([]*PresetChange, error) {
	// Get all manageable miners (enabled, not locked, configured model)
	miners, err := s.repo.GetManageableMiners(ctx)
	if err != nil {
		return nil, err
	}

	// Filter out miners on cooldown and those with no headroom
	var available []*MinerWithContext
	for _, m := range miners {
		if !m.OnCooldown && m.HeadroomWatts > 0 {
			available = append(available, m)
		}
	}

	// Sort by efficiency ascending (least efficient miners reduced first)
	// This preserves the most hashrate per watt when reducing consumption
	sort.Slice(available, func(i, j int) bool {
		return available[i].Efficiency < available[j].Efficiency
	})

	// Calculate changes to reach target reduction
	var changes []*PresetChange
	remainingReduction := targetReductionW

	for _, miner := range available {
		if remainingReduction <= 0 {
			break
		}

		// Get available presets for this miner's model
		presets, err := s.repo.GetModelPresets(ctx, miner.Model.ID)
		if err != nil {
			continue
		}

		// Find a target preset that reduces power
		targetPreset := s.findReductionPreset(miner, presets, remainingReduction)
		if targetPreset == nil {
			continue
		}

		delta := miner.CurrentPreset.Watts - targetPreset.Watts
		if delta <= 0 {
			continue
		}

		changes = append(changes, &PresetChange{
			Miner:          miner,
			FromPreset:     miner.CurrentPreset,
			ToPreset:       targetPreset,
			ExpectedDeltaW: delta,
		})

		remainingReduction -= delta
	}

	return changes, nil
}

// CalculateIncrease calculates which miners to increase and by how much.
// Returns a list of preset changes that would achieve the target increase.
func (s *Strategy) CalculateIncrease(ctx context.Context, targetIncreaseW int) ([]*PresetChange, error) {
	// Get all manageable miners
	miners, err := s.repo.GetManageableMiners(ctx)
	if err != nil {
		return nil, err
	}

	// Filter out miners on cooldown and those already at max
	var available []*MinerWithContext
	for _, m := range miners {
		if !m.OnCooldown {
			// Check if there's room to increase
			roomToIncrease := m.MaxPreset.Watts - m.CurrentPreset.Watts
			if roomToIncrease > 0 {
				available = append(available, m)
			}
		}
	}

	// Sort by efficiency descending (most efficient miners increased first)
	// This maximizes hashrate gained per watt when increasing consumption
	sort.Slice(available, func(i, j int) bool {
		return available[i].Efficiency > available[j].Efficiency
	})

	// Calculate changes to reach target increase
	var changes []*PresetChange
	remainingIncrease := targetIncreaseW

	for _, miner := range available {
		if remainingIncrease <= 0 {
			break
		}

		// Get available presets for this miner's model
		presets, err := s.repo.GetModelPresets(ctx, miner.Model.ID)
		if err != nil {
			continue
		}

		// Find a target preset that increases power
		targetPreset := s.findIncreasePreset(miner, presets, remainingIncrease)
		if targetPreset == nil {
			continue
		}

		delta := targetPreset.Watts - miner.CurrentPreset.Watts
		if delta <= 0 {
			continue
		}

		changes = append(changes, &PresetChange{
			Miner:          miner,
			FromPreset:     miner.CurrentPreset,
			ToPreset:       targetPreset,
			ExpectedDeltaW: -delta, // Negative = increase
		})

		remainingIncrease -= delta
	}

	return changes, nil
}

// findReductionPreset finds the best preset to reduce to.
// It tries to match the needed reduction without over-reducing.
func (s *Strategy) findReductionPreset(miner *MinerWithContext, presets []*ModelPreset, neededReductionW int) *ModelPreset {
	currentWatts := miner.CurrentPreset.Watts
	minWatts := miner.MinPreset.Watts

	// Sort presets by watts descending (highest first)
	sorted := make([]*ModelPreset, len(presets))
	copy(sorted, presets)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Watts > sorted[j].Watts
	})

	// Find the best preset:
	// - Must be lower than current
	// - Must be >= min
	// - Ideally provides close to neededReductionW without exceeding it too much
	var bestPreset *ModelPreset
	var bestDiff int = -1

	for _, preset := range sorted {
		if preset.Watts >= currentWatts {
			continue // Must be lower
		}
		if preset.Watts < minWatts {
			continue // Must be >= min
		}

		reduction := currentWatts - preset.Watts
		diff := reduction - neededReductionW
		if diff < 0 {
			diff = -diff
		}

		// Prefer presets that give us close to what we need
		if bestPreset == nil || diff < bestDiff {
			bestPreset = preset
			bestDiff = diff
		}

		// If we found one that provides at least what we need, use it
		if reduction >= neededReductionW {
			return preset
		}
	}

	return bestPreset
}

// findIncreasePreset finds the best preset to increase to.
func (s *Strategy) findIncreasePreset(miner *MinerWithContext, presets []*ModelPreset, neededIncreaseW int) *ModelPreset {
	currentWatts := miner.CurrentPreset.Watts
	maxWatts := miner.MaxPreset.Watts

	// Sort presets by watts ascending (lowest first)
	sorted := make([]*ModelPreset, len(presets))
	copy(sorted, presets)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Watts < sorted[j].Watts
	})

	// Find the best preset:
	// - Must be higher than current
	// - Must be <= max
	// - Ideally provides close to neededIncreaseW
	var bestPreset *ModelPreset

	for _, preset := range sorted {
		if preset.Watts <= currentWatts {
			continue // Must be higher
		}
		if preset.Watts > maxWatts {
			continue // Must be <= max
		}

		increase := preset.Watts - currentWatts

		// Take the first one that gives us at least what we need
		if increase >= neededIncreaseW {
			return preset
		}

		// Otherwise keep the highest valid one
		bestPreset = preset
	}

	return bestPreset
}

// GetAvailableReductionCapacity returns the total watts that can be reduced.
func (s *Strategy) GetAvailableReductionCapacity(ctx context.Context) (int, error) {
	miners, err := s.repo.GetManageableMiners(ctx)
	if err != nil {
		return 0, err
	}

	var total int
	for _, m := range miners {
		if !m.OnCooldown {
			total += m.HeadroomWatts
		}
	}
	return total, nil
}

// GetAvailableIncreaseCapacity returns the total watts that can be increased.
func (s *Strategy) GetAvailableIncreaseCapacity(ctx context.Context) (int, error) {
	miners, err := s.repo.GetManageableMiners(ctx)
	if err != nil {
		return 0, err
	}

	var total int
	for _, m := range miners {
		if !m.OnCooldown {
			room := m.MaxPreset.Watts - m.CurrentPreset.Watts
			if room > 0 {
				total += room
			}
		}
	}
	return total, nil
}
