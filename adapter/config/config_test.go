package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

func TestParseValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
CONTEXT_WARNING_THRESHOLD=75
RATE_CACHE_TTL=30
WORK_DAYS_PER_WEEK=4
`), 0o644)

	cfg := types.DefaultConfig()
	parseFile(path, &cfg)

	if cfg.ContextWarningThreshold != 75 {
		t.Errorf("ContextWarningThreshold = %d, want 75", cfg.ContextWarningThreshold)
	}
	if cfg.RateCacheTTL != 30*time.Second {
		t.Errorf("RateCacheTTL = %v, want 30s", cfg.RateCacheTTL)
	}
	if cfg.WorkDaysPerWeek != 4 {
		t.Errorf("WorkDaysPerWeek = %d, want 4", cfg.WorkDaysPerWeek)
	}
}

func TestParseInvalidRangesUseDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
CONTEXT_WARNING_THRESHOLD=0
RATE_CACHE_TTL=5
WORK_DAYS_PER_WEEK=8
`), 0o644)

	cfg := types.DefaultConfig()
	parseFile(path, &cfg)

	if cfg.ContextWarningThreshold != 0 { // 0 is default (disabled), invalid 0 doesn't change it
		t.Errorf("ContextWarningThreshold = %d, want 0", cfg.ContextWarningThreshold)
	}
	if cfg.RateCacheTTL != 15*time.Second { // 5 is below min 10
		t.Errorf("RateCacheTTL = %v, want 15s", cfg.RateCacheTTL)
	}
	if cfg.WorkDaysPerWeek != 5 { // 8 is above max 7
		t.Errorf("WorkDaysPerWeek = %d, want 5", cfg.WorkDaysPerWeek)
	}
}

func TestParseMissingFile(t *testing.T) {
	cfg := types.DefaultConfig()
	found := parseFile("/nonexistent/path", &cfg)

	if found {
		t.Error("should return false for missing file")
	}
	if cfg.WorkDaysPerWeek != 5 {
		t.Error("defaults should be unchanged")
	}
}

func TestParseWithComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
# This is a comment
WORK_DAYS_PER_WEEK=3

# Another comment
`), 0o644)

	cfg := types.DefaultConfig()
	parseFile(path, &cfg)

	if cfg.WorkDaysPerWeek != 3 {
		t.Errorf("WorkDaysPerWeek = %d, want 3", cfg.WorkDaysPerWeek)
	}
}
