package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server is the HTTP server for the dashboard.
type Server struct {
	repo     *Repository
	balancer *Balancer
	cfg      *Config
	tmpl     *template.Template
	srv      *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(repo *Repository, balancer *Balancer, cfg *Config) *Server {
	s := &Server{
		repo:     repo,
		balancer: balancer,
		cfg:      cfg,
	}

	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"deref": func(p *int64) int64 {
			if p == nil {
				return 0
			}
			return *p
		},
	}
	var err error
	s.tmpl, err = template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Printf("Warning: Failed to parse templates: %v", err)
	}

	return s
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Pages
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/models", s.handleModels)
	mux.HandleFunc("/logs", s.handleLogs)

	// API endpoints
	mux.HandleFunc("/api/status", s.handleAPIStatus)
	mux.HandleFunc("/api/miners", s.handleAPIMiners)
	mux.HandleFunc("/api/models", s.handleAPIModels)
	mux.HandleFunc("/api/models/", s.handleAPIModelUpdate)
	mux.HandleFunc("/api/miners/", s.handleAPIMinerConfig)
	mux.HandleFunc("/api/logs", s.handleAPILogs)
	mux.HandleFunc("/api/readings", s.handleAPIReadings)

	// SSE endpoint for live updates
	mux.HandleFunc("/api/sse", s.handleSSE)

	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.DashboardPort),
		Handler: mux,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// handleDashboard renders the main dashboard page.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Status": s.balancer.GetStatus(),
	}

	if s.tmpl == nil {
		// Fallback JSON response if templates aren't loaded
		json.NewEncoder(w).Encode(data)
		return
	}

	if err := s.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleModels renders the models configuration page.
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	models, err := s.repo.ListModels(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load presets for each model
	for _, m := range models {
		presets, err := s.repo.GetModelPresets(ctx, m.ID)
		if err == nil {
			m.Presets = presets
		}
		if m.MinPresetID != nil {
			m.MinPreset, _ = s.repo.GetPresetByID(ctx, *m.MinPresetID)
		}
		if m.MaxPresetID != nil {
			m.MaxPreset, _ = s.repo.GetPresetByID(ctx, *m.MaxPresetID)
		}
	}

	data := map[string]interface{}{
		"Models": models,
		"Status": s.balancer.GetStatus(),
	}

	if s.tmpl == nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if err := s.tmpl.ExecuteTemplate(w, "models.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLogs renders the change log page.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logs, err := s.repo.GetRecentChangeLogs(ctx, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Logs":   logs,
		"Status": s.balancer.GetStatus(),
	}

	if s.tmpl == nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if err := s.tmpl.ExecuteTemplate(w, "logs.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIStatus returns the current system status as JSON.
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.balancer.GetStatus())
}

// handleAPIMiners returns all miners as JSON.
func (s *Server) handleAPIMiners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	miners, err := s.repo.ListMiners(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich with related data
	for _, m := range miners {
		if m.ModelID != nil {
			m.Model, _ = s.repo.GetModelByID(ctx, *m.ModelID)
		}
		if m.CurrentPresetID != nil {
			m.CurrentPreset, _ = s.repo.GetPresetByID(ctx, *m.CurrentPresetID)
		}
		m.Config, _ = s.repo.GetOrCreateBalanceConfig(ctx, m.ID)
		m.Cooldown, _ = s.repo.GetCooldown(ctx, m.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(miners)
}

// handleAPIModels returns all models as JSON.
func (s *Server) handleAPIModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	models, err := s.repo.ListModels(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, m := range models {
		m.Presets, _ = s.repo.GetModelPresets(ctx, m.ID)
		if m.MinPresetID != nil {
			m.MinPreset, _ = s.repo.GetPresetByID(ctx, *m.MinPresetID)
		}
		if m.MaxPresetID != nil {
			m.MaxPreset, _ = s.repo.GetPresetByID(ctx, *m.MaxPresetID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// handleAPIModelUpdate handles model configuration updates.
func (s *Server) handleAPIModelUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ModelID     int64  `json:"model_id"`
		MinPresetID *int64 `json:"min_preset_id"`
		MaxPresetID *int64 `json:"max_preset_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.repo.UpdateModelLimits(r.Context(), req.ModelID, req.MinPresetID, req.MaxPresetID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAPIMinerConfig handles miner balance config updates.
func (s *Server) handleAPIMinerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config BalanceConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.repo.UpdateBalanceConfig(r.Context(), &config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAPILogs returns recent change logs as JSON.
func (s *Server) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.repo.GetRecentChangeLogs(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleAPIReadings returns recent energy readings as JSON.
func (s *Server) handleAPIReadings(w http.ResponseWriter, r *http.Request) {
	readings, err := s.repo.GetRecentEnergyReadings(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}

// handleSSE handles Server-Sent Events for live updates.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial status
	s.sendSSEStatus(w, flusher)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			s.sendSSEStatus(w, flusher)
		}
	}
}

func (s *Server) sendSSEStatus(w http.ResponseWriter, flusher http.Flusher) {
	status := s.balancer.GetStatus()
	data, err := json.Marshal(status)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)
	flusher.Flush()
}
