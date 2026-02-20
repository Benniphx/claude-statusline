package cost

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

// mockPlatform implements ports.PlatformInfo for testing.
type mockPlatform struct {
	sessionID string
	stable    bool
}

func (m *mockPlatform) ParseISODate(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
func (m *mockPlatform) FormatTime(t time.Time, format string) string {
	return t.Format(format)
}
func (m *mockPlatform) CountWorkDays(start, end time.Time, wdpw int) float64 {
	return end.Sub(start).Hours() / 24.0
}
func (m *mockPlatform) GetStableSessionID() (string, bool) {
	return m.sessionID, m.stable
}

// mockCache implements ports.CacheStore for testing.
type mockCache struct {
	files map[string][]byte
}

func newMockCache() *mockCache {
	return &mockCache{files: make(map[string][]byte)}
}

func (m *mockCache) AtomicWrite(path string, data []byte) error {
	m.files[path] = data
	return nil
}
func (m *mockCache) ReadIfFresh(path string, ttl time.Duration) ([]byte, bool) {
	return nil, false
}
func (m *mockCache) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return data, nil
}
func (m *mockCache) WriteFile(path string, data []byte) error {
	m.files[path] = data
	return nil
}
func (m *mockCache) FileMTime(path string) (time.Time, error) {
	return time.Now(), nil
}
func (m *mockCache) CleanOld(dir, pattern, keep string) error {
	return nil
}

func TestDeltaAccounting(t *testing.T) {
	store := newMockCache()
	plat := &mockPlatform{sessionID: "session1", stable: true}
	cfg := types.DefaultConfig()

	// First call: cost is 0.50
	input1 := types.Input{
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 120000},
	}
	display1 := Track(input1, cfg, plat, store)
	if display1.SessionCost != 0.50 {
		t.Errorf("SessionCost = %f, want 0.50", display1.SessionCost)
	}

	// Second call: cost is 1.00 (delta = 0.50)
	input2 := types.Input{
		Cost: types.Cost{TotalCostUSD: 1.00, TotalDurationMS: 240000},
	}
	display2 := Track(input2, cfg, plat, store)
	if display2.SessionCost != 1.00 {
		t.Errorf("SessionCost = %f, want 1.00", display2.SessionCost)
	}

	// Daily cost should be accumulated deltas
	if display2.DailyCost < 0.99 || display2.DailyCost > 1.01 {
		t.Errorf("DailyCost = %f, want ≈1.00", display2.DailyCost)
	}
}

func TestReplaceAccounting(t *testing.T) {
	store := newMockCache()
	plat := &mockPlatform{sessionID: "unstable1", stable: false}
	cfg := types.DefaultConfig()

	// First call
	input1 := types.Input{
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 120000},
	}
	display1 := Track(input1, cfg, plat, store)
	if display1.SessionCost != 0.50 {
		t.Errorf("SessionCost = %f, want 0.50", display1.SessionCost)
	}

	// Second call: replaces, not accumulates
	input2 := types.Input{
		Cost: types.Cost{TotalCostUSD: 1.00, TotalDurationMS: 240000},
	}
	display2 := Track(input2, cfg, plat, store)
	// Daily should be the replaced value, not accumulated
	if display2.DailyCost != 1.00 {
		t.Errorf("DailyCost = %f, want 1.00", display2.DailyCost)
	}
}

func TestCostPerHour(t *testing.T) {
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	input := types.Input{
		Cost: types.Cost{
			TotalCostUSD:    2.00,
			TotalDurationMS: 3600000, // 1 hour
		},
	}

	display := Track(input, cfg, plat, store)
	if display.CostPerHour < 1.99 || display.CostPerHour > 2.01 {
		t.Errorf("CostPerHour = %f, want ≈2.00", display.CostPerHour)
	}
}

func TestNegativeDelta(t *testing.T) {
	store := newMockCache()
	plat := &mockPlatform{sessionID: "session1", stable: true}
	cfg := types.DefaultConfig()

	// Set last known higher than current (new session scenario)
	totalPath := fmt.Sprintf("%s/claude_session_total_%s.txt", cfg.CacheDir, "session1")
	store.WriteFile(totalPath, []byte("5.000000"))

	input := types.Input{
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 60000},
	}

	display := Track(input, cfg, plat, store)
	// Delta should be 0 (negative clamped)
	if display.DailyCost < 0 {
		t.Errorf("DailyCost should not be negative: %f", display.DailyCost)
	}
}

func TestRenderCost(t *testing.T) {
	// Create a simple mock renderer
	r := &mockRenderer{}

	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	input := types.Input{
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 120000},
	}

	result := Render(input, cfg, plat, store, r)
	if !strings.Contains(result, "💰") {
		t.Error("should contain session cost emoji")
	}
	if !strings.Contains(result, "📅") {
		t.Error("should contain daily cost emoji")
	}
}

type mockRenderer struct{}

func (m *mockRenderer) Colorize(text string, percent int) string { return text }
func (m *mockRenderer) Color(text, color string) string          { return text }
func (m *mockRenderer) Dim(text string) string                   { return text }
func (m *mockRenderer) MakeBar(percent, width int) string        { return "[bar]" }
func (m *mockRenderer) FormatTokens(n int) string                { return fmt.Sprintf("%d", n) }
func (m *mockRenderer) FormatTokensF(n int) string               { return fmt.Sprintf("%d", n) }
func (m *mockRenderer) FormatCost(f float64) string              { return fmt.Sprintf("$%.2f", f) }

func TestCostColor(t *testing.T) {
	tests := []struct {
		name      string
		cost      float64
		warn      float64
		crit      float64
		wantColor string
	}{
		// Session thresholds: <$0.50 green, <$2.00 yellow, >=$2.00 red
		{"session_green", 0.15, 0.50, 2.00, "\033[32m"},
		{"session_yellow", 1.25, 0.50, 2.00, "\033[33m"},
		{"session_red", 8.50, 0.50, 2.00, "\033[31m"},
		{"session_boundary_warn", 0.50, 0.50, 2.00, "\033[33m"},
		{"session_boundary_crit", 2.00, 0.50, 2.00, "\033[31m"},
		// Daily thresholds: <$5 green, <$20 yellow, >=$20 red
		{"daily_green", 3.50, 5.00, 20.00, "\033[32m"},
		{"daily_yellow", 12.00, 5.00, 20.00, "\033[33m"},
		{"daily_red", 25.00, 5.00, 20.00, "\033[31m"},
		// Burn thresholds: <$1 green, <$5 yellow, >=$5 red
		{"burn_green", 0.50, 1.00, 5.00, "\033[32m"},
		{"burn_yellow", 3.00, 1.00, 5.00, "\033[33m"},
		{"burn_red", 12.00, 1.00, 5.00, "\033[31m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := costColor(tt.cost, tt.warn, tt.crit)
			if got != tt.wantColor {
				t.Errorf("costColor(%f, %f, %f) = %q, want %q", tt.cost, tt.warn, tt.crit, got, tt.wantColor)
			}
		})
	}
}

func TestRenderSectionsWithBurnRate(t *testing.T) {
	r := &mockRenderer{}
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// 90K tokens over 5 minutes (300s) → 18K TPM, $0.50/300s = $6/h
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: types.CurrentUsage{
				InputTokens:              80000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     5000,
			},
		},
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 300000},
	}

	sections := RenderSections(input, cfg, plat, store, r, types.CostTierSonnet)

	// Session
	if !strings.Contains(sections.Session, "💰") {
		t.Error("Session should contain 💰")
	}
	if !strings.Contains(sections.Session, "$0.50") {
		t.Errorf("Session should contain $0.50, got: %s", sections.Session)
	}

	// Daily
	if !strings.Contains(sections.Daily, "📅") {
		t.Error("Daily should contain 📅")
	}

	// Burn
	if !strings.Contains(sections.Burn, "🔥") {
		t.Error("Burn should contain 🔥")
	}
	if !strings.Contains(sections.Burn, "t/m") {
		t.Errorf("Burn should contain 't/m' (has TPM), got: %s", sections.Burn)
	}
	if !strings.Contains(sections.Burn, "/h") {
		t.Errorf("Burn should contain '/h' (cost per hour), got: %s", sections.Burn)
	}
}

func TestRenderSectionsNoBurn(t *testing.T) {
	r := &mockRenderer{}
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// Short duration (45s < 60s threshold) → no burn rate
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage:      types.CurrentUsage{InputTokens: 5000},
		},
		Cost: types.Cost{TotalCostUSD: 0.02, TotalDurationMS: 45000},
	}

	sections := RenderSections(input, cfg, plat, store, r, types.CostTierSonnet)

	if !strings.Contains(sections.Burn, "🔥") {
		t.Error("Burn should contain 🔥 even with no burn data")
	}
	if !strings.Contains(sections.Burn, "--") {
		t.Errorf("Burn should show '--' when no burn data, got: %s", sections.Burn)
	}
	if strings.Contains(sections.Burn, "t/m") {
		t.Errorf("Burn should not contain 't/m' with short duration, got: %s", sections.Burn)
	}
}

func TestRenderSectionsZeroDuration(t *testing.T) {
	r := &mockRenderer{}
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// Zero duration → no burn, no cost per hour
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
		},
		Cost: types.Cost{TotalCostUSD: 0, TotalDurationMS: 0},
	}

	sections := RenderSections(input, cfg, plat, store, r, types.CostTierSonnet)

	if !strings.Contains(sections.Session, "$0.00") {
		t.Errorf("Session should be $0.00, got: %s", sections.Session)
	}
	if !strings.Contains(sections.Burn, "--") {
		t.Errorf("Burn should be '--' with zero duration, got: %s", sections.Burn)
	}
}

func TestRenderSectionsLegacyTokenBurn(t *testing.T) {
	r := &mockRenderer{}
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// Legacy tokens (no current_usage), 75K tokens over 5 minutes
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			TotalInputTokens:  60000,
			TotalOutputTokens: 15000,
		},
		Cost: types.Cost{TotalCostUSD: 0.75, TotalDurationMS: 300000},
	}

	sections := RenderSections(input, cfg, plat, store, r, types.CostTierSonnet)

	// Should compute burn from legacy tokens
	if !strings.Contains(sections.Burn, "t/m") {
		t.Errorf("Burn should have TPM from legacy tokens, got: %s", sections.Burn)
	}
}

func TestCostPerHourZeroWhenShort(t *testing.T) {
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// Duration <= 60s → no cost per hour
	input := types.Input{
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 60000},
	}

	display := Track(input, cfg, plat, store)
	if display.CostPerHour != 0 {
		t.Errorf("CostPerHour should be 0 when duration <= 60s, got %f", display.CostPerHour)
	}
}

func TestMultipleSessionsDailyAccumulation(t *testing.T) {
	store := newMockCache()
	cfg := types.DefaultConfig()

	// Session A (stable)
	platA := &mockPlatform{sessionID: "sessionA", stable: true}
	inputA := types.Input{
		Cost: types.Cost{TotalCostUSD: 1.00, TotalDurationMS: 120000},
	}
	Track(inputA, cfg, platA, store)

	// Session B (stable, different ID)
	platB := &mockPlatform{sessionID: "sessionB", stable: true}
	inputB := types.Input{
		Cost: types.Cost{TotalCostUSD: 2.00, TotalDurationMS: 120000},
	}
	displayB := Track(inputB, cfg, platB, store)

	// Daily should be sum of both sessions
	if displayB.DailyCost < 2.99 || displayB.DailyCost > 3.01 {
		t.Errorf("DailyCost should be ≈3.00 (1+2), got %f", displayB.DailyCost)
	}
}

func TestRenderSectionsNormalizedOpus(t *testing.T) {
	r := &mockRenderer{}
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// 90K tokens over 5 minutes → 18K TPM raw, Opus 5x → 90K normalized
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: types.CurrentUsage{
				InputTokens:              80000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     5000,
			},
		},
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 300000},
	}

	sections := RenderSections(input, cfg, plat, store, r, types.CostTierOpus)

	// Should have approximate prefix for non-baseline model
	if !strings.Contains(sections.Burn, "≈") {
		t.Errorf("Opus burn should contain ≈ prefix, got: %s", sections.Burn)
	}
}

func TestRenderSectionsNormalizedSonnet(t *testing.T) {
	r := &mockRenderer{}
	store := newMockCache()
	plat := &mockPlatform{sessionID: "s1", stable: true}
	cfg := types.DefaultConfig()

	// Sonnet (baseline) should NOT have ≈ prefix
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: types.CurrentUsage{
				InputTokens:              80000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     5000,
			},
		},
		Cost: types.Cost{TotalCostUSD: 0.50, TotalDurationMS: 300000},
	}

	sections := RenderSections(input, cfg, plat, store, r, types.CostTierSonnet)

	if strings.Contains(sections.Burn, "≈") {
		t.Errorf("Sonnet burn should NOT contain ≈ prefix, got: %s", sections.Burn)
	}
}

func TestResolveSession(t *testing.T) {
	plat := &mockPlatform{sessionID: "test-session", stable: true}
	id, stable := ResolveSession(plat)
	if id != "test-session" {
		t.Errorf("id = %q, want %q", id, "test-session")
	}
	if !stable {
		t.Error("stable = false, want true")
	}

	plat2 := &mockPlatform{sessionID: "unstable-1", stable: false}
	id2, stable2 := ResolveSession(plat2)
	if id2 != "unstable-1" {
		t.Errorf("id = %q, want %q", id2, "unstable-1")
	}
	if stable2 {
		t.Error("stable = true, want false")
	}
}
