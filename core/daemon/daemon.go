package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

const (
	refreshInterval = 15 * time.Second
	maxIdleChecks   = 4
	lockFile        = "/tmp/claude_statusline_daemon.lock"
	pidFile         = "/tmp/claude_statusline_daemon.pid"
	logFile         = "/tmp/claude_statusline_daemon.log"
	rateCacheFile   = "claude_rate_limit_cache.json"
	burnCacheFile   = "claude_global_burn.json"
	tokensPerPct    = 5000 // 1% of 5h utilization ≈ 5000 tokens
)

type burnState struct {
	TokensPerMin float64 `json:"tokens_per_min"`
	LastPct      float64 `json:"last_pct"`
	LastTime     int64   `json:"last_time"`
}

// Run starts the daemon loop.
func Run(cfg types.Config, creds types.Credentials, plat ports.ProcessDetector, store ports.CacheStore, api ports.APIClient) error {
	// Acquire lock
	lock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("daemon already running: %w", err)
	}
	defer releaseLock(lock)

	// Write PID file
	os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	defer os.Remove(pidFile)

	idleCount := 0
	var state burnState

	// Load previous burn state
	if data, err := store.ReadFile(fmt.Sprintf("%s/%s", cfg.CacheDir, burnCacheFile)); err == nil {
		json.Unmarshal(data, &state)
	}

	for {
		if !plat.HasClaudeProcesses() {
			idleCount++
			if idleCount >= maxIdleChecks {
				logMsg("No Claude processes for %d checks, exiting", idleCount)
				return nil
			}
			time.Sleep(refreshInterval)
			continue
		}

		idleCount = 0

		// Fetch rate limits
		if creds.HasOAuth() {
			resp, err := api.FetchRateLimits(creds.OAuthToken)
			if err != nil {
				logMsg("Error fetching rate limits: %v", err)
			} else {
				// Cache response
				if raw, err := json.Marshal(resp); err == nil {
					cachePath := fmt.Sprintf("%s/%s", cfg.CacheDir, rateCacheFile)
					store.AtomicWrite(cachePath, raw)
				}

				// Update burn rate
				state = updateBurnRate(state, resp.FiveHour.Utilization)
				burnData, _ := json.Marshal(map[string]float64{"tokens_per_min": state.TokensPerMin})
				store.WriteFile(fmt.Sprintf("%s/%s", cfg.CacheDir, burnCacheFile), burnData)
			}
		}

		time.Sleep(refreshInterval)
	}
}

func updateBurnRate(prev burnState, currentPct float64) burnState {
	now := time.Now().Unix()
	state := burnState{
		LastPct:  currentPct,
		LastTime: now,
	}

	if prev.LastTime == 0 {
		return state
	}

	deltaSecs := float64(now - prev.LastTime)
	if deltaSecs <= 0 {
		state.TokensPerMin = prev.TokensPerMin
		return state
	}

	deltaPct := currentPct - prev.LastPct

	if deltaPct <= 0 {
		// No change or decrease: decay
		state.TokensPerMin = prev.TokensPerMin * 0.5
	} else {
		// Active: calculate new TPM
		pctPerMin := (deltaPct / deltaSecs) * 60.0
		newTPM := pctPerMin * tokensPerPct

		// Light smoothing: 80% new, 20% old
		if prev.TokensPerMin > 0 {
			state.TokensPerMin = newTPM*0.8 + prev.TokensPerMin*0.2
		} else {
			state.TokensPerMin = newTPM
		}
	}

	return state
}

func acquireLock() (*os.File, error) {
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("lock held by another process")
	}

	return f, nil
}

func releaseLock(f *os.File) {
	if f != nil {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		os.Remove(lockFile)
	}
}

func logMsg(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n", ts, msg)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(entry)
}

// IsRunning checks if a daemon is currently running.
func IsRunning() bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. Send signal 0 to check.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
