package ratelimit

import (
	"encoding/json"
	"testing"
	"time"

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

func TestCalculateGlobalBurnFromStdin_FirstCall(t *testing.T) {
	store := newMockCache()
	cfg := types.DefaultConfig()

	// First call: no previous snapshot → TPM=0, but snapshot is written
	info := CalculateGlobalBurnFromStdin(42.0, cfg, store)
	if info.GlobalTPM != 0 {
		t.Errorf("First call should have GlobalTPM=0, got %f", info.GlobalTPM)
	}

	// Verify snapshot was written
	cachePath := cfg.CacheDir + "/" + globalBurnFile
	if _, ok := store.files[cachePath]; !ok {
		t.Error("Snapshot should have been written to cache")
	}
}

func TestCalculateGlobalBurnFromStdin_WithDelta(t *testing.T) {
	store := newMockCache()
	cfg := types.DefaultConfig()
	cachePath := cfg.CacheDir + "/" + globalBurnFile

	// Seed a previous snapshot: 40% at 60 seconds ago
	prev := burnSnapshot{
		FiveHourPct:  40.0,
		Timestamp:    time.Now().Unix() - 60, // 60s ago
		TokensPerMin: 0,
	}
	raw, _ := json.Marshal(prev)
	store.files[cachePath] = raw

	// Now at 42% → delta = 2% in 60s = 2%/min → 10000 t/m
	info := CalculateGlobalBurnFromStdin(42.0, cfg, store)
	if info.GlobalTPM < 9000 || info.GlobalTPM > 11000 {
		t.Errorf("GlobalTPM should be ~10000 (2%%/min * 5000), got %f", info.GlobalTPM)
	}
}

func TestCalculateGlobalBurnFromStdin_Decay(t *testing.T) {
	store := newMockCache()
	cfg := types.DefaultConfig()
	cachePath := cfg.CacheDir + "/" + globalBurnFile

	// Seed: 40% at 60s ago with 10000 TPM
	prev := burnSnapshot{
		FiveHourPct:  40.0,
		Timestamp:    time.Now().Unix() - 60,
		TokensPerMin: 10000,
	}
	raw, _ := json.Marshal(prev)
	store.files[cachePath] = raw

	// Same % → no delta → decay by 50%
	info := CalculateGlobalBurnFromStdin(40.0, cfg, store)
	if info.GlobalTPM < 4500 || info.GlobalTPM > 5500 {
		t.Errorf("GlobalTPM should decay to ~5000 (50%% of 10000), got %f", info.GlobalTPM)
	}
}

func TestCalculateGlobalBurnFromStdin_Smoothing(t *testing.T) {
	store := newMockCache()
	cfg := types.DefaultConfig()
	cachePath := cfg.CacheDir + "/" + globalBurnFile

	// Seed: 40% at 60s ago with 8000 TPM
	prev := burnSnapshot{
		FiveHourPct:  40.0,
		Timestamp:    time.Now().Unix() - 60,
		TokensPerMin: 8000,
	}
	raw, _ := json.Marshal(prev)
	store.files[cachePath] = raw

	// Now at 42% → new = 10000, smoothed = 0.8*10000 + 0.2*8000 = 9600
	info := CalculateGlobalBurnFromStdin(42.0, cfg, store)
	if info.GlobalTPM < 9000 || info.GlobalTPM > 10000 {
		t.Errorf("GlobalTPM should be ~9600 (smoothed), got %f", info.GlobalTPM)
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
