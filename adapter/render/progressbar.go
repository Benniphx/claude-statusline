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
