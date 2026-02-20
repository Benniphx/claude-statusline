package ratelimit

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/Benniphx/claude-statusline/adapter/render"
	"github.com/Benniphx/claude-statusline/core/cost"
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

	// Try cache first
	if data, fresh := store.ReadIfFresh(cachePath, cfg.RateCacheTTL); fresh {
		var resp types.RateLimitResponse
		if err := json.Unmarshal(data, &resp); err == nil {
			return parseResponse(&resp)
		}
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
func RenderSections(input types.Input, creds types.Credentials, cfg types.Config, plat ports.PlatformInfo, store ports.CacheStore, api ports.APIClient, r ports.Renderer, costTier types.CostTier) RateSections {
	data, err := Load(creds, cfg, store, api)
	if err != nil {
		return RateSections{
			FiveHour: "5h: " + r.Dim("--"),
			Burn:     "🔥 " + r.Dim("--"),
			SevenDay: "7d: " + r.Dim("--"),
		}
	}

	pace := CalculatePace(data, cfg, plat)
	localBurn := CalculateBurnRate(input, cfg)
	globalBurn := LoadBurnRate(cfg, store)
	burn := MergeLocalGlobal(localBurn, globalBurn)

	// Apply cost normalization to pace
	normFiveHourPace, _ := cost.NormalizePace(pace.FiveHourPace, costTier, cfg)
	normSevenDayPace, _ := cost.NormalizePace(pace.SevenDayPace, costTier, cfg)
	normPace := pace
	normPace.FiveHourPace = normFiveHourPace
	normPace.SevenDayPace = normSevenDayPace

	isNormalized := cost.IsNonBaseline(costTier) && cfg.CostNormalize

	return RateSections{
		FiveHour: renderFiveHour(data, normPace, r, isNormalized),
		Burn:     renderBurn(burn, r, costTier, cfg),
		SevenDay: renderSevenDay(data, normPace, r, isNormalized),
	}
}

func renderFiveHour(data types.RateLimitData, pace types.PaceInfo, r ports.Renderer, isNormalized bool) string {
	fivePct := int(math.Round(data.FiveHourPercent))
	bar := r.MakeBar(fivePct, 8)
	rateDisplay := r.Colorize(fmt.Sprintf("%d%%", fivePct), fivePct)

	// Pace (with ≈ prefix if normalized)
	if pace.FiveHourPace > 0 {
		prefix := ""
		if isNormalized {
			prefix = "≈"
		}
		rateDisplay += " " + prefix + paceColorize(pace.FiveHourPace, r)
	}

	// Hitting limit warning
	if pace.HittingLimit {
		rateDisplay += " " + r.Color("⚠️", render.Red)
	}

	// Reset time info (already contains →, cyan coloring)
	if pace.ResetInfo != "" {
		rateDisplay += " " + pace.ResetInfo
	}

	return fmt.Sprintf("5h: %s %s", bar, rateDisplay)
}

func renderBurn(burn types.BurnInfo, r ports.Renderer, costTier types.CostTier, cfg types.Config) string {
	localTPM := int(burn.LocalTPM)

	if localTPM > 0 {
		normTPM, isNormalized := cost.NormalizeBurnRate(float64(localTPM), costTier, cfg)
		tpmStr := r.FormatTokensF(int(normTPM))
		prefix := ""
		if isNormalized {
			prefix = "≈"
		}
		display := fmt.Sprintf("🔥 %s%s %s", prefix, r.Color(tpmStr, render.Magenta), r.Dim("t/m"))
		if burn.IsHighActivity {
			display += " " + r.Color("⚡", render.Yellow)
		}
		return display
	}

	if burn.IsHighActivity {
		return "🔥 " + r.Dim("--") + " " + r.Color("⚡", render.Yellow)
	}

	return "🔥 " + r.Dim("--")
}

func renderSevenDay(data types.RateLimitData, pace types.PaceInfo, r ports.Renderer, isNormalized bool) string {
	sevenPct := int(math.Round(data.SevenDayPercent))
	bar := r.MakeBar(sevenPct, 8)
	display := r.Colorize(fmt.Sprintf("%d%%", sevenPct), sevenPct)

	// Pace (with ≈ prefix if normalized)
	if pace.SevenDayPace > 0 {
		prefix := ""
		if isNormalized {
			prefix = "≈"
		}
		display += " " + prefix + paceColorize(pace.SevenDayPace, r)
	}

	// 7-day warning (pace > 1.0, using normalized value)
	if pace.SevenDayPace > 1.0 {
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

func paceColorize(pace float64, r ports.Renderer) string {
	text := fmt.Sprintf("%.1fx", pace)
	switch {
	case pace < 1.0:
		return r.Color(text, render.Green)
	case pace < 1.5:
		return r.Color(text, render.Yellow)
	default:
		return r.Color(text, render.Red)
	}
}
