package context

import (
	"fmt"

	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

// Calculate computes context window display data from input and model info.
func Calculate(input types.Input, model types.ModelInfo, cfg types.Config) types.ContextDisplay {
	cw := input.ContextWindow

	// Determine context size
	contextSize := cw.ContextWindowSize
	if contextSize <= 0 {
		contextSize = model.DefaultContext
	}

	// Calculate total tokens used
	// Prefer current_usage (accurate for current context)
	totalTokens := cw.CurrentUsage.InputTokens +
		cw.CurrentUsage.CacheCreationInputTokens +
		cw.CurrentUsage.CacheReadInputTokens

	// Fallback to cumulative if current_usage is empty
	if totalTokens == 0 {
		totalTokens = cw.TotalInputTokens + cw.TotalOutputTokens
	}

	// Cap at context size for display
	displayTokens := totalTokens
	if displayTokens > contextSize {
		displayTokens = contextSize
	}

	// Calculate percentage
	var percent int
	if cw.UsedPercentage != nil && *cw.UsedPercentage > 0 && totalTokens > 0 {
		// API provides percentage — but verify plausibility against token count
		apiPercent := int(*cw.UsedPercentage)
		calcPercent := 0
		if contextSize > 0 {
			calcPercent = displayTokens * 100 / contextSize
		}
		// If API says >50% but tokens say <5%, the API value is stale (e.g. after clear/compact)
		if apiPercent > 50 && calcPercent < 5 {
			percent = calcPercent
		} else {
			percent = apiPercent
		}
	} else if contextSize > 0 {
		percent = displayTokens * 100 / contextSize
	}

	if percent > 100 {
		percent = 100
	}

	// Check for initial state (no tokens, or 0% with 0 duration)
	// Also treat as initial when API reports percentage but tokens are 0 (stale after clear/compact)
	isInitial := totalTokens == 0 || (percent == 0 && input.Cost.TotalDurationMS == 0)

	durationMin := input.Cost.TotalDurationMS / 60000

	return types.ContextDisplay{
		PercentUsed:  percent,
		TokensUsed:   displayTokens,
		TokensTotal:  contextSize,
		IsInitial:    isInitial,
		LinesAdded:   input.Cost.TotalLinesAdded,
		LinesRemoved: input.Cost.TotalLinesRemoved,
		DurationMin:  durationMin,
	}
}

// Render produces the context section string (bar + percentage + tokens only).
// Does NOT include model name, duration, or lines — those are assembled by main.go.
func Render(display types.ContextDisplay, cfg types.Config, r ports.Renderer) string {
	bar := r.MakeBar(display.PercentUsed, 8)

	// Percentage or "--" for initial state
	var pctStr string
	if display.IsInitial {
		pctStr = r.Colorize("--", display.PercentUsed)
	} else {
		pctStr = r.Colorize(fmt.Sprintf("%d%%", display.PercentUsed), display.PercentUsed)
	}

	// Warning threshold
	warning := ""
	if !display.IsInitial && cfg.ContextWarningThreshold > 0 && display.PercentUsed >= cfg.ContextWarningThreshold {
		warning = " ⚠️"
	}

	// Token counts: used gets 1 decimal, max gets 0 decimals
	tokens := r.FormatTokensF(display.TokensUsed)
	total := r.FormatTokens(display.TokensTotal)

	return fmt.Sprintf("Ctx: %s %s%s %s", bar, pctStr, warning, r.Dim(fmt.Sprintf("(%s/%s)", tokens, total)))
}
