package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

// Load reads configuration from XDG or legacy paths and returns a Config.
func Load() types.Config {
	cfg := types.DefaultConfig()

	// Determine cache dir
	if d := os.Getenv("CLAUDE_CODE_TMPDIR"); d != "" {
		cfg.CacheDir = d
	}

	// Try XDG config, then legacy
	paths := configPaths()
	for _, p := range paths {
		if parseFile(p, &cfg) {
			break
		}
	}

	return cfg
}

func configPaths() []string {
	var paths []string

	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		if home, err := os.UserHomeDir(); err == nil {
			xdg = filepath.Join(home, ".config")
		}
	}
	if xdg != "" {
		paths = append(paths, filepath.Join(xdg, "claude-statusline", "config"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".claude-statusline.conf"))
	}

	return paths
}

func parseFile(path string, cfg *types.Config) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "CONTEXT_WARNING_THRESHOLD":
			if n, err := strconv.Atoi(value); err == nil && n >= 1 && n <= 100 {
				cfg.ContextWarningThreshold = n
			}
		case "RATE_CACHE_TTL":
			if n, err := strconv.Atoi(value); err == nil && n >= 10 && n <= 120 {
				cfg.RateCacheTTL = time.Duration(n) * time.Second
			}
		case "WORK_DAYS_PER_WEEK":
			if n, err := strconv.Atoi(value); err == nil && n >= 1 && n <= 7 {
				cfg.WorkDaysPerWeek = n
			}
		case "COST_NORMALIZE":
			cfg.CostNormalize = parseBool(value)
		case "COST_WEIGHT_OPUS":
			if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 {
				cfg.CostWeightOpus = f
			}
		case "COST_WEIGHT_SONNET":
			if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 {
				cfg.CostWeightSonnet = f
			}
		case "COST_WEIGHT_HAIKU":
			if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 {
				cfg.CostWeightHaiku = f
			}
		}
	}

	return true
}

func parseBool(s string) bool {
	lower := strings.ToLower(s)
	return lower == "true" || lower == "1" || lower == "yes"
}
