package ratelimit

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

// mockCacheStore implements ports.CacheStore for testing.
type mockCacheStore struct {
	files map[string][]byte
	fresh map[string]bool
}

func newMockCache() *mockCacheStore {
	return &mockCacheStore{
		files: make(map[string][]byte),
		fresh: make(map[string]bool),
	}
}

func (m *mockCacheStore) AtomicWrite(path string, data []byte) error {
	m.files[path] = data
	return nil
}

func (m *mockCacheStore) ReadIfFresh(path string, ttl time.Duration) ([]byte, bool) {
	data, ok := m.files[path]
	if !ok {
		return nil, false
	}
	return data, m.fresh[path]
}

func (m *mockCacheStore) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, &notFoundErr{}
	}
	return data, nil
}

func (m *mockCacheStore) WriteFile(path string, data []byte) error {
	m.files[path] = data
	return nil
}

func (m *mockCacheStore) FileMTime(path string) (time.Time, error) {
	return time.Now(), nil
}

func (m *mockCacheStore) CleanOld(dir, pattern, keep string) error {
	return nil
}

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

// mockAPIClient implements ports.APIClient for testing.
type mockAPIClient struct {
	resp *types.RateLimitResponse
	err  error
}

func (m *mockAPIClient) FetchRateLimits(token string) (*types.RateLimitResponse, error) {
	return m.resp, m.err
}

func (m *mockAPIClient) FetchLatestRelease(repo string) (string, error) {
	return "v1.0.0", nil
}

func TestLoadFromCache(t *testing.T) {
	store := newMockCache()

	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{Utilization: 50, ResetsAt: "2025-02-06T19:00:00Z"},
		SevenDay: types.RateLimitWindow{Utilization: 30, ResetsAt: "2025-02-10T00:00:00Z"},
	}
	raw, _ := json.Marshal(resp)
	cachePath := "/tmp/" + cacheName
	store.files[cachePath] = raw
	store.fresh[cachePath] = true

	creds := types.Credentials{OAuthToken: "token"}
	cfg := types.DefaultConfig()

	data, err := Load(creds, cfg, store, &mockAPIClient{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if data.FiveHourPercent != 50 {
		t.Errorf("FiveHourPercent = %f, want 50", data.FiveHourPercent)
	}
	if data.SevenDayPercent != 30 {
		t.Errorf("SevenDayPercent = %f, want 30", data.SevenDayPercent)
	}
}

// mockPlatform implements ports.PlatformInfo for ratelimit tests.
type mockPlatform struct{}

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
	return "test-session", true
}

// mockRenderer implements ports.Renderer for ratelimit render tests.
type mockRenderer struct{}

func (m *mockRenderer) Colorize(text string, percent int) string { return text }
func (m *mockRenderer) Color(text, color string) string          { return text }
func (m *mockRenderer) Dim(text string) string                   { return text }
func (m *mockRenderer) MakeBar(percent, width int) string        { return "[bar]" }
func (m *mockRenderer) FormatTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.0fK", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}
func (m *mockRenderer) FormatTokensF(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}
func (m *mockRenderer) FormatCost(f float64) string { return fmt.Sprintf("$%.2f", f) }

func TestLoadFromAPI(t *testing.T) {
	store := newMockCache()

	resp := &types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{Utilization: 72, ResetsAt: "2025-02-06T19:00:00Z"},
		SevenDay: types.RateLimitWindow{Utilization: 45, ResetsAt: "2025-02-10T00:00:00Z"},
	}

	api := &mockAPIClient{resp: resp}
	creds := types.Credentials{OAuthToken: "token"}
	cfg := types.DefaultConfig()

	data, err := Load(creds, cfg, store, api)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if data.FiveHourPercent != 72 {
		t.Errorf("FiveHourPercent = %f, want 72", data.FiveHourPercent)
	}

	// Should have cached the response
	cachePath := "/tmp/" + cacheName
	if _, ok := store.files[cachePath]; !ok {
		t.Error("should have cached the API response")
	}
}

func TestLoadNoOAuth(t *testing.T) {
	store := newMockCache()
	api := &mockAPIClient{}
	creds := types.Credentials{} // No OAuth

	_, err := Load(creds, types.DefaultConfig(), store, api)
	if err == nil {
		t.Error("Load with no OAuth should return error")
	}
}

func TestLoadStaleCacheFallback(t *testing.T) {
	store := newMockCache()

	// Put stale cache (not fresh)
	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{Utilization: 40, ResetsAt: "2025-02-06T19:00:00Z"},
		SevenDay: types.RateLimitWindow{Utilization: 20, ResetsAt: "2025-02-10T00:00:00Z"},
	}
	raw, _ := json.Marshal(resp)
	cachePath := "/tmp/" + cacheName
	store.files[cachePath] = raw
	store.fresh[cachePath] = false // stale

	// API fails
	api := &mockAPIClient{err: fmt.Errorf("network error")}
	creds := types.Credentials{OAuthToken: "token"}

	data, err := Load(creds, types.DefaultConfig(), store, api)
	if err != nil {
		t.Fatalf("should fallback to stale cache, got error: %v", err)
	}
	if data.FiveHourPercent != 40 {
		t.Errorf("FiveHourPercent = %f, want 40 (from stale cache)", data.FiveHourPercent)
	}
	if !data.FromCache {
		t.Error("FromCache should be true for stale cache fallback")
	}
}

func TestRenderSectionsSuccess(t *testing.T) {
	store := newMockCache()
	r := &mockRenderer{}

	// Pre-populate cache with rate data
	now := time.Now()
	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{
			Utilization: 50,
			ResetsAt:    now.Add(3 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: types.RateLimitWindow{
			Utilization: 25,
			ResetsAt:    now.Add(5 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
	raw, _ := json.Marshal(resp)
	cachePath := "/tmp/" + cacheName
	store.files[cachePath] = raw
	store.fresh[cachePath] = true

	creds := types.Credentials{OAuthToken: "token"}
	cfg := types.DefaultConfig()
	plat := &mockPlatform{}

	// Input with tokens for burn rate
	input := types.Input{
		ContextWindow: types.ContextWindow{
			CurrentUsage: types.CurrentUsage{
				InputTokens:              80000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     5000,
			},
		},
		Cost: types.Cost{TotalDurationMS: 300000}, // 5 min
	}

	sections := RenderSections(input, creds, cfg, plat, store, &mockAPIClient{}, r, types.CostTierSonnet)

	// FiveHour section
	if !strings.Contains(sections.FiveHour, "5h:") {
		t.Errorf("FiveHour should contain '5h:', got: %s", sections.FiveHour)
	}
	if !strings.Contains(sections.FiveHour, "50%") {
		t.Errorf("FiveHour should contain '50%%', got: %s", sections.FiveHour)
	}
	if !strings.Contains(sections.FiveHour, "[bar]") {
		t.Errorf("FiveHour should contain bar, got: %s", sections.FiveHour)
	}

	// Burn section
	if !strings.Contains(sections.Burn, "🔥") {
		t.Errorf("Burn should contain 🔥, got: %s", sections.Burn)
	}
	if !strings.Contains(sections.Burn, "t/m") {
		t.Errorf("Burn should contain 't/m' with tokens, got: %s", sections.Burn)
	}

	// SevenDay section
	if !strings.Contains(sections.SevenDay, "7d:") {
		t.Errorf("SevenDay should contain '7d:', got: %s", sections.SevenDay)
	}
	if !strings.Contains(sections.SevenDay, "25%") {
		t.Errorf("SevenDay should contain '25%%', got: %s", sections.SevenDay)
	}
}

func TestRenderSectionsError(t *testing.T) {
	store := newMockCache()
	r := &mockRenderer{}

	// No cache, no OAuth → error path
	creds := types.Credentials{}
	cfg := types.DefaultConfig()
	plat := &mockPlatform{}
	input := types.Input{}

	sections := RenderSections(input, creds, cfg, plat, store, &mockAPIClient{}, r, types.CostTierSonnet)

	if !strings.Contains(sections.FiveHour, "5h:") || !strings.Contains(sections.FiveHour, "--") {
		t.Errorf("FiveHour on error should be '5h: --', got: %s", sections.FiveHour)
	}
	if !strings.Contains(sections.Burn, "🔥") || !strings.Contains(sections.Burn, "--") {
		t.Errorf("Burn on error should be '🔥 --', got: %s", sections.Burn)
	}
	if !strings.Contains(sections.SevenDay, "7d:") || !strings.Contains(sections.SevenDay, "--") {
		t.Errorf("SevenDay on error should be '7d: --', got: %s", sections.SevenDay)
	}
}

func TestRenderSectionsNoBurn(t *testing.T) {
	store := newMockCache()
	r := &mockRenderer{}

	now := time.Now()
	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{
			Utilization: 30,
			ResetsAt:    now.Add(4 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: types.RateLimitWindow{
			Utilization: 15,
			ResetsAt:    now.Add(6 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
	raw, _ := json.Marshal(resp)
	store.files["/tmp/"+cacheName] = raw
	store.fresh["/tmp/"+cacheName] = true

	creds := types.Credentials{OAuthToken: "token"}
	cfg := types.DefaultConfig()
	plat := &mockPlatform{}

	// Short duration (< 60s) → no burn rate
	input := types.Input{
		ContextWindow: types.ContextWindow{
			CurrentUsage: types.CurrentUsage{InputTokens: 5000},
		},
		Cost: types.Cost{TotalDurationMS: 45000},
	}

	sections := RenderSections(input, creds, cfg, plat, store, &mockAPIClient{}, r, types.CostTierSonnet)

	if !strings.Contains(sections.Burn, "--") {
		t.Errorf("Burn should show '--' with short duration, got: %s", sections.Burn)
	}
	if strings.Contains(sections.Burn, "t/m") {
		t.Errorf("Burn should not show 't/m' with short duration, got: %s", sections.Burn)
	}
}

func TestRenderLegacyFallback(t *testing.T) {
	store := newMockCache()
	r := &mockRenderer{}

	// Pre-populate cache
	now := time.Now()
	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{
			Utilization: 33,
			ResetsAt:    now.Add(3 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: types.RateLimitWindow{
			Utilization: 20,
			ResetsAt:    now.Add(5 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
	raw, _ := json.Marshal(resp)
	store.files["/tmp/"+cacheName] = raw
	store.fresh["/tmp/"+cacheName] = true

	creds := types.Credentials{OAuthToken: "token"}
	cfg := types.DefaultConfig()
	plat := &mockPlatform{}

	result := Render(creds, cfg, plat, store, &mockAPIClient{}, r)
	if !strings.Contains(result, "5h:") {
		t.Errorf("Render should contain '5h:', got: %s", result)
	}
	if !strings.Contains(result, "33%") {
		t.Errorf("Render should contain '33%%', got: %s", result)
	}
}

func TestRenderLegacyNoCredentials(t *testing.T) {
	store := newMockCache()
	r := &mockRenderer{}
	creds := types.Credentials{}
	cfg := types.DefaultConfig()
	plat := &mockPlatform{}

	result := Render(creds, cfg, plat, store, &mockAPIClient{}, r)
	if result != "" {
		t.Errorf("Render with no credentials should return empty, got: %s", result)
	}
}

func TestPaceColorize(t *testing.T) {
	r := &mockRenderer{}

	tests := []struct {
		pace float64
		want string
	}{
		{0.5, "0.5x"},  // green (<1.0)
		{0.9, "0.9x"},  // green (<1.0)
		{1.0, "1.0x"},  // yellow (>=1.0, <1.5)
		{1.3, "1.3x"},  // yellow
		{1.5, "1.5x"},  // red (>=1.5)
		{2.5, "2.5x"},  // red
	}

	for _, tt := range tests {
		got := paceColorize(tt.pace, r)
		if got != tt.want {
			t.Errorf("paceColorize(%f) = %q, want %q", tt.pace, got, tt.want)
		}
	}
}

func TestRenderFiveHourWithPace(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{
		FiveHourPercent: 50.0,
	}
	pace := types.PaceInfo{
		FiveHourPace: 1.2,
		HittingLimit: false,
	}

	result := renderFiveHour(data, pace, r, false)
	if !strings.Contains(result, "50%") {
		t.Errorf("should contain '50%%', got: %s", result)
	}
	if !strings.Contains(result, "1.2x") {
		t.Errorf("should contain pace '1.2x', got: %s", result)
	}
}

func TestRenderFiveHourHittingLimit(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{
		FiveHourPercent: 90.0,
	}
	pace := types.PaceInfo{
		FiveHourPace: 3.0,
		HittingLimit: true,
	}

	result := renderFiveHour(data, pace, r, false)
	if !strings.Contains(result, "⚠️") {
		t.Errorf("should contain ⚠️ when hitting limit, got: %s", result)
	}
}

func TestRenderBurnWithTPM(t *testing.T) {
	r := &mockRenderer{}

	burn := types.BurnInfo{LocalTPM: 15000}
	result := renderBurn(burn, r, types.CostTierSonnet, types.DefaultConfig())
	if !strings.Contains(result, "15.0K") {
		t.Errorf("should format 15000 TPM as '15.0K', got: %s", result)
	}
	if !strings.Contains(result, "t/m") {
		t.Errorf("should contain 't/m', got: %s", result)
	}
}

func TestRenderBurnNone(t *testing.T) {
	r := &mockRenderer{}

	burn := types.BurnInfo{LocalTPM: 0}
	result := renderBurn(burn, r, types.CostTierSonnet, types.DefaultConfig())
	if !strings.Contains(result, "--") {
		t.Errorf("should show '--' with no TPM, got: %s", result)
	}
}

func TestRenderBurnHighActivity(t *testing.T) {
	r := &mockRenderer{}

	burn := types.BurnInfo{LocalTPM: 5000, IsHighActivity: true}
	result := renderBurn(burn, r, types.CostTierSonnet, types.DefaultConfig())
	if !strings.Contains(result, "⚡") {
		t.Errorf("should contain ⚡ for high activity, got: %s", result)
	}
}

func TestRenderBurnHighActivityNoLocal(t *testing.T) {
	r := &mockRenderer{}

	burn := types.BurnInfo{LocalTPM: 0, IsHighActivity: true}
	result := renderBurn(burn, r, types.CostTierSonnet, types.DefaultConfig())
	if !strings.Contains(result, "⚡") {
		t.Errorf("should contain ⚡ for high activity even without local TPM, got: %s", result)
	}
	if !strings.Contains(result, "--") {
		t.Errorf("should show '--' without local TPM, got: %s", result)
	}
}

func TestRenderSevenDayWithPace(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{
		SevenDayPercent: 25.0,
	}
	pace := types.PaceInfo{
		SevenDayPace: 0.8,
	}

	result := renderSevenDay(data, pace, r, false)
	if !strings.Contains(result, "25%") {
		t.Errorf("should contain '25%%', got: %s", result)
	}
	if !strings.Contains(result, "0.8x") {
		t.Errorf("should contain pace '0.8x', got: %s", result)
	}
}

func TestRenderSevenDayOverPace(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{
		SevenDayPercent: 60.0,
	}
	pace := types.PaceInfo{
		SevenDayPace: 1.5,
	}

	result := renderSevenDay(data, pace, r, false)
	if !strings.Contains(result, "⚠️") {
		t.Errorf("should contain ⚠️ when 7d pace > 1.0, got: %s", result)
	}
}

func TestRenderSevenDayWithResetDays(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{
		SevenDayPercent: 30.0,
	}
	pace := types.PaceInfo{
		SevenDayPace:     0.5,
		SevenDayResetFmt: "→2d",
	}

	result := renderSevenDay(data, pace, r, false)
	if !strings.Contains(result, "→2d") {
		t.Errorf("should contain '→2d', got: %s", result)
	}
}

func TestRenderBurnNormalizedOpus(t *testing.T) {
	r := &mockRenderer{}
	cfg := types.DefaultConfig()

	// 15K TPM raw, Opus 5x → 75K normalized
	burn := types.BurnInfo{LocalTPM: 15000}
	result := renderBurn(burn, r, types.CostTierOpus, cfg)

	// Should have ≈ prefix
	if !strings.Contains(result, "≈") {
		t.Errorf("Opus burn should contain ≈ prefix, got: %s", result)
	}
	// Should show normalized value (75K)
	if !strings.Contains(result, "75.0K") {
		t.Errorf("Opus burn should show 75.0K (15K*5), got: %s", result)
	}
}

func TestRenderBurnNormalizedSonnet(t *testing.T) {
	r := &mockRenderer{}
	cfg := types.DefaultConfig()

	// Sonnet (baseline) should NOT have ≈ prefix
	burn := types.BurnInfo{LocalTPM: 15000}
	result := renderBurn(burn, r, types.CostTierSonnet, cfg)

	if strings.Contains(result, "≈") {
		t.Errorf("Sonnet burn should NOT contain ≈ prefix, got: %s", result)
	}
	if !strings.Contains(result, "15.0K") {
		t.Errorf("Sonnet burn should show raw 15.0K, got: %s", result)
	}
}

func TestRenderFiveHourNormalized(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{FiveHourPercent: 50.0}
	pace := types.PaceInfo{FiveHourPace: 1.2}

	// Normalized (Opus) → should have ≈ prefix on pace
	result := renderFiveHour(data, pace, r, true)
	if !strings.Contains(result, "≈1.2x") {
		t.Errorf("Normalized 5h pace should contain ≈1.2x, got: %s", result)
	}

	// Not normalized (Sonnet) → no prefix
	result2 := renderFiveHour(data, pace, r, false)
	if strings.Contains(result2, "≈") {
		t.Errorf("Non-normalized 5h pace should NOT contain ≈, got: %s", result2)
	}
}

func TestRenderSevenDayNormalized(t *testing.T) {
	r := &mockRenderer{}

	data := types.RateLimitData{SevenDayPercent: 25.0}
	pace := types.PaceInfo{SevenDayPace: 0.8}

	// Normalized → should have ≈ prefix
	result := renderSevenDay(data, pace, r, true)
	if !strings.Contains(result, "≈0.8x") {
		t.Errorf("Normalized 7d pace should contain ≈0.8x, got: %s", result)
	}

	// Not normalized → no prefix
	result2 := renderSevenDay(data, pace, r, false)
	if strings.Contains(result2, "≈") {
		t.Errorf("Non-normalized 7d pace should NOT contain ≈, got: %s", result2)
	}
}

func TestRenderSectionsNormalizedOpus(t *testing.T) {
	store := newMockCache()
	r := &mockRenderer{}

	now := time.Now()
	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{
			Utilization: 50,
			ResetsAt:    now.Add(3 * time.Hour).Format(time.RFC3339),
		},
		SevenDay: types.RateLimitWindow{
			Utilization: 25,
			ResetsAt:    now.Add(5 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
	raw, _ := json.Marshal(resp)
	store.files["/tmp/"+cacheName] = raw
	store.fresh["/tmp/"+cacheName] = true

	creds := types.Credentials{OAuthToken: "token"}
	cfg := types.DefaultConfig()
	plat := &mockPlatform{}

	input := types.Input{
		ContextWindow: types.ContextWindow{
			CurrentUsage: types.CurrentUsage{
				InputTokens:              80000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     5000,
			},
		},
		Cost: types.Cost{TotalDurationMS: 300000},
	}

	sections := RenderSections(input, creds, cfg, plat, store, &mockAPIClient{}, r, types.CostTierOpus)

	// Burn should have ≈ prefix for Opus
	if !strings.Contains(sections.Burn, "≈") {
		t.Errorf("Opus RenderSections burn should contain ≈, got: %s", sections.Burn)
	}
}
