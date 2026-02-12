package update

import (
	"strconv"
	"strings"
	"time"

	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/adapter/render"
)

const cacheFile = "claude_statusline_update.txt"
const cacheTTL = 24 * time.Hour
const repo = "jobrad-gmbh/devex-claude-marketplace"

// Check determines if an update is available.
// Returns true and the latest version if newer than current.
func Check(currentVersion string, cfg_cacheDir string, store ports.CacheStore, api ports.APIClient) (bool, string) {
	if currentVersion == "" || currentVersion == "dev" {
		return false, ""
	}

	cachePath := cfg_cacheDir + "/" + cacheFile

	// Check cache
	if data, fresh := store.ReadIfFresh(cachePath, cacheTTL); fresh {
		latest := strings.TrimSpace(string(data))
		if latest != "" && GreaterThan(latest, currentVersion) {
			return true, latest
		}
		return false, ""
	}

	// Fetch in background on first miss
	go func() {
		latest, err := api.FetchLatestRelease(repo)
		if err != nil {
			return
		}
		latest = strings.TrimPrefix(latest, "v")
		store.WriteFile(cachePath, []byte(latest))
	}()

	return false, ""
}

// Render produces the update notice string, or empty if no update.
func Render(currentVersion, cacheDir string, store ports.CacheStore, api ports.APIClient, r ports.Renderer) string {
	hasUpdate, _ := Check(currentVersion, cacheDir, store, api)
	if !hasUpdate {
		return ""
	}
	return " " + r.Color("[Update]", render.Yellow)
}

// GreaterThan returns true if v1 is semantically greater than v2.
func GreaterThan(v1, v2 string) bool {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	if v1 == v2 {
		return false
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}
		if n1 > n2 {
			return true
		}
		if n1 < n2 {
			return false
		}
	}

	return false
}
