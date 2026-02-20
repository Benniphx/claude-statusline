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

func TestParseCostNormalizeSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
COST_NORMALIZE=true
COST_WEIGHT_OPUS=10.0
COST_WEIGHT_SONNET=1.0
COST_WEIGHT_HAIKU=0.5
`), 0o644)

	cfg := types.DefaultConfig()
	cfg.CostNormalize = false // Override default to test parsing
	parseFile(path, &cfg)

	if !cfg.CostNormalize {
		t.Error("CostNormalize should be true")
	}
	if cfg.CostWeightOpus != 10.0 {
		t.Errorf("CostWeightOpus = %f, want 10.0", cfg.CostWeightOpus)
	}
	if cfg.CostWeightSonnet != 1.0 {
		t.Errorf("CostWeightSonnet = %f, want 1.0", cfg.CostWeightSonnet)
	}
	if cfg.CostWeightHaiku != 0.5 {
		t.Errorf("CostWeightHaiku = %f, want 0.5", cfg.CostWeightHaiku)
	}
}

func TestParseCostNormalizeDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`COST_NORMALIZE=false`), 0o644)

	cfg := types.DefaultConfig() // Default is true
	parseFile(path, &cfg)

	if cfg.CostNormalize {
		t.Error("CostNormalize should be false when set to 'false'")
	}
}

func TestParseCostWeightInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(`
COST_WEIGHT_OPUS=0
COST_WEIGHT_HAIKU=-1
`), 0o644)

	cfg := types.DefaultConfig()
	parseFile(path, &cfg)

	// Invalid values should keep defaults
	if cfg.CostWeightOpus != 5.0 {
		t.Errorf("CostWeightOpus = %f, want 5.0 (default)", cfg.CostWeightOpus)
	}
	if cfg.CostWeightHaiku != 0.25 {
		t.Errorf("CostWeightHaiku = %f, want 0.25 (default)", cfg.CostWeightHaiku)
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		got := parseBool(tt.input)
		if got != tt.want {
			t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
