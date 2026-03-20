package ratelimit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

const globalBurnFile = "claude_global_burn.json"
const tokensPerPercent = 5000 // 1% of 5h ≈ 5000 tokens

// burnSnapshot persists the last known 5h% and timestamp for delta calculation.
type burnSnapshot struct {
	FiveHourPct  float64 `json:"five_hour_percent"`
	Timestamp    int64   `json:"timestamp"`
	TokensPerMin float64 `json:"tokens_per_min"`
}

// CalculateBurnRate computes local burn rate from input data.
func CalculateBurnRate(input types.Input, cfg types.Config) types.BurnInfo {
	var info types.BurnInfo

	// Local t/m: tokens / duration
	durationMS := input.Cost.TotalDurationMS
	if durationMS > 60000 { // Need at least 1 minute of data
		totalTokens := input.ContextWindow.CurrentUsage.InputTokens +
			input.ContextWindow.CurrentUsage.CacheCreationInputTokens +
			input.ContextWindow.CurrentUsage.CacheReadInputTokens
		if totalTokens == 0 {
			totalTokens = input.ContextWindow.TotalInputTokens + input.ContextWindow.TotalOutputTokens
		}

		minutes := float64(durationMS) / 60000.0
		info.LocalTPM = float64(totalTokens) / minutes
	}

	return info
}

// CalculateGlobalBurnFromStdin computes account-wide burn rate from stdin rate_limits deltas.
// On each render, it reads the previous snapshot, calculates the TPM delta, and writes
// the new snapshot. This replaces the daemon for global burn rate tracking.
func CalculateGlobalBurnFromStdin(currentPct float64, cfg types.Config, store ports.CacheStore) types.BurnInfo {
	var info types.BurnInfo
	cachePath := fmt.Sprintf("%s/%s", cfg.CacheDir, globalBurnFile)
	now := time.Now().Unix()

	// Read previous snapshot
	var prev burnSnapshot
	if data, err := store.ReadFile(cachePath); err == nil {
		json.Unmarshal(data, &prev)
	}

	// Write current snapshot (always, even on first call)
	snap := burnSnapshot{
		FiveHourPct: currentPct,
		Timestamp:   now,
	}

	if prev.Timestamp > 0 {
		deltaSecs := float64(now - prev.Timestamp)
		if deltaSecs > 0 {
			deltaPct := currentPct - prev.FiveHourPct

			if deltaPct <= 0 {
				// No change or decrease: decay previous TPM
				snap.TokensPerMin = prev.TokensPerMin * 0.5
			} else {
				// Active: calculate new TPM from delta
				pctPerMin := (deltaPct / deltaSecs) * 60.0
				newTPM := pctPerMin * float64(tokensPerPercent)

				// Light smoothing: 80% new, 20% old
				if prev.TokensPerMin > 0 {
					snap.TokensPerMin = newTPM*0.8 + prev.TokensPerMin*0.2
				} else {
					snap.TokensPerMin = newTPM
				}
			}
		} else {
			snap.TokensPerMin = prev.TokensPerMin
		}
	}

	// Persist snapshot
	if raw, err := json.Marshal(snap); err == nil {
		store.AtomicWrite(cachePath, raw)
	}

	info.GlobalTPM = snap.TokensPerMin
	return info
}

// LoadBurnRate reads the global burn rate from cache (works with both daemon and stdin snapshots).
func LoadBurnRate(cfg types.Config, store ports.CacheStore) types.BurnInfo {
	var info types.BurnInfo

	cachePath := fmt.Sprintf("%s/%s", cfg.CacheDir, globalBurnFile)
	data, err := store.ReadFile(cachePath)
	if err != nil {
		return info
	}

	var snap burnSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return info
	}

	info.GlobalTPM = snap.TokensPerMin
	return info
}

// MergeLocalGlobal combines local and global burn rate info.
func MergeLocalGlobal(local, global types.BurnInfo) types.BurnInfo {
	merged := local
	merged.GlobalTPM = global.GlobalTPM

	// High activity: global >> local + 5000
	if global.GlobalTPM > local.LocalTPM+5000 {
		merged.IsHighActivity = true
	}

	return merged
}
