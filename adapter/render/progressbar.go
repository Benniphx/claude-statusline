package render

// MakeBar creates a visual progress bar with ANSI colors.
// percent is clamped to 0-100, width is number of characters.
func (a *ANSI) MakeBar(percent, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := percent * width / 100
	empty := width - filled

	var color string
	switch {
	case percent < 50:
		color = Green
	case percent < 80:
		color = Yellow
	default:
		color = Red
	}

	bar := color
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	bar += DimCode
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	bar += Reset

	return bar
}

// MakeSplitBar creates a dual-layer progress bar using Unicode half-block characters.
// Top half (▀) shows usage percentage, bottom half (▄) shows time elapsed percentage.
// When usage is ahead of time → consuming too fast. When time is ahead → on track.
//
// Characters:
//   █ = both filled (usage AND time)
//   ▀ = only usage (top) — consuming ahead of time
//   ▄ = only time (bottom) — time ahead of usage (good)
//   ░ = neither filled
func (a *ANSI) MakeSplitBar(usagePct, timePct, width int) string {
	usagePct = clamp(usagePct, 0, 100)
	timePct = clamp(timePct, 0, 100)

	usageFilled := usagePct * width / 100
	timeFilled := timePct * width / 100

	usageColor := ColorForPercent(usagePct)

	var bar string
	for i := 0; i < width; i++ {
		hasUsage := i < usageFilled
		hasTime := i < timeFilled

		switch {
		case hasUsage && hasTime:
			// Both: ▀ with fg=usage color (top), bg=blue (bottom)
			bar += usageColor + BgBlue + "▀" + Reset
		case hasUsage && !hasTime:
			// Only usage (top half): ▀ in usage color — burning ahead
			bar += usageColor + "▀" + Reset
		case !hasUsage && hasTime:
			// Only time (bottom half): ▄ in blue — time ahead (good)
			bar += Blue + "▄" + Reset
		default:
			// Neither: empty
			bar += DimCode + "░" + Reset
		}
	}

	return bar
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
