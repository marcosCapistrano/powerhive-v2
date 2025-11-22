// harvest-dashboard is a web dashboard for monitoring mining hardware.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/database"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server holds the dashboard server dependencies.
type Server struct {
	repo      database.Repository
	templates *template.Template
}

// Config holds server configuration.
type Config struct {
	Port   string
	DBPath string
}

func main() {
	cfg := loadConfig()

	// Initialize database
	repo, err := database.NewSQLiteRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer repo.Close()

	// Parse templates
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templatesFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	server := &Server{
		repo:      repo,
		templates: tmpl,
	}

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Pages
	mux.HandleFunc("/", server.handleIndex)
	mux.HandleFunc("/miner/", server.handleMinerDetail)

	// API endpoints
	mux.HandleFunc("/api/miners", server.handleAPIMiners)
	mux.HandleFunc("/api/miner/", server.handleAPIMiner)

	log.Printf("Starting dashboard on http://localhost:%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadConfig() *Config {
	cfg := &Config{
		Port:   "8080",
		DBPath: "powerhive.db",
	}

	if p := os.Getenv("DASHBOARD_PORT"); p != "" {
		cfg.Port = p
	}
	if db := os.Getenv("POWERHIVE_DB"); db != "" {
		cfg.DBPath = db
	}

	return cfg
}

// Template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(seconds int) string {
			d := time.Duration(seconds) * time.Second
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
		},
		"formatHashrate": func(hr float64, unit string) string {
			if hr >= 1000000 {
				return fmt.Sprintf("%.2f P%s", hr/1000000, unit)
			}
			if hr >= 1000 {
				return fmt.Sprintf("%.2f T%s", hr/1000, unit)
			}
			return fmt.Sprintf("%.2f %s", hr, unit)
		},
		"statusColor": func(state string) string {
			switch state {
			case "running":
				return "success"
			case "stopped":
				return "warning"
			case "failure":
				return "danger"
			default:
				return "secondary"
			}
		},
		"tempColor": func(temp int) string {
			if temp >= 85 {
				return "danger"
			}
			if temp >= 75 {
				return "warning"
			}
			return "success"
		},
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}
}

// Page handlers

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()

	// Parse filter parameters
	filter := database.MinerFilter{
		MinerType:    r.URL.Query().Get("model"),
		FirmwareType: r.URL.Query().Get("firmware"),
		OnlineStatus: r.URL.Query().Get("status"),
		SortBy:       r.URL.Query().Get("sort"),
		SortOrder:    r.URL.Query().Get("order"),
	}

	// Get filtered miners
	miners, err := s.repo.ListMinersFiltered(ctx, filter)
	if err != nil {
		http.Error(w, "Failed to load miners", http.StatusInternalServerError)
		return
	}

	// Get all miners for stats (unfiltered)
	allMiners, _ := s.repo.ListMiners(ctx)

	// Get distinct miner types for filter dropdown
	minerTypes, _ := s.repo.GetDistinctMinerTypes(ctx)

	// Gather summary data for each miner
	type MinerView struct {
		Miner   *database.Miner
		Status  *database.MinerStatus
		Summary *database.MinerSummary
	}

	var minerViews []MinerView
	var totalHashrate float64
	var totalPower int
	var onlineCount, offlineCount int
	var runningCount, failedCount int

	// Calculate stats from all miners (not filtered)
	for _, m := range allMiners {
		status, _ := s.repo.GetMinerStatus(ctx, m.ID)
		summary, _ := s.repo.GetMinerSummary(ctx, m.ID)

		if m.IsOnline {
			onlineCount++
		} else {
			offlineCount++
		}

		if status != nil && status.State == "running" {
			runningCount++
		}
		if status != nil && status.State == "failure" {
			failedCount++
		}

		if summary != nil {
			totalHashrate += summary.HashrateAvg
			totalPower += summary.PowerConsumption
		}
	}

	// Build view models for filtered miners
	for _, m := range miners {
		mv := MinerView{Miner: m}
		mv.Status, _ = s.repo.GetMinerStatus(ctx, m.ID)
		mv.Summary, _ = s.repo.GetMinerSummary(ctx, m.ID)
		minerViews = append(minerViews, mv)
	}

	data := map[string]interface{}{
		"Title":          "PowerHive Dashboard",
		"Miners":         minerViews,
		"TotalMiners":    len(allMiners),
		"FilteredCount":  len(miners),
		"TotalHashrate":  totalHashrate,
		"TotalPower":     totalPower,
		"OnlineCount":    onlineCount,
		"OfflineCount":   offlineCount,
		"RunningCount":   runningCount,
		"FailedCount":    failedCount,
		"MinerTypes":     minerTypes,
		"Filter":         filter,
	}

	s.render(w, "index.html", data)
}

func (s *Server) handleMinerDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract miner ID from path: /miner/{id}
	idStr := r.URL.Path[len("/miner/"):]
	if idStr == "" {
		http.NotFound(w, r)
		return
	}

	// Try parsing as ID first, then as IP
	var miner *database.Miner
	var err error

	if id, parseErr := strconv.ParseInt(idStr, 10, 64); parseErr == nil {
		miner, err = s.repo.GetMiner(ctx, id)
	} else {
		miner, err = s.repo.GetMinerByIP(ctx, idStr)
	}

	if err != nil || miner == nil {
		http.NotFound(w, r)
		return
	}

	// Get all related data
	details, err := database.GetMinerWithDetails(ctx, s.repo, miner.ID)
	if err != nil {
		http.Error(w, "Failed to load miner details", http.StatusInternalServerError)
		return
	}

	// Get metrics for charts (default 24h)
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "24h"
	}

	var from time.Time
	now := time.Now()
	switch rangeParam {
	case "1h":
		from = now.Add(-1 * time.Hour)
	case "24h":
		from = now.Add(-24 * time.Hour)
	case "7d":
		from = now.Add(-7 * 24 * time.Hour)
	case "30d":
		from = now.Add(-30 * 24 * time.Hour)
	default:
		from = now.Add(-24 * time.Hour)
	}

	metrics, _ := s.repo.GetMinerMetrics(ctx, miner.ID, from, now)

	// Get autotune presets (VNish)
	presets, _ := s.repo.GetAutotunePresets(ctx, miner.ID)

	// Get log sessions
	logSessions, _ := s.repo.GetLogSessions(ctx, miner.ID)
	var currentSession *database.MinerLogSession
	if len(logSessions) > 0 {
		// The current session is the most recent one (first in list, sorted by boot_time desc)
		currentSession = logSessions[0]
	}

	data := map[string]interface{}{
		"Title":          fmt.Sprintf("%s - PowerHive", miner.IPAddress),
		"Miner":          details.Miner,
		"Network":        details.Network,
		"Hardware":       details.Hardware,
		"Status":         details.Status,
		"Summary":        details.Summary,
		"Chains":         details.Chains,
		"Pools":          details.Pools,
		"Fans":           details.Fans,
		"Metrics":        metrics,
		"Presets":        presets,
		"TimeRange":      rangeParam,
		"TimeRanges":     []string{"1h", "24h", "7d", "30d"},
		"LogSessions":    logSessions,
		"CurrentSession": currentSession,
	}

	s.render(w, "miner.html", data)
}

// API handlers

func (s *Server) handleAPIMiners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	miners, err := s.repo.ListMiners(ctx)
	if err != nil {
		s.jsonError(w, "Failed to load miners", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, miners)
}

func (s *Server) handleAPIMiner(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path after /api/miner/
	path := r.URL.Path[len("/api/miner/"):]
	parts := splitPath(path)

	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	// Get miner by ID
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		s.jsonError(w, "Invalid miner ID", http.StatusBadRequest)
		return
	}

	// Check for sub-resource
	if len(parts) > 1 {
		switch parts[1] {
		case "metrics":
			s.handleAPIMinerMetrics(w, r, ctx, id)
			return
		case "status":
			s.handleAPIMinerStatus(w, r, ctx, id)
			return
		case "logs":
			s.handleAPIMinerLogs(w, r, ctx, id)
			return
		}
	}

	// Return full miner details
	details, err := database.GetMinerWithDetails(ctx, s.repo, id)
	if err != nil || details == nil {
		http.NotFound(w, r)
		return
	}

	s.jsonResponse(w, details)
}

func (s *Server) handleAPIMinerMetrics(w http.ResponseWriter, r *http.Request, ctx context.Context, minerID int64) {
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "24h"
	}

	var from time.Time
	now := time.Now()
	switch rangeParam {
	case "1h":
		from = now.Add(-1 * time.Hour)
	case "24h":
		from = now.Add(-24 * time.Hour)
	case "7d":
		from = now.Add(-7 * 24 * time.Hour)
	case "30d":
		from = now.Add(-30 * 24 * time.Hour)
	default:
		from = now.Add(-24 * time.Hour)
	}

	metrics, err := s.repo.GetMinerMetrics(ctx, minerID, from, now)
	if err != nil {
		s.jsonError(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, metrics)
}

func (s *Server) handleAPIMinerStatus(w http.ResponseWriter, r *http.Request, ctx context.Context, minerID int64) {
	status, err := s.repo.GetMinerStatus(ctx, minerID)
	if err != nil {
		s.jsonError(w, "Failed to load status", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, status)
}

func (s *Server) handleAPIMinerLogs(w http.ResponseWriter, r *http.Request, ctx context.Context, minerID int64) {
	sessionIDStr := r.URL.Query().Get("session")
	logType := r.URL.Query().Get("type")

	if sessionIDStr == "" || logType == "" {
		s.jsonError(w, "session and type parameters are required", http.StatusBadRequest)
		return
	}

	sessionID, err := strconv.ParseInt(sessionIDStr, 10, 64)
	if err != nil {
		s.jsonError(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	// Fetch logs (limit to 1000 entries)
	logs, err := s.repo.GetSessionLogs(ctx, sessionID, logType, 1000, 0)
	if err != nil {
		s.jsonError(w, "Failed to load logs", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"logs": logs,
	})
}

// Helper methods

func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range splitBy(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitBy(s string, sep rune) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}
