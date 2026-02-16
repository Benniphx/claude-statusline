package render

import "fmt"

// ANSI color codes.
const (
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Cyan    = "\033[36m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	DimCode = "\033[2m"
	Reset   = "\033[0m"
)

// ANSI implements the ports.Renderer interface.
type ANSI struct{}

// New creates a new ANSI renderer.
func New() *ANSI {
	return &ANSI{}
}

// Colorize applies color based on percentage thresholds: <50 green, <80 yellow, >=80 red.
func (a *ANSI) Colorize(text string, percent int) string {
	var color string
	switch {
	case percent < 50:
		color = Green
	case percent < 80:
		color = Yellow
	default:
		color = Red
	}
	return color + text + Reset
}

// ColorForPercent returns the color code for a percentage threshold.
func ColorForPercent(percent int) string {
	switch {
	case percent < 50:
		return Green
	case percent < 80:
		return Yellow
	default:
		return Red
	}
}

// Color wraps text with the given color code.
func (a *ANSI) Color(text, color string) string {
	return color + text + Reset
}

// Dim applies dim formatting.
func (a *ANSI) Dim(text string) string {
	return DimCode + text + Reset
}

// FormatTokens formats a token count with 0 decimals (for max/total context).
// 200000 → "200K", 32768 → "33K", 500 → "500".
func (a *ANSI) FormatTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.0fK", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

// FormatTokensF formats a token count with 1 decimal (for used tokens, burn rate).
// 100000 → "100.0K", 1500 → "1.5K", 500 → "500".
func (a *ANSI) FormatTokensF(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

// FormatCost formats a cost as "$X.XX".
func (a *ANSI) FormatCost(f float64) string {
	return fmt.Sprintf("$%.2f", f)
}
