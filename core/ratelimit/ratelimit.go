package ratelimit

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/Benniphx/claude-statusline/adapter/render"
	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

const cacheName = "claude_rate_limit_cache.json"

// RateSections holds the three rate limit display sections.
type RateSections struct {
	FiveHour string
	Burn     string
	SevenDay string
}

// Load retrieves rate limit data, using cache or fetching from the API.
func Load(creds types.Credentials, cfg types.Config, store ports.CacheStore, api ports.APIClient) (types.RateLimitData, error) {
	cachePath := fmt.Sprintf("%s/%s", cfg.CacheDir, cacheName)

	// STATUSLINE_NO_POLL=1 → only serve cache, never call API
	noPoll := os.Getenv("STATUSLINE_NO_POLL") == "1"

	// Try cache first
	if data, fresh := store.ReadIfFresh(cachePath, cfg.RateCacheTTL); fresh {
		var resp types.RateLimitResponse
		if err := json.Unmarshal(data, &resp); err == nil {
			return parseResponse(&resp)
		}
	}

	// If polling disabled, try stale cache then give up
	if noPoll {
		if data, err := store.ReadFile(cachePath); err == nil {
			var stale types.RateLimitResponse
			if err2 := json.Unmarshal(data, &stale); err2 == nil {
				result, _ := parseResponse(&stale)
				result.FromCache = true
				return result, nil
			}
		}
		return types.RateLimitData{}, fmt.Errorf("polling disabled (STATUSLINE_NO_POLL=1)")
	}

	// Fetch from API
	if !creds.HasOAuth() {
		return types.RateLimitData{}, fmt.Errorf("no OAuth credentials")
	}

	resp, err := api.FetchRateLimits(creds.OAuthToken)
	if err != nil {
		// Try stale cache as fallback
		if data, err2 := store.ReadFile(cachePath); err2 == nil {
			var stale types.RateLimitResponse
			if err3 := json.Unmarshal(data, &stale); err3 == nil {
				result, _ := parseResponse(&stale)
				result.FromCache = true
				return result, nil
			}
		}
		return types.RateLimitData{}, err
	}

	// Cache the response
	if raw, err := json.Marshal(resp); err == nil {
		store.AtomicWrite(cachePath, raw)
	}

	return parseResponse(resp)
}

func parseResponse(resp *types.RateLimitResponse) (types.RateLimitData, error) {
	fiveHourReset, _ := time.Parse(time.RFC3339, resp.FiveHour.ResetsAt)
	sevenDayReset, _ := time.Parse(time.RFC3339, resp.SevenDay.ResetsAt)

	return types.RateLimitData{
		FiveHourPercent: resp.FiveHour.Utilization,
		FiveHourReset:   fiveHourReset,
		SevenDayPercent: resp.SevenDay.Utilization,
		SevenDayReset:   sevenDayReset,
	}, nil
}

// RenderSections produces the three rate limit sections for assembly by main.go.
func RenderSections(input types.Input, creds types.Credentials, cfg types.Config, plat ports.PlatformInfo, store ports.CacheStore, api ports.APIClient, r ports.Renderer, modelInfo types.ModelInfo) RateSections {
	data, err := Load(creds, cfg, store, api)
	if err != nil {
		return RateSections{
			FiveHour: "5h: " + r.Dim("--"),
			Burn:     "🔥 " + r.Dim("--"),
			SevenDay: "7d: " + r.Dim("--"),
		}
	}

	cn := types.ResolveCostNorm(cfg, modelInfo)

	pace := CalculatePace(data, cfg, plat)
	localBurn := CalculateBurnRate(input, cfg)
	globalBurn := LoadBurnRate(cfg, store)
	burn := MergeLocalGlobal(localBurn, globalBurn)

	return RateSections{
		FiveHour: renderFiveHour(data, pace, cn, r),
		Burn:     renderBurn(burn, cn, r),
		SevenDay: renderSevenDay(data, pace, cn, r),
	}
}

func renderFiveHour(data types.RateLimitData, pace types.PaceInfo, cn types.CostNorm, r ports.Renderer) string {
	fivePct := int(math.Round(data.FiveHourPercent))
	bar := r.MakeBar(fivePct, 8)
	rateDisplay := r.Colorize(fmt.Sprintf("%d%%", fivePct), fivePct)

	// Pace (cost-normalized)
	if pace.FiveHourPace > 0 {
		normalizedPace := pace.FiveHourPace * cn.Mult
		rateDisplay += " " + paceColorize(normalizedPace, cn.Prefix, r)
	}

	// Hitting limit warning (raw capacity, not cost-normalized)
	if pace.HittingLimit {
		rateDisplay += " " + r.Color("⚠️", render.Red)
	}

	// Reset time info (already contains →, cyan coloring)
	if pace.ResetInfo != "" {
		rateDisplay += " " + pace.ResetInfo
	}

	return fmt.Sprintf("5h: %s %s", bar, rateDisplay)
}

func renderBurn(burn types.BurnInfo, cn types.CostNorm, r ports.Renderer) string {
	normalizedTPM := int(math.Round(burn.LocalTPM * cn.Mult))
	normalizedGlobal := int(math.Round(burn.GlobalTPM * cn.Mult))

	if normalizedTPM > 0 {
		tpmStr := cn.Prefix + r.FormatTokensF(normalizedTPM)
		display := fmt.Sprintf("🔥 %s", r.Color(tpmStr, render.Magenta))

		// Show global total when other sessions/subagents are active
		if burn.IsHighActivity && normalizedGlobal > normalizedTPM {
			globalStr := cn.Prefix + r.FormatTokensF(normalizedGlobal)
			display += r.Dim("/") + r.Color(globalStr, render.Magenta)
		}

		display += " " + r.Dim("t/m")
		return display
	}

	// No local activity but others are active — show only global
	if burn.IsHighActivity && normalizedGlobal > 0 {
		globalStr := cn.Prefix + r.FormatTokensF(normalizedGlobal)
		return fmt.Sprintf("🔥 %s %s", r.Dim("--")+r.Dim("/")+r.Color(globalStr, render.Magenta), r.Dim("t/m"))
	}

	return "🔥 " + r.Dim("--")
}

func renderSevenDay(data types.RateLimitData, pace types.PaceInfo, cn types.CostNorm, r ports.Renderer) string {
	sevenPct := int(math.Round(data.SevenDayPercent))
	bar := r.MakeBar(sevenPct, 8)
	display := r.Colorize(fmt.Sprintf("%d%%", sevenPct), sevenPct)

	// Pace (cost-normalized)
	if pace.SevenDayPace > 0 {
		normalizedPace := pace.SevenDayPace * cn.Mult
		display += " " + paceColorize(normalizedPace, cn.Prefix, r)
	}

	// 7-day warning (cost-normalized budget sustainability)
	if pace.SevenDayPace*cn.Mult > 1.0 {
		display += " " + r.Color("⚠️", render.Red)
	}

	// Days left (only ≤3 days)
	if pace.SevenDayResetFmt != "" {
		display += " " + pace.SevenDayResetFmt
	}

	return fmt.Sprintf("7d: %s %s", bar, display)
}

// Render produces the full rate limit string (legacy, for empty stdin fallback).
func Render(creds types.Credentials, cfg types.Config, plat ports.PlatformInfo, store ports.CacheStore, api ports.APIClient, r ports.Renderer) string {
	data, err := Load(creds, cfg, store, api)
	if err != nil {
		return ""
	}
	fivePct := int(math.Round(data.FiveHourPercent))
	return fmt.Sprintf("5h: %s", r.Colorize(fmt.Sprintf("%d%%", fivePct), fivePct))
}

func paceColorize(pace float64, prefix string, r ports.Renderer) string {
	text := fmt.Sprintf("%s%.1fx", prefix, pace)
	switch {
	case pace < 1.0:
		return r.Color(text, render.Green)
	case pace < 1.5:
		return r.Color(text, render.Yellow)
	default:
		return r.Color(text, render.Red)
	}
}
