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

// MakeSplitBar creates a layered progress bar with usage and time indicators.
// Style parameter selects the visual approach.
func (a *ANSI) MakeSplitBar(usagePct, timePct, width int) string {
	return a.MakeSplitBarStyled(usagePct, timePct, width, "thin")
}

// MakeSplitBarStyled renders the split bar in different visual styles (for testing).
func (a *ANSI) MakeSplitBarStyled(usagePct, timePct, width int, style string) string {
	usagePct = clamp(usagePct, 0, 100)
	timePct = clamp(timePct, 0, 100)

	usageFilled := usagePct * width / 100
	timeFilled := timePct * width / 100

	usageColor := ColorForPercent(usagePct)

	var bar string
	for i := 0; i < width; i++ {
		hasUsage := i < usageFilled
		hasTime := i < timeFilled

		switch style {
		case "thin":
			// Style A: ▇ (7/8 block) with red bg where both overlap, ▁ red for time-only
			// The thin gap at the bottom of ▇ lets the red background show through
			switch {
			case hasUsage && hasTime:
				bar += usageColor + BgRed + "▇" + Reset
			case hasUsage:
				bar += usageColor + "█" + Reset
			case hasTime:
				bar += Red + "▁" + Reset
			default:
				bar += DimCode + "░" + Reset
			}

		case "bg":
			// Style B: Normal █ for usage, blue background on ░ for time
			switch {
			case hasUsage:
				bar += usageColor + "█" + Reset
			case hasTime:
				bar += BgBlue + DimCode + "░" + Reset
			default:
				bar += DimCode + "░" + Reset
			}

		case "lower-quarter":
			// Style C: Normal █ for usage, ▂ (lower quarter) in cyan for time
			switch {
			case hasUsage:
				bar += usageColor + "█" + Reset
			case hasTime:
				bar += Cyan + "▂" + Reset
			default:
				bar += DimCode + "░" + Reset
			}

		case "lower-half-dim":
			// Style D: Normal █ for usage, dim ▄ for time (subtle)
			switch {
			case hasUsage:
				bar += usageColor + "█" + Reset
			case hasTime:
				bar += DimCode + Blue + "▄" + Reset
			default:
				bar += DimCode + "░" + Reset
			}

		case "dot":
			// Style E: Normal █ for usage, · dots in blue for time
			switch {
			case hasUsage:
				bar += usageColor + "█" + Reset
			case hasTime:
				bar += Blue + "·" + Reset
			default:
				bar += DimCode + "░" + Reset
			}
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
