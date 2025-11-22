package database

import (
	"regexp"
	"strings"
	"time"
)

// LogParser handles parsing of log lines from different firmware types.
type LogParser struct {
	// VNish timestamp format: [YYYY/MM/DD HH:MM:SS]
	vnishTimestampRE *regexp.Regexp
	// VNish API log format: [DD Mon HH:MM:SS]
	vnishAPITimestampRE *regexp.Regexp
}

// NewLogParser creates a new log parser.
func NewLogParser() *LogParser {
	return &LogParser{
		vnishTimestampRE:    regexp.MustCompile(`^\[(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\]`),
		vnishAPITimestampRE: regexp.MustCompile(`^\[(\d{2} \w{3} \d{2}:\d{2}:\d{2})\]`),
	}
}

// ParsedLog represents a parsed log line.
type ParsedLog struct {
	Timestamp *time.Time
	Message   string
}

// ParseVNishLogLine parses a VNish log line and extracts timestamp.
// Format: [YYYY/MM/DD HH:MM:SS] LEVEL: Message
// Or for epoch-based: [1970/01/01 HH:MM:SS] (converted using bootTime)
func (p *LogParser) ParseVNishLogLine(line string, bootTime time.Time) *ParsedLog {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	result := &ParsedLog{Message: line}

	// Try to extract timestamp
	matches := p.vnishTimestampRE.FindStringSubmatch(line)
	if len(matches) >= 2 {
		tsStr := matches[1]

		// Parse the timestamp
		t, err := time.ParseInLocation("2006/01/02 15:04:05", tsStr, time.Local)
		if err == nil {
			// Check if this is an epoch-based timestamp (1970/01/01)
			if t.Year() == 1970 {
				// Convert to actual time using boot time
				// The timestamp represents seconds since boot
				secsSinceBoot := t.Hour()*3600 + t.Minute()*60 + t.Second()
				actualTime := bootTime.Add(time.Duration(secsSinceBoot) * time.Second)
				result.Timestamp = &actualTime
			} else {
				result.Timestamp = &t
			}
		}
	}

	return result
}

// ParseVNishAPILogLine parses a VNish API log line.
// Format: [DD Mon HH:MM:SS] IP:Port "METHOD /endpoint" StatusCode "-" ResponseTimeMS
func (p *LogParser) ParseVNishAPILogLine(line string, referenceDate time.Time) *ParsedLog {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	result := &ParsedLog{Message: line}

	matches := p.vnishAPITimestampRE.FindStringSubmatch(line)
	if len(matches) >= 2 {
		tsStr := matches[1]
		// Format: "21 Nov 15:04:05"
		// We need to add the year from reference date
		fullTs := tsStr + " " + referenceDate.Format("2006")
		t, err := time.ParseInLocation("02 Jan 15:04:05 2006", fullTs, time.Local)
		if err == nil {
			result.Timestamp = &t
		}
	}

	return result
}

// ParseStockLogLine parses a Stock firmware log line.
// Stock logs typically don't have reliable timestamps, so we use current time.
func (p *LogParser) ParseStockLogLine(line string) *ParsedLog {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	return &ParsedLog{
		Message:   line,
		Timestamp: nil, // Stock logs don't have reliable timestamps
	}
}

// ParseLogLines parses multiple log lines and creates MinerLog entries.
func (p *LogParser) ParseLogLines(rawLogs string, minerID, sessionID int64, logType string, bootTime time.Time) []*MinerLog {
	lines := strings.Split(rawLogs, "\n")
	var logs []*MinerLog

	referenceDate := time.Now()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var parsed *ParsedLog

		switch logType {
		case LogTypeAPI:
			parsed = p.ParseVNishAPILogLine(line, referenceDate)
		case LogTypeKernel:
			parsed = p.ParseStockLogLine(line)
		default:
			// VNish standard log format
			parsed = p.ParseVNishLogLine(line, bootTime)
		}

		if parsed != nil {
			logs = append(logs, &MinerLog{
				MinerID:   minerID,
				SessionID: sessionID,
				LogType:   logType,
				LogTime:   parsed.Timestamp,
				Message:   parsed.Message,
			})
		}
	}

	return logs
}

// FilterNewLogs filters out logs that are older than lastLogTime.
// This is used to avoid storing duplicate logs.
func (p *LogParser) FilterNewLogs(logs []*MinerLog, lastLogTime *time.Time) []*MinerLog {
	if lastLogTime == nil {
		return logs // No previous logs, return all
	}

	var newLogs []*MinerLog
	for _, log := range logs {
		// If log has no timestamp, include it (we can't determine if it's new)
		// If log has timestamp and it's after lastLogTime, include it
		if log.LogTime == nil || log.LogTime.After(*lastLogTime) {
			newLogs = append(newLogs, log)
		}
	}

	return newLogs
}

// CalculateBootTime calculates the boot time from current time and uptime.
func CalculateBootTime(uptimeSeconds int) time.Time {
	return time.Now().Add(-time.Duration(uptimeSeconds) * time.Second)
}

// IsSameBootSession checks if two boot times are within tolerance (2 minutes).
func IsSameBootSession(bootTime1, bootTime2 time.Time) bool {
	diff := bootTime1.Sub(bootTime2)
	if diff < 0 {
		diff = -diff
	}
	return diff < 2*time.Minute
}
