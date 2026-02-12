package ollama

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const (
	// StatsPath is the default location for Ollama agent stats.
	StatsPath = "/tmp/claude_ollama_stats.json"

	// FreshnessLimit is how old stats can be before they're considered stale.
	FreshnessLimit = 5 * time.Minute

	// Haiku pricing per million tokens (used for savings calculation).
	haikuInputPricePerM  = 0.25
	haikuOutputPricePerM = 1.25
)

// Stats holds aggregated Ollama agent usage data.
type Stats struct {
	Requests              int   `json:"requests"`
	TotalPromptTokens     int   `json:"total_prompt_tokens"`
	TotalCompletionTokens int   `json:"total_completion_tokens"`
	LastUpdated           int64 `json:"last_updated"`
}

// ReadStats reads and parses the Ollama stats file.
// Returns nil if the file doesn't exist or is stale.
func ReadStats(path string) (*Stats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var stats Stats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("invalid stats JSON: %w", err)
	}

	// Check freshness
	updated := time.Unix(stats.LastUpdated, 0)
	if time.Since(updated) > FreshnessLimit {
		return nil, fmt.Errorf("stats stale (last updated %s ago)", time.Since(updated).Truncate(time.Second))
	}

	return &stats, nil
}

// Savings calculates the estimated cost savings from using Ollama instead of Haiku.
func Savings(stats *Stats) float64 {
	if stats == nil {
		return 0
	}
	inputCost := float64(stats.TotalPromptTokens) / 1_000_000 * haikuInputPricePerM
	outputCost := float64(stats.TotalCompletionTokens) / 1_000_000 * haikuOutputPricePerM
	return inputCost + outputCost
}

// Render formats Ollama stats for display in the statusline.
// Returns empty string if stats are nil or empty.
func Render(stats *Stats) string {
	if stats == nil || stats.Requests == 0 {
		return ""
	}

	saved := Savings(stats)
	if saved < 0.01 {
		return fmt.Sprintf("%d req", stats.Requests)
	}
	return fmt.Sprintf("%d req | saved ~$%.2f", stats.Requests, saved)
}
