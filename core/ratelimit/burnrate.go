package ratelimit

import (
	"encoding/json"
	"fmt"

	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

const globalBurnFile = "claude_global_burn.json"
const tokensPerPercent = 5000 // 1% of 5h ≈ 5000 tokens

type globalBurnData struct {
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

// LoadBurnRate reads the global burn rate from daemon cache.
func LoadBurnRate(cfg types.Config, store ports.CacheStore) types.BurnInfo {
	var info types.BurnInfo

	cachePath := fmt.Sprintf("%s/%s", cfg.CacheDir, globalBurnFile)
	data, err := store.ReadFile(cachePath)
	if err != nil {
		return info
	}

	var burn globalBurnData
	if err := json.Unmarshal(data, &burn); err != nil {
		return info
	}

	info.GlobalTPM = burn.TokensPerMin

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
