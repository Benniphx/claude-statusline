package platform

import (
	"os"
	"runtime"
	"time"
)

// Platform provides OS-specific operations and implements multiple port interfaces:
// ports.PlatformInfo, ports.CredentialProvider, ports.ProcessDetector.
type Platform struct {
	goos string
}

// Detect creates a Platform for the current OS.
func Detect() *Platform {
	return &Platform{goos: runtime.GOOS}
}

// ParseISODate parses an ISO 8601 / RFC 3339 date string.
func (p *Platform) ParseISODate(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// FormatTime formats a time using Go's reference time format.
func (p *Platform) FormatTime(t time.Time, format string) string {
	return t.Format(format)
}

// CountWorkDays counts the number of work days between start and end,
// scaled by workDaysPerWeek (e.g. 5 = Mon-Fri, 7 = all days).
func (p *Platform) CountWorkDays(start, end time.Time, workDaysPerWeek int) float64 {
	if workDaysPerWeek >= 7 {
		return end.Sub(start).Hours() / 24.0
	}

	days := end.Sub(start).Hours() / 24.0
	return days * float64(workDaysPerWeek) / 7.0
}

// GetStableSessionID attempts to find a stable session identifier.
// It checks the CLAUDE_SESSION_ID env var first, then walks the process tree.
func (p *Platform) GetStableSessionID() (string, bool) {
	if id := os.Getenv("CLAUDE_SESSION_ID"); id != "" {
		return id, true
	}
	return p.getSessionFromProcessTree()
}
