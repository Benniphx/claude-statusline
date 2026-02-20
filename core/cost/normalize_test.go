package cost

import (
	"testing"

	"github.com/Benniphx/claude-statusline/core/types"
)

func TestCostMultiplier(t *testing.T) {
	cfg := types.DefaultConfig()

	tests := []struct {
		name string
		tier types.CostTier
		want float64
	}{
		{"opus", types.CostTierOpus, 5.0},
		{"sonnet", types.CostTierSonnet, 1.0},
		{"haiku", types.CostTierHaiku, 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CostMultiplier(tt.tier, cfg)
			if got != tt.want {
				t.Errorf("CostMultiplier(%v) = %f, want %f", tt.tier, got, tt.want)
			}
		})
	}
}

func TestCostMultiplierCustomWeights(t *testing.T) {
	cfg := types.DefaultConfig()
	cfg.CostWeightOpus = 10.0
	cfg.CostWeightHaiku = 0.5

	if got := CostMultiplier(types.CostTierOpus, cfg); got != 10.0 {
		t.Errorf("CostMultiplier(opus, custom) = %f, want 10.0", got)
	}
	if got := CostMultiplier(types.CostTierHaiku, cfg); got != 0.5 {
		t.Errorf("CostMultiplier(haiku, custom) = %f, want 0.5", got)
	}
}

func TestNormalizeBurnRate(t *testing.T) {
	cfg := types.DefaultConfig()

	tests := []struct {
		name     string
		tpm      float64
		tier     types.CostTier
		wantTPM  float64
		wantNorm bool
	}{
		{"sonnet_baseline", 1000, types.CostTierSonnet, 1000, false},
		{"opus_5x", 1000, types.CostTierOpus, 5000, true},
		{"haiku_0.25x", 1000, types.CostTierHaiku, 250, true},
		{"zero_tpm", 0, types.CostTierOpus, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTPM, gotNorm := NormalizeBurnRate(tt.tpm, tt.tier, cfg)
			if gotTPM != tt.wantTPM {
				t.Errorf("NormalizeBurnRate tpm = %f, want %f", gotTPM, tt.wantTPM)
			}
			if gotNorm != tt.wantNorm {
				t.Errorf("NormalizeBurnRate normalized = %v, want %v", gotNorm, tt.wantNorm)
			}
		})
	}
}

func TestNormalizeBurnRateDisabled(t *testing.T) {
	cfg := types.DefaultConfig()
	cfg.CostNormalize = false

	// With normalization disabled, Opus should return raw value
	gotTPM, gotNorm := NormalizeBurnRate(1000, types.CostTierOpus, cfg)
	if gotTPM != 1000 {
		t.Errorf("Disabled: tpm = %f, want 1000", gotTPM)
	}
	if gotNorm {
		t.Error("Disabled: normalized should be false")
	}
}

func TestNormalizePace(t *testing.T) {
	cfg := types.DefaultConfig()

	tests := []struct {
		name     string
		pace     float64
		tier     types.CostTier
		wantPace float64
		wantNorm bool
	}{
		{"sonnet_baseline", 1.0, types.CostTierSonnet, 1.0, false},
		{"opus_5x", 1.0, types.CostTierOpus, 5.0, true},
		{"haiku_0.25x", 1.0, types.CostTierHaiku, 0.25, true},
		{"opus_partial", 0.5, types.CostTierOpus, 2.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPace, gotNorm := NormalizePace(tt.pace, tt.tier, cfg)
			if gotPace != tt.wantPace {
				t.Errorf("NormalizePace = %f, want %f", gotPace, tt.wantPace)
			}
			if gotNorm != tt.wantNorm {
				t.Errorf("NormalizePace normalized = %v, want %v", gotNorm, tt.wantNorm)
			}
		})
	}
}

func TestNormalizeCostPerHour(t *testing.T) {
	cfg := types.DefaultConfig()

	// Opus $2/h → normalized $10/h
	got, norm := NormalizeCostPerHour(2.0, types.CostTierOpus, cfg)
	if got != 10.0 {
		t.Errorf("NormalizeCostPerHour(opus) = %f, want 10.0", got)
	}
	if !norm {
		t.Error("NormalizeCostPerHour(opus) should be normalized")
	}

	// Sonnet stays the same
	got2, norm2 := NormalizeCostPerHour(2.0, types.CostTierSonnet, cfg)
	if got2 != 2.0 {
		t.Errorf("NormalizeCostPerHour(sonnet) = %f, want 2.0", got2)
	}
	if norm2 {
		t.Error("NormalizeCostPerHour(sonnet) should not be normalized")
	}
}

func TestIsNonBaseline(t *testing.T) {
	if IsNonBaseline(types.CostTierSonnet) {
		t.Error("Sonnet should be baseline")
	}
	if !IsNonBaseline(types.CostTierOpus) {
		t.Error("Opus should be non-baseline")
	}
	if !IsNonBaseline(types.CostTierHaiku) {
		t.Error("Haiku should be non-baseline")
	}
}
