package types

import (
	"encoding/json"
	"testing"
)

func TestInputUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		check   func(Input) bool
	}{
		{
			name: "full input",
			json: `{
				"context_window": {
					"context_window_size": 200000,
					"used_percentage": 45.5,
					"current_usage": {
						"input_tokens": 5000,
						"cache_creation_input_tokens": 1000,
						"cache_read_input_tokens": 2000
					},
					"total_input_tokens": 10000,
					"total_output_tokens": 3000
				},
				"model": {
					"display_name": "Claude Sonnet 4",
					"model_id": "claude-sonnet-4-20250514"
				},
				"cost": {
					"total_cost_usd": 0.50,
					"total_duration_ms": 120000,
					"total_lines_added": 50,
					"total_lines_removed": 10
				}
			}`,
			check: func(i Input) bool {
				return i.ContextWindow.ContextWindowSize == 200000 &&
					i.ContextWindow.UsedPercentage != nil &&
					*i.ContextWindow.UsedPercentage == 45.5 &&
					i.ContextWindow.CurrentUsage.InputTokens == 5000 &&
					i.Model.ModelID == "claude-sonnet-4-20250514" &&
					i.Cost.TotalCostUSD == 0.50
			},
		},
		{
			name: "minimal input",
			json: `{"model":{"model_id":"claude-sonnet-4-20250514"}}`,
			check: func(i Input) bool {
				return i.Model.ModelID == "claude-sonnet-4-20250514" &&
					i.ContextWindow.UsedPercentage == nil &&
					i.Cost.TotalCostUSD == 0
			},
		},
		{
			name: "no used_percentage",
			json: `{"context_window":{"context_window_size":200000,"total_input_tokens":50000}}`,
			check: func(i Input) bool {
				return i.ContextWindow.UsedPercentage == nil &&
					i.ContextWindow.TotalInputTokens == 50000
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input Input
			if err := json.Unmarshal([]byte(tt.json), &input); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if !tt.check(input) {
				t.Errorf("check failed for %s", tt.name)
			}
		})
	}
}

func TestCredentialsHasOAuth(t *testing.T) {
	tests := []struct {
		creds    Credentials
		expected bool
	}{
		{Credentials{OAuthToken: "token"}, true},
		{Credentials{APIKey: "key"}, false},
		{Credentials{}, false},
	}

	for _, tt := range tests {
		if got := tt.creds.HasOAuth(); got != tt.expected {
			t.Errorf("HasOAuth() = %v, want %v", got, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.WorkDaysPerWeek != 5 {
		t.Errorf("WorkDaysPerWeek = %d, want 5", cfg.WorkDaysPerWeek)
	}
	if cfg.CacheDir != "/tmp" {
		t.Errorf("CacheDir = %s, want /tmp", cfg.CacheDir)
	}
}
