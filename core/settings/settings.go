package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Benniphx/claude-statusline/core/ports"
)

// Result holds the outcome of a setup operation.
type Result struct {
	Action  string // "noop", "created", "updated"
	Path    string // settings file that was modified
	Message string // human-readable description
}

// DesiredSettings returns the settings keys the plugin needs configured.
// To add future settings, add entries to this map.
func DesiredSettings(binaryPath string) map[string]any {
	return map[string]any{
		"statusLine": map[string]any{
			"type":    "command",
			"command": binaryPath,
		},
	}
}

// ManagedKeys returns the top-level keys managed by this plugin.
func ManagedKeys() []string {
	return []string{"statusLine"}
}

// DeepMerge merges src into dst recursively. For nested maps it recurses;
// for all other types src overwrites dst. Returns a new map.
func DeepMerge(dst, src map[string]any) map[string]any {
	out := make(map[string]any, len(dst))
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := out[k].(map[string]any); ok {
				out[k] = DeepMerge(dstMap, srcMap)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// NeedsUpdate returns true if current does not contain all key-value pairs
// from desired (deep comparison).
func NeedsUpdate(current, desired map[string]any) bool {
	for k, dv := range desired {
		cv, ok := current[k]
		if !ok {
			return true
		}
		dMap, dIsMap := dv.(map[string]any)
		cMap, cIsMap := cv.(map[string]any)
		if dIsMap && cIsMap {
			if NeedsUpdate(cMap, dMap) {
				return true
			}
			continue
		}
		if !reflect.DeepEqual(cv, dv) {
			return true
		}
	}
	return false
}

// FindSettingsFile locates the settings file where this plugin is enabled.
// It checks projectDir/.claude/settings.json first, then ~/.claude/settings.json.
func FindSettingsFile(projectDir, pluginID string) (path string, exists bool) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(projectDir, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.json"),
	}
	for _, f := range candidates {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), pluginID) {
			return f, true
		}
	}
	// Fallback: user-scope settings
	fallback := filepath.Join(home, ".claude", "settings.json")
	_, err := os.Stat(fallback)
	return fallback, err == nil
}

// Setup ensures the settings file contains all desired settings.
// It is idempotent: if settings already match, it returns a noop result.
func Setup(settingsPath, binaryPath string, store ports.CacheStore) (Result, error) {
	desired := DesiredSettings(binaryPath)

	data, err := store.ReadFile(settingsPath)
	if err != nil {
		// File does not exist — create it.
		merged, merr := marshalSettings(desired)
		if merr != nil {
			return Result{}, fmt.Errorf("marshal: %w", merr)
		}
		if werr := store.AtomicWrite(settingsPath, merged); werr != nil {
			return Result{}, fmt.Errorf("write %s: %w", settingsPath, werr)
		}
		return Result{
			Action:  "created",
			Path:    settingsPath,
			Message: fmt.Sprintf("Statusline configured in %s.", settingsPath),
		}, nil
	}

	var current map[string]any
	if err := json.Unmarshal(data, &current); err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", settingsPath, err)
	}

	if !NeedsUpdate(current, desired) {
		return Result{
			Action:  "noop",
			Path:    settingsPath,
			Message: "Statusline ready.",
		}, nil
	}

	updated := DeepMerge(current, desired)
	out, err := marshalSettings(updated)
	if err != nil {
		return Result{}, fmt.Errorf("marshal: %w", err)
	}
	if err := store.AtomicWrite(settingsPath, out); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", settingsPath, err)
	}
	return Result{
		Action:  "updated",
		Path:    settingsPath,
		Message: fmt.Sprintf("Statusline updated in %s.", settingsPath),
	}, nil
}

// marshalSettings serializes a settings map to indented JSON with a trailing newline.
func marshalSettings(data map[string]any) ([]byte, error) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
