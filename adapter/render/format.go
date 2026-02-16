package render

import "fmt"

// FormatDuration converts milliseconds to a human-readable duration string.
func (a *ANSI) FormatDuration(ms int) string {
	min := ms / 60000
	return fmt.Sprintf("%dm", min)
}
