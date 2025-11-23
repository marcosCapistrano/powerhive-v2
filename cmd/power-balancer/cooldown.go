package main

import (
	"context"
	"time"
)

// CooldownManager handles cooldown tracking for miners.
type CooldownManager struct {
	repo     *Repository
	duration time.Duration
}

// NewCooldownManager creates a new cooldown manager.
func NewCooldownManager(repo *Repository, duration time.Duration) *CooldownManager {
	return &CooldownManager{
		repo:     repo,
		duration: duration,
	}
}

// SetCooldown sets a cooldown for a miner starting now.
func (cm *CooldownManager) SetCooldown(ctx context.Context, minerID int64) error {
	until := time.Now().Add(cm.duration)
	return cm.repo.SetCooldown(ctx, minerID, until)
}

// IsOnCooldown checks if a miner is currently on cooldown.
func (cm *CooldownManager) IsOnCooldown(ctx context.Context, minerID int64) (bool, *time.Time, error) {
	cooldown, err := cm.repo.GetCooldown(ctx, minerID)
	if err != nil {
		return false, nil, err
	}
	if cooldown == nil {
		return false, nil, nil
	}

	now := time.Now()
	if cooldown.Until.After(now) {
		return true, &cooldown.Until, nil
	}
	return false, nil, nil
}

// GetRemainingCooldown returns the remaining cooldown time for a miner.
func (cm *CooldownManager) GetRemainingCooldown(ctx context.Context, minerID int64) (time.Duration, error) {
	cooldown, err := cm.repo.GetCooldown(ctx, minerID)
	if err != nil {
		return 0, err
	}
	if cooldown == nil {
		return 0, nil
	}

	remaining := time.Until(cooldown.Until)
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// CleanupExpired removes expired cooldowns from the database.
func (cm *CooldownManager) CleanupExpired(ctx context.Context) error {
	return cm.repo.ClearExpiredCooldowns(ctx)
}

// CountActive returns the number of miners currently on cooldown.
func (cm *CooldownManager) CountActive(ctx context.Context) (int, error) {
	return cm.repo.CountMinersOnCooldown(ctx)
}
