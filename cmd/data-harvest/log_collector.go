package main

import (
	"context"
	"log"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/database"
	"github.com/powerhive/powerhive-v2/pkg/stock"
	"github.com/powerhive/powerhive-v2/pkg/vnish"
)

// LogCollector handles log collection from miners.
type LogCollector struct {
	repo   database.Repository
	parser *database.LogParser
}

// NewLogCollector creates a new log collector.
func NewLogCollector(repo database.Repository) *LogCollector {
	return &LogCollector{
		repo:   repo,
		parser: database.NewLogParser(),
	}
}

// CollectVNishLogs collects logs from a VNish miner.
func (lc *LogCollector) CollectVNishLogs(ctx context.Context, client *vnish.HTTPClient, minerID int64, uptimeSeconds int) error {
	ip := client.Host()

	// Calculate boot time from uptime
	bootTime := database.CalculateBootTime(uptimeSeconds)

	// Get or create log session
	session, isNew, err := lc.ensureLogSession(ctx, minerID, bootTime)
	if err != nil {
		return err
	}

	if isNew {
		log.Printf("[%s] New log session started (boot time: %s)", ip, bootTime.Format(time.RFC3339))
	}

	// Collect each log type
	logTypes := []struct {
		name    string
		fetcher func(ctx context.Context) (string, error)
	}{
		{database.LogTypeStatus, client.GetStatusLogs},
		{database.LogTypeMiner, client.GetMinerLogs},
		{database.LogTypeSystem, client.GetSystemLogs},
		{database.LogTypeAutotune, client.GetAutotuneLogs},
		{database.LogTypeMessages, client.GetMessagesLogs},
		{database.LogTypeAPI, client.GetAPILogs},
	}

	totalNewLogs := 0
	for _, lt := range logTypes {
		rawLogs, err := lt.fetcher(ctx)
		if err != nil {
			log.Printf("[%s] Warning: failed to fetch %s logs: %v", ip, lt.name, err)
			continue
		}

		// Parse logs
		logs := lc.parser.ParseLogLines(rawLogs, minerID, session.ID, lt.name, bootTime)

		// Filter out logs we've already stored (unless new session)
		if !isNew {
			lastTime, _ := lc.repo.GetLastLogTime(ctx, session.ID, lt.name)
			logs = lc.parser.FilterNewLogs(logs, lastTime)
		}

		// Store new logs
		if len(logs) > 0 {
			if err := lc.repo.InsertLogs(ctx, logs); err != nil {
				log.Printf("[%s] Warning: failed to store %s logs: %v", ip, lt.name, err)
			} else {
				totalNewLogs += len(logs)
			}
		}
	}

	if totalNewLogs > 0 {
		log.Printf("[%s] Stored %d new log entries", ip, totalNewLogs)
	}

	return nil
}

// CollectStockLogs collects logs from a Stock firmware miner.
func (lc *LogCollector) CollectStockLogs(ctx context.Context, client *stock.HTTPClient, minerID int64, uptimeSeconds int) error {
	ip := client.Host()

	// Calculate boot time from uptime
	bootTime := database.CalculateBootTime(uptimeSeconds)

	// Get or create log session
	session, isNew, err := lc.ensureLogSession(ctx, minerID, bootTime)
	if err != nil {
		return err
	}

	if isNew {
		log.Printf("[%s] New log session started (boot time: %s)", ip, bootTime.Format(time.RFC3339))
	}

	// Fetch kernel logs
	rawLogs, err := client.GetLogs(ctx)
	if err != nil {
		log.Printf("[%s] Warning: failed to fetch logs: %v", ip, err)
		return nil // Don't fail the whole harvest for logs
	}

	// Parse logs
	logs := lc.parser.ParseLogLines(rawLogs, minerID, session.ID, database.LogTypeKernel, bootTime)

	// For Stock, since logs don't have reliable timestamps, we use message deduplication
	// We'll store all logs for new sessions, or check message hash for existing sessions
	if !isNew {
		// Get existing log count to determine if we need to skip duplicates
		existingCount, _ := lc.repo.GetLogCount(ctx, session.ID, database.LogTypeKernel)
		if existingCount > 0 && len(logs) <= existingCount {
			// Logs haven't grown, skip
			return nil
		}
		// Only store logs beyond what we already have
		if existingCount > 0 && existingCount < len(logs) {
			logs = logs[existingCount:]
		}
	}

	// Store logs
	if len(logs) > 0 {
		if err := lc.repo.InsertLogs(ctx, logs); err != nil {
			log.Printf("[%s] Warning: failed to store logs: %v", ip, err)
		} else {
			log.Printf("[%s] Stored %d new log entries", ip, len(logs))
		}
	}

	return nil
}

// ensureLogSession ensures a log session exists for the given boot time.
// Returns the session, whether it was newly created, and any error.
func (lc *LogCollector) ensureLogSession(ctx context.Context, minerID int64, bootTime time.Time) (*database.MinerLogSession, bool, error) {
	// Check if there's an existing session with matching boot time
	existingSession, err := lc.repo.GetLogSessionByBootTime(ctx, minerID, bootTime)
	if err != nil {
		return nil, false, err
	}

	if existingSession != nil {
		// Session exists and boot time matches
		return existingSession, false, nil
	}

	// Check if there's a current session that needs to be ended
	currentSession, err := lc.repo.GetCurrentLogSession(ctx, minerID)
	if err != nil {
		return nil, false, err
	}

	if currentSession != nil {
		// End the current session - the miner rebooted
		if err := lc.repo.EndLogSession(ctx, currentSession.ID, time.Now(), "reboot"); err != nil {
			log.Printf("Warning: failed to end previous log session: %v", err)
		}
	}

	// Create new session
	newSession := &database.MinerLogSession{
		MinerID:  minerID,
		BootTime: bootTime,
	}

	if err := lc.repo.CreateLogSession(ctx, newSession); err != nil {
		return nil, false, err
	}

	return newSession, true, nil
}
