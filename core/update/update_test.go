package update

import (
	"testing"
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   bool
	}{
		{"1.0.0", "0.9.0", true},
		{"1.0.0", "1.0.0", false},
		{"0.9.0", "1.0.0", false},
		{"1.2.3", "1.2.2", true},
		{"1.2.3", "1.2.4", false},
		{"2.0.0", "1.9.9", true},
		{"1.10.0", "1.9.0", true},
		{"v1.2.0", "v1.1.0", true},
		{"1.0", "1.0.0", false},
		{"1.1", "1.0.0", true},
		{"1.0.0.1", "1.0.0", true},
	}

	for _, tt := range tests {
		got := GreaterThan(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("GreaterThan(%q, %q) = %v, want %v", tt.v1, tt.v2, got, tt.want)
		}
	}
}

// mockCache for update tests.
type mockCache struct {
	files map[string][]byte
	fresh map[string]bool
}

func newMockCache() *mockCache {
	return &mockCache{
		files: make(map[string][]byte),
		fresh: make(map[string]bool),
	}
}

func (m *mockCache) AtomicWrite(path string, data []byte) error {
	m.files[path] = data
	return nil
}

func (m *mockCache) ReadIfFresh(path string, ttl time.Duration) ([]byte, bool) {
	data, ok := m.files[path]
	if !ok {
		return nil, false
	}
	return data, m.fresh[path]
}

func (m *mockCache) ReadFile(path string) ([]byte, error) {
	return nil, nil
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

type mockAPI struct{}

func (m *mockAPI) FetchRateLimits(token string) (*types.RateLimitResponse, error) {
	return nil, nil
}

func (m *mockAPI) FetchLatestRelease(repo string) (string, error) {
	return "v2.0.0", nil
}

func TestCheckWithFreshCache(t *testing.T) {
	store := newMockCache()
	cachePath := "/tmp/" + cacheFile
	store.files[cachePath] = []byte("2.0.0")
	store.fresh[cachePath] = true

	api := &mockAPI{}
	hasUpdate, latest := Check("1.0.0", "/tmp", store, api)

	if !hasUpdate {
		t.Error("should have update")
	}
	if latest != "2.0.0" {
		t.Errorf("latest = %q, want %q", latest, "2.0.0")
	}
}

func TestCheckNoUpdate(t *testing.T) {
	store := newMockCache()
	cachePath := "/tmp/" + cacheFile
	store.files[cachePath] = []byte("1.0.0")
	store.fresh[cachePath] = true

	api := &mockAPI{}
	hasUpdate, _ := Check("1.0.0", "/tmp", store, api)

	if hasUpdate {
		t.Error("should not have update when versions are equal")
	}
}

func TestCheckDevVersion(t *testing.T) {
	store := newMockCache()
	api := &mockAPI{}

	hasUpdate, _ := Check("dev", "/tmp", store, api)
	if hasUpdate {
		t.Error("dev version should never show update")
	}

	hasUpdate2, _ := Check("", "/tmp", store, api)
	if hasUpdate2 {
		t.Error("empty version should never show update")
	}
}

// mockRenderer for Render tests.
type mockRenderer struct{}

func (m *mockRenderer) Colorize(text string, percent int) string { return text }
func (m *mockRenderer) Color(text, color string) string          { return text }
func (m *mockRenderer) Dim(text string) string                   { return text }
func (m *mockRenderer) MakeBar(percent, width int) string        { return "[bar]" }
func (m *mockRenderer) FormatTokens(n int) string                { return "" }
func (m *mockRenderer) FormatTokensF(n int) string               { return "" }
func (m *mockRenderer) FormatCost(f float64) string              { return "" }

func TestRenderWithUpdate(t *testing.T) {
	store := newMockCache()
	cachePath := "/tmp/" + cacheFile
	store.files[cachePath] = []byte("2.0.0")
	store.fresh[cachePath] = true

	api := &mockAPI{}
	r := &mockRenderer{}

	result := Render("1.0.0", "/tmp", store, api, r)
	if result == "" {
		t.Error("Render should return update notice when update available")
	}
	if result != " [Update]" {
		t.Errorf("Render = %q, want %q", result, " [Update]")
	}
}

func TestRenderNoUpdate(t *testing.T) {
	store := newMockCache()
	cachePath := "/tmp/" + cacheFile
	store.files[cachePath] = []byte("1.0.0")
	store.fresh[cachePath] = true

	api := &mockAPI{}
	r := &mockRenderer{}

	result := Render("1.0.0", "/tmp", store, api, r)
	if result != "" {
		t.Errorf("Render should return empty when no update, got: %q", result)
	}
}

func TestRenderDevVersion(t *testing.T) {
	store := newMockCache()
	api := &mockAPI{}
	r := &mockRenderer{}

	result := Render("dev", "/tmp", store, api, r)
	if result != "" {
		t.Errorf("Render should return empty for dev version, got: %q", result)
	}
}
