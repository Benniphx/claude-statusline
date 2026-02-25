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

func TestParseCostNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
COST_NORMALIZE=false
COST_WEIGHT_HAIKU=0.20
COST_WEIGHT_SONNET=1.5
COST_WEIGHT_OPUS=8.0
`), 0o644)

	cfg := types.DefaultConfig()
	parseFile(path, &cfg)

	if cfg.CostNormalize {
		t.Error("CostNormalize should be false")
	}
	if cfg.CostWeightHaiku != 0.20 {
		t.Errorf("CostWeightHaiku = %f, want 0.20", cfg.CostWeightHaiku)
	}
	if cfg.CostWeightSonnet != 1.5 {
		t.Errorf("CostWeightSonnet = %f, want 1.5", cfg.CostWeightSonnet)
	}
	if cfg.CostWeightOpus != 8.0 {
		t.Errorf("CostWeightOpus = %f, want 8.0", cfg.CostWeightOpus)
	}
}

func TestParseCostNormalizeTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
COST_NORMALIZE=true
`), 0o644)

	cfg := types.DefaultConfig()
	cfg.CostNormalize = false // Override default to test parsing
	parseFile(path, &cfg)

	if !cfg.CostNormalize {
		t.Error("CostNormalize should be true")
	}
}

func TestParseCostWeightInvalidIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
COST_WEIGHT_OPUS=0
COST_WEIGHT_HAIKU=-1
COST_WEIGHT_SONNET=abc
`), 0o644)

	cfg := types.DefaultConfig()
	parseFile(path, &cfg)

	// Invalid values should not change defaults
	if cfg.CostWeightOpus != 5.0 {
		t.Errorf("CostWeightOpus = %f, want 5.0 (default)", cfg.CostWeightOpus)
	}
	if cfg.CostWeightHaiku != 0.25 {
		t.Errorf("CostWeightHaiku = %f, want 0.25 (default)", cfg.CostWeightHaiku)
	}
	if cfg.CostWeightSonnet != 1.0 {
		t.Errorf("CostWeightSonnet = %f, want 1.0 (default)", cfg.CostWeightSonnet)
	}
}
