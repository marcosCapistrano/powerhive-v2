package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Aggregator fetches energy generation/consumption data from the energy aggregator API.
type Aggregator struct {
	url    string
	apiKey string
	client *http.Client
}

// AggregatorResponse represents the JSON response from the energy aggregator.
type AggregatorResponse struct {
	Reading struct {
		CollectionTimestamp string `json:"collection_timestamp"`
		Consumption         struct {
			ContainerEles struct {
				SourceTimestamp string  `json:"source_timestamp"`
				Status          string  `json:"status"`
				ValueMW         float64 `json:"value_mw"`
			} `json:"container_eles"`
			ContainerMazp struct {
				SourceTimestamp string  `json:"source_timestamp"`
				Status          string  `json:"status"`
				ValueMW         float64 `json:"value_mw"`
			} `json:"container_mazp"`
		} `json:"consumption"`
		Generation struct {
			Generoso struct {
				SourceTimestamp string  `json:"source_timestamp"`
				Status          string  `json:"status"`
				ValueMW         float64 `json:"value_mw"`
			} `json:"generoso"`
			Nogueira struct {
				SourceTimestamp string  `json:"source_timestamp"`
				Status          string  `json:"status"`
				ValueMW         float64 `json:"value_mw"`
			} `json:"nogueira"`
		} `json:"generation"`
		ID      int    `json:"id"`
		PlantID string `json:"plant_id"`
		Totals  struct {
			ConsumptionMW float64 `json:"consumption_mw"`
			ExportedMW    float64 `json:"exported_mw"`
			GenerationMW  float64 `json:"generation_mw"`
		} `json:"totals"`
		Trust struct {
			ConfidenceScore float64 `json:"confidence_score"`
			Status          string  `json:"status"`
			Summary         string  `json:"summary"`
		} `json:"trust"`
	} `json:"reading"`
}

// NewAggregator creates a new aggregator client.
func NewAggregator(url, apiKey string) *Aggregator {
	return &Aggregator{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchLatest fetches the latest energy data from the aggregator.
func (a *Aggregator) FetchLatest(ctx context.Context) (*EnergyReading, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var data AggregatorResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	reading := a.toEnergyReading(&data)
	return reading, nil
}

// toEnergyReading converts the API response to our internal model.
func (a *Aggregator) toEnergyReading(data *AggregatorResponse) *EnergyReading {
	r := data.Reading

	generationMW := r.Totals.GenerationMW
	consumptionMW := r.Totals.ConsumptionMW
	marginMW := generationMW - consumptionMW

	var marginPercent float64
	if generationMW > 0 {
		marginPercent = (marginMW / generationMW) * 100
	}

	return &EnergyReading{
		Timestamp:      time.Now(),
		GenerationMW:   generationMW,
		ConsumptionMW:  consumptionMW,
		MarginMW:       marginMW,
		MarginPercent:  marginPercent,
		GenerosoMW:     r.Generation.Generoso.ValueMW,
		GenerosoStatus: r.Generation.Generoso.Status,
		NogueiraMW:     r.Generation.Nogueira.ValueMW,
		NogueiraStatus: r.Generation.Nogueira.Status,
	}
}
