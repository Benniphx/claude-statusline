package ratelimit

import (
	"testing"

	"github.com/Benniphx/claude-statusline/core/types"
)

func TestCalculateBurnRate(t *testing.T) {
	cfg := types.DefaultConfig()

	// Insufficient duration
	input := types.Input{
		ContextWindow: types.ContextWindow{
			CurrentUsage: types.CurrentUsage{InputTokens: 10000},
		},
		Cost: types.Cost{TotalDurationMS: 30000}, // 30s
	}
	info := CalculateBurnRate(input, cfg)
	if info.LocalTPM != 0 {
		t.Errorf("LocalTPM = %f, want 0 (insufficient duration)", info.LocalTPM)
	}

	// With sufficient duration
	input2 := types.Input{
		ContextWindow: types.ContextWindow{
			CurrentUsage: types.CurrentUsage{InputTokens: 60000},
		},
		Cost: types.Cost{TotalDurationMS: 300000}, // 5 min
	}
	info2 := CalculateBurnRate(input2, cfg)
	expected := 60000.0 / 5.0 // 12000 t/m
	diff := info2.LocalTPM - expected
	if diff < -1 || diff > 1 {
		t.Errorf("LocalTPM = %f, want ≈%f", info2.LocalTPM, expected)
	}
}

func TestMergeLocalGlobal(t *testing.T) {
	local := types.BurnInfo{LocalTPM: 1000}
	global := types.BurnInfo{GlobalTPM: 1500}

	merged := MergeLocalGlobal(local, global)
	if merged.IsHighActivity {
		t.Error("should not be high activity when global is close to local")
	}

	// High activity
	globalHigh := types.BurnInfo{GlobalTPM: 7000}
	mergedHigh := MergeLocalGlobal(local, globalHigh)
	if !mergedHigh.IsHighActivity {
		t.Error("should be high activity when global >> local + 5000")
	}
}
