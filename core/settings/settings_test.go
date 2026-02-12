package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- mock CacheStore (follows core/update/update_test.go pattern) ---

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
	data, ok := m.files[path]
	return data, ok
}

func (m *mockCache) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", path)
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

// --- DeepMerge tests ---

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name string
		dst  map[string]any
		src  map[string]any
		want map[string]any
	}{
		{
			name: "empty dst",
			dst:  map[string]any{},
			src:  map[string]any{"a": float64(1)},
			want: map[string]any{"a": float64(1)},
		},
		{
			name: "empty src",
			dst:  map[string]any{"a": float64(1)},
			src:  map[string]any{},
			want: map[string]any{"a": float64(1)},
		},
		{
			name: "disjoint keys",
			dst:  map[string]any{"a": float64(1)},
			src:  map[string]any{"b": float64(2)},
			want: map[string]any{"a": float64(1), "b": float64(2)},
		},
		{
			name: "overwrite scalar",
			dst:  map[string]any{"a": float64(1)},
			src:  map[string]any{"a": float64(2)},
			want: map[string]any{"a": float64(2)},
		},
		{
			name: "deep merge nested maps",
			dst:  map[string]any{"a": map[string]any{"x": float64(1), "y": float64(2)}},
			src:  map[string]any{"a": map[string]any{"y": float64(3), "z": float64(4)}},
			want: map[string]any{"a": map[string]any{"x": float64(1), "y": float64(3), "z": float64(4)}},
		},
		{
			name: "src map replaces non-map",
			dst:  map[string]any{"a": "string"},
			src:  map[string]any{"a": map[string]any{"x": float64(1)}},
			want: map[string]any{"a": map[string]any{"x": float64(1)}},
		},
		{
			name: "preserves unrelated keys",
			dst: map[string]any{
				"keep":       true,
				"statusLine": map[string]any{"old": true},
			},
			src: map[string]any{
				"statusLine": map[string]any{"new": true},
			},
			want: map[string]any{
				"keep":       true,
				"statusLine": map[string]any{"old": true, "new": true},
			},
		},
		{
			name: "does not mutate dst",
			dst:  map[string]any{"a": map[string]any{"x": float64(1)}},
			src:  map[string]any{"a": map[string]any{"y": float64(2)}},
			want: map[string]any{"a": map[string]any{"x": float64(1), "y": float64(2)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Snapshot dst to verify it is not mutated
			dstJSON, _ := json.Marshal(tt.dst)

			got := DeepMerge(tt.dst, tt.src)

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("DeepMerge() = %s, want %s", gotJSON, wantJSON)
			}

			// Verify dst was not mutated
			afterJSON, _ := json.Marshal(tt.dst)
			if string(dstJSON) != string(afterJSON) {
				t.Errorf("DeepMerge mutated dst: was %s, now %s", dstJSON, afterJSON)
			}
		})
	}
}

// --- NeedsUpdate tests ---

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current map[string]any
		desired map[string]any
		want    bool
	}{
		{
			name: "exact match",
			current: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/path"},
			},
			desired: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/path"},
			},
			want: false,
		},
		{
			name:    "missing key",
			current: map[string]any{},
			desired: map[string]any{
				"statusLine": map[string]any{"type": "command"},
			},
			want: true,
		},
		{
			name: "wrong value",
			current: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/old"},
			},
			desired: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/new"},
			},
			want: true,
		},
		{
			name: "extra keys in current are ok",
			current: map[string]any{
				"statusLine":     map[string]any{"type": "command", "command": "/path"},
				"enabledPlugins": map[string]any{"foo": true},
			},
			desired: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/path"},
			},
			want: false,
		},
		{
			name: "extra nested keys in current are ok",
			current: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/path", "extra": true},
			},
			desired: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/path"},
			},
			want: false,
		},
		{
			name: "missing nested key",
			current: map[string]any{
				"statusLine": map[string]any{"type": "command"},
			},
			desired: map[string]any{
				"statusLine": map[string]any{"type": "command", "command": "/path"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsUpdate(tt.current, tt.desired)
			if got != tt.want {
				t.Errorf("NeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Setup tests ---

func TestSetup(t *testing.T) {
	t.Run("creates file when missing", func(t *testing.T) {
		store := newMockCache()
		result, err := Setup("/tmp/settings.json", "/usr/bin/statusline", store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Action != "created" {
			t.Errorf("action = %q, want %q", result.Action, "created")
		}
		// Verify file was written
		data, ok := store.files["/tmp/settings.json"]
		if !ok {
			t.Fatal("settings file was not written")
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("invalid JSON written: %v", err)
		}
		sl, ok := parsed["statusLine"].(map[string]any)
		if !ok {
			t.Fatal("statusLine key missing")
		}
		if sl["command"] != "/usr/bin/statusline" {
			t.Errorf("command = %v, want /usr/bin/statusline", sl["command"])
		}
	})

	t.Run("noop when settings correct", func(t *testing.T) {
		store := newMockCache()
		store.files["/tmp/settings.json"] = []byte(`{
  "statusLine": {
    "type": "command",
    "command": "/usr/bin/statusline"
  }
}`)
		result, err := Setup("/tmp/settings.json", "/usr/bin/statusline", store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Action != "noop" {
			t.Errorf("action = %q, want %q", result.Action, "noop")
		}
	})

	t.Run("updates when binary path changed", func(t *testing.T) {
		store := newMockCache()
		store.files["/tmp/settings.json"] = []byte(`{
  "statusLine": {
    "type": "command",
    "command": "/old/path/statusline"
  }
}`)
		result, err := Setup("/tmp/settings.json", "/new/path/statusline", store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Action != "updated" {
			t.Errorf("action = %q, want %q", result.Action, "updated")
		}
		var parsed map[string]any
		json.Unmarshal(store.files["/tmp/settings.json"], &parsed)
		sl := parsed["statusLine"].(map[string]any)
		if sl["command"] != "/new/path/statusline" {
			t.Errorf("command = %v, want /new/path/statusline", sl["command"])
		}
	})

	t.Run("preserves other keys", func(t *testing.T) {
		store := newMockCache()
		store.files["/tmp/settings.json"] = []byte(`{
  "enabledPlugins": {
    "statusline@marketplace": true
  },
  "includeCoAuthoredBy": false
}`)
		result, err := Setup("/tmp/settings.json", "/usr/bin/statusline", store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Action != "updated" {
			t.Errorf("action = %q, want %q", result.Action, "updated")
		}
		var parsed map[string]any
		json.Unmarshal(store.files["/tmp/settings.json"], &parsed)

		// statusLine was added
		if _, ok := parsed["statusLine"]; !ok {
			t.Error("statusLine not added")
		}
		// Other keys preserved
		if _, ok := parsed["enabledPlugins"]; !ok {
			t.Error("enabledPlugins was removed")
		}
		if _, ok := parsed["includeCoAuthoredBy"]; !ok {
			t.Error("includeCoAuthoredBy was removed")
		}
	})
}

// --- FindSettingsFile tests ---

func TestFindSettingsFile(t *testing.T) {
	pluginID := "statusline@test-marketplace"

	t.Run("finds project scope", func(t *testing.T) {
		dir := t.TempDir()
		projectSettings := filepath.Join(dir, ".claude", "settings.json")
		os.MkdirAll(filepath.Dir(projectSettings), 0o755)
		os.WriteFile(projectSettings, []byte(fmt.Sprintf(`{"enabledPlugins":{"%s":true}}`, pluginID)), 0o644)

		path, exists := FindSettingsFile(dir, pluginID)
		if !exists {
			t.Error("expected exists=true")
		}
		if path != projectSettings {
			t.Errorf("path = %q, want %q", path, projectSettings)
		}
	})

	t.Run("falls back to user scope when project has no match", func(t *testing.T) {
		dir := t.TempDir()
		// Create project settings without the plugin ID
		projectSettings := filepath.Join(dir, ".claude", "settings.json")
		os.MkdirAll(filepath.Dir(projectSettings), 0o755)
		os.WriteFile(projectSettings, []byte(`{"other": true}`), 0o644)

		path, _ := FindSettingsFile(dir, pluginID)
		// Should fall back to user home (won't match temp dir)
		if path == projectSettings {
			t.Error("should not have returned project settings without matching plugin ID")
		}
	})

	t.Run("returns user scope fallback path", func(t *testing.T) {
		dir := t.TempDir()
		path, _ := FindSettingsFile(filepath.Join(dir, "nonexistent"), pluginID)
		// Should return user home fallback regardless of whether it exists
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".claude", "settings.json")
		if path != expected {
			t.Errorf("path = %q, want %q", path, expected)
		}
	})
}
