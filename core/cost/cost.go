package cost

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Benniphx/claude-statusline/adapter/render"
	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

// CostSections holds the rendered cost display sections.
type CostSections struct {
	Session string // "💰 $0.50"
	Daily   string // "📅 $3.25"
	Burn    string // "🔥 1.2K t/m $1.20/h"
}

// Track computes session and daily cost using delta or replace accounting.
func Track(input types.Input, cfg types.Config, plat ports.PlatformInfo, store ports.CacheStore) types.CostDisplay {
	sessionID, stable := ResolveSession(plat)
	cost := input.Cost.TotalCostUSD

	today := time.Now().Format("2006-01-02")
	trackerPath := fmt.Sprintf("%s/claude_daily_cost_%s.txt", cfg.CacheDir, today)
	totalPath := fmt.Sprintf("%s/claude_session_total_%s.txt", cfg.CacheDir, sessionID)

	var sessionCost float64

	if stable {
		sessionCost = deltaAccounting(cost, sessionID, totalPath, trackerPath, store)
	} else {
		sessionCost = replaceAccounting(cost, sessionID, trackerPath, store)
	}

	dailyCost := sumTracker(trackerPath, store)

	var costPerHour float64
	if input.Cost.TotalDurationMS > 60000 {
		hours := float64(input.Cost.TotalDurationMS) / 3600000.0
		costPerHour = cost / hours
	}

	return types.CostDisplay{
		SessionCost: sessionCost,
		DailyCost:   dailyCost,
		CostPerHour: costPerHour,
		SessionID:   sessionID,
	}
}

func deltaAccounting(cost float64, sessionID, totalPath, trackerPath string, store ports.CacheStore) float64 {
	var lastKnown float64
	if data, err := store.ReadFile(totalPath); err == nil {
		lastKnown, _ = strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	}

	delta := cost - lastKnown
	if delta < 0 {
		delta = cost // New session or reset
	}

	store.WriteFile(totalPath, []byte(fmt.Sprintf("%.6f", cost)))
	updateTracker(trackerPath, sessionID, delta, true, store)

	return cost
}

func replaceAccounting(cost float64, sessionID, trackerPath string, store ports.CacheStore) float64 {
	updateTracker(trackerPath, sessionID, cost, false, store)
	return cost
}

func updateTracker(path, sessionID string, value float64, accumulate bool, store ports.CacheStore) {
	entries := readTracker(path, store)

	if accumulate {
		prev := entries[sessionID]
		entries[sessionID] = prev + value
	} else {
		entries[sessionID] = value
	}

	writeTracker(path, entries, store)
}

func readTracker(path string, store ports.CacheStore) map[string]float64 {
	entries := make(map[string]float64)

	data, err := store.ReadFile(path)
	if err != nil {
		return entries
	}

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		key, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			continue
		}
		entries[strings.TrimSpace(key)] = f
	}

	return entries
}

func writeTracker(path string, entries map[string]float64, store ports.CacheStore) {
	var lines []string
	for k, v := range entries {
		lines = append(lines, fmt.Sprintf("%s:%.6f", k, v))
	}
	store.WriteFile(path, []byte(strings.Join(lines, "\n")))
}

func sumTracker(path string, store ports.CacheStore) float64 {
	entries := readTracker(path, store)
	var total float64
	for _, v := range entries {
		total += v
	}
	return total
}

// RenderSections produces the three cost display sections for assembly by main.go.
func RenderSections(input types.Input, cfg types.Config, plat ports.PlatformInfo, store ports.CacheStore, r ports.Renderer, costTier types.CostTier) CostSections {
	display := Track(input, cfg, plat, store)

	// Local burn rate
	var localTPM int
	if input.Cost.TotalDurationMS > 60000 {
		totalTokens := input.ContextWindow.CurrentUsage.InputTokens +
			input.ContextWindow.CurrentUsage.CacheCreationInputTokens +
			input.ContextWindow.CurrentUsage.CacheReadInputTokens
		if totalTokens == 0 {
			totalTokens = input.ContextWindow.TotalInputTokens + input.ContextWindow.TotalOutputTokens
		}
		minutes := float64(input.Cost.TotalDurationMS) / 60000.0
		localTPM = int(float64(totalTokens) / minutes)
	}

	// Session cost with color thresholds: <$0.50 green, <$2.00 yellow, ≥$2.00 red
	sessionColor := costColor(display.SessionCost, 0.50, 2.00)
	sessionStr := fmt.Sprintf("💰 %s", r.Color(fmt.Sprintf("$%.2f", display.SessionCost), sessionColor))

	// Daily cost with color thresholds: <$5 green, <$20 yellow, ≥$20 red
	dailyColor := costColor(display.DailyCost, 5.00, 20.00)
	dailyStr := fmt.Sprintf("📅 %s", r.Color(fmt.Sprintf("$%.2f", display.DailyCost), dailyColor))

	// Apply cost normalization to burn rate and cost/hour
	normTPM, isNormalized := NormalizeBurnRate(float64(localTPM), costTier, cfg)
	normCPH, _ := NormalizeCostPerHour(display.CostPerHour, costTier, cfg)

	// Burn: "TPM t/m $X.XX/h" or "--"
	var burnStr string
	if localTPM > 0 {
		tpmFmt := r.FormatTokensF(int(normTPM))
		burnColor := costColor(normCPH, 1.00, 5.00)
		prefix := ""
		if isNormalized {
			prefix = "≈"
		}
		burnStr = fmt.Sprintf("🔥 %s%s %s %s%s%s",
			prefix,
			r.Color(tpmFmt, render.Magenta),
			r.Dim("t/m"),
			prefix,
			r.Color(fmt.Sprintf("$%.2f", normCPH), burnColor),
			r.Dim("/h"))
	} else {
		burnStr = "🔥 " + r.Dim("--")
	}

	return CostSections{
		Session: sessionStr,
		Daily:   dailyStr,
		Burn:    burnStr,
	}
}

// Render produces the full cost string (legacy compatibility).
func Render(input types.Input, cfg types.Config, plat ports.PlatformInfo, store ports.CacheStore, r ports.Renderer) string {
	sections := RenderSections(input, cfg, plat, store, r, types.CostTierSonnet)
	sep := "  " + r.Dim("│") + "  "
	return sections.Session + sep + sections.Daily + sep + sections.Burn
}

func costColor(cost, warnThreshold, critThreshold float64) string {
	switch {
	case cost < warnThreshold:
		return render.Green
	case cost < critThreshold:
		return render.Yellow
	default:
		return render.Red
	}
}
