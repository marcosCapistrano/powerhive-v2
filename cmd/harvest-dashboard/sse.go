package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/powerhive/powerhive-v2/pkg/database"
)

// SSEHub manages Server-Sent Events connections.
type SSEHub struct {
	repo    database.Repository
	clients sync.Map // map[*sseClient]bool
}

type sseClient struct {
	id       string
	response http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
}

// DashboardUpdate contains live dashboard data.
type DashboardUpdate struct {
	Timestamp     string        `json:"timestamp"`
	OnlineCount   int           `json:"online_count"`
	OfflineCount  int           `json:"offline_count"`
	TotalHashrate float64       `json:"total_hashrate"`
	TotalPower    int           `json:"total_power"`
	RunningCount  int           `json:"running_count"`
	FailedCount   int           `json:"failed_count"`
	Miners        []MinerUpdate `json:"miners"`
}

// MinerUpdate contains live miner data.
type MinerUpdate struct {
	ID              int64    `json:"id"`
	IPAddress       string   `json:"ip_address"`
	IsOnline        bool     `json:"is_online"`
	State           string   `json:"state"`
	Model           string   `json:"model"`
	FirmwareType    string   `json:"firmware_type"`
	FirmwareVersion string   `json:"firmware_version"`
	Hashrate        float64  `json:"hashrate"`
	HRUnit          string   `json:"hr_unit"`
	ChipTempMax     int      `json:"chip_temp_max"`
	PCBTempMax      int      `json:"pcb_temp_max"`
	ChainStates     []string `json:"chain_states"`
	FanStates       []string `json:"fan_states"`
	Preset          string   `json:"preset"`
	Uptime          int      `json:"uptime"`
}

// MinerDetailUpdate contains live miner detail data.
type MinerDetailUpdate struct {
	Timestamp  string                  `json:"timestamp"`
	Status     *database.MinerStatus   `json:"status"`
	Summary    *database.MinerSummary  `json:"summary"`
	Chains     []*database.MinerChain  `json:"chains"`
	LastMetric *database.MinerMetric   `json:"last_metric"`
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub(repo database.Repository) *SSEHub {
	return &SSEHub{repo: repo}
}

// handleDashboardSSE handles SSE connections for the dashboard.
func (h *SSEHub) handleDashboardSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	client := &sseClient{
		id:       fmt.Sprintf("dashboard-%d", time.Now().UnixNano()),
		response: w,
		flusher:  flusher,
		done:     make(chan struct{}),
	}

	h.clients.Store(client, true)
	defer h.clients.Delete(client)

	log.Printf("[SSE] Dashboard client connected: %s", client.id)

	// Send initial data
	h.sendDashboardUpdate(client)

	// Create ticker for updates
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Printf("[SSE] Dashboard client disconnected: %s", client.id)
			return
		case <-client.done:
			return
		case <-ticker.C:
			h.sendDashboardUpdate(client)
		}
	}
}

// handleMinerSSE handles SSE connections for a specific miner.
func (h *SSEHub) handleMinerSSE(w http.ResponseWriter, r *http.Request) {
	// Extract miner ID from path: /api/sse/miner/{id}
	path := r.URL.Path[len("/api/sse/miner/"):]
	if path == "" {
		http.Error(w, "Miner ID required", http.StatusBadRequest)
		return
	}

	minerID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid miner ID", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	client := &sseClient{
		id:       fmt.Sprintf("miner-%d-%d", minerID, time.Now().UnixNano()),
		response: w,
		flusher:  flusher,
		done:     make(chan struct{}),
	}

	h.clients.Store(client, true)
	defer h.clients.Delete(client)

	log.Printf("[SSE] Miner client connected: %s", client.id)

	// Send initial data
	h.sendMinerUpdate(client, minerID)

	// Create ticker for updates (faster for detail page)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Printf("[SSE] Miner client disconnected: %s", client.id)
			return
		case <-client.done:
			return
		case <-ticker.C:
			h.sendMinerUpdate(client, minerID)
		}
	}
}

// sendDashboardUpdate sends dashboard data to a client.
func (h *SSEHub) sendDashboardUpdate(client *sseClient) {
	ctx := context.Background()

	miners, err := h.repo.ListMiners(ctx)
	if err != nil {
		log.Printf("[SSE] Error fetching miners: %v", err)
		return
	}

	update := DashboardUpdate{
		Timestamp: time.Now().Format(time.RFC3339),
		Miners:    make([]MinerUpdate, 0, len(miners)),
	}

	for _, m := range miners {
		if m.IsOnline {
			update.OnlineCount++
		} else {
			update.OfflineCount++
		}

		status, _ := h.repo.GetMinerStatus(ctx, m.ID)
		summary, _ := h.repo.GetMinerSummary(ctx, m.ID)
		chains, _ := h.repo.GetMinerChains(ctx, m.ID)
		fans, _ := h.repo.GetMinerFans(ctx, m.ID)
		presets, _ := h.repo.GetAutotunePresets(ctx, m.ID)

		mu := MinerUpdate{
			ID:              m.ID,
			IPAddress:       m.IPAddress,
			IsOnline:        m.IsOnline,
			Model:           m.MinerType,
			FirmwareType:    string(m.FirmwareType),
			FirmwareVersion: m.FirmwareVersion,
			HRUnit:          m.HRMeasure,
		}

		if status != nil {
			mu.State = status.State
			mu.Uptime = status.UptimeSeconds
			if status.State == "running" {
				update.RunningCount++
			} else if status.State == "failure" {
				update.FailedCount++
			}
		}

		if summary != nil {
			mu.Hashrate = summary.HashrateAvg
			mu.ChipTempMax = summary.ChipTempMax
			mu.PCBTempMax = summary.PCBTempMax
			update.TotalHashrate += summary.HashrateAvg
			update.TotalPower += summary.PowerConsumption
		}

		// Compute chain health states
		mu.ChainStates = computeChainStates(chains, m.IsOnline)

		// Compute fan health states
		mu.FanStates = computeFanStates(fans, m.IsOnline)

		// Get current preset
		mu.Preset = getCurrentPreset(presets)

		update.Miners = append(update.Miners, mu)
	}

	h.sendEvent(client, "dashboard", update)
}

// sendMinerUpdate sends miner detail data to a client.
func (h *SSEHub) sendMinerUpdate(client *sseClient, minerID int64) {
	ctx := context.Background()

	update := MinerDetailUpdate{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	update.Status, _ = h.repo.GetMinerStatus(ctx, minerID)
	update.Summary, _ = h.repo.GetMinerSummary(ctx, minerID)
	update.Chains, _ = h.repo.GetMinerChains(ctx, minerID)

	// Get latest metric
	now := time.Now()
	metrics, _ := h.repo.GetMinerMetrics(ctx, minerID, now.Add(-5*time.Minute), now)
	if len(metrics) > 0 {
		update.LastMetric = metrics[len(metrics)-1]
	}

	h.sendEvent(client, "miner", update)
}

// sendEvent sends an SSE event to a client.
func (h *SSEHub) sendEvent(client *sseClient, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[SSE] Error marshaling data: %v", err)
		return
	}

	fmt.Fprintf(client.response, "event: %s\n", eventType)
	fmt.Fprintf(client.response, "data: %s\n\n", jsonData)
	client.flusher.Flush()
}
