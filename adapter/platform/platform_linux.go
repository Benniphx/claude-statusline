//go:build linux

package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Benniphx/claude-statusline/core/types"
)

// GetCredentials retrieves OAuth credentials on Linux.
// Priority: env var → credentials file → error.
func (p *Platform) GetCredentials() (types.Credentials, error) {
	// 1. Environment variable
	if token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); token != "" {
		return types.Credentials{OAuthToken: token}, nil
	}

	// 2. Credentials file
	home, err := os.UserHomeDir()
	if err != nil {
		return types.Credentials{}, fmt.Errorf("no home directory: %w", err)
	}

	paths := []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(home, ".claude", "credentials.json"),
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var creds struct {
			ClaudeAiOauth struct {
				AccessToken string `json:"accessToken"`
			} `json:"claudeAiOauth"`
		}
		if err := json.Unmarshal(data, &creds); err == nil && creds.ClaudeAiOauth.AccessToken != "" {
			return types.Credentials{OAuthToken: creds.ClaudeAiOauth.AccessToken}, nil
		}
	}

	return types.Credentials{}, fmt.Errorf("no credentials found")
}

// HasClaudeProcesses checks if any Claude Code processes are running.
func (p *Platform) HasClaudeProcesses() bool {
	out, err := exec.Command("pgrep", "-f", "[c]laude").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// getSessionFromProcessTree walks the process tree on Linux using /proc.
func (p *Platform) getSessionFromProcessTree() (string, bool) {
	pid := os.Getppid()
	for i := 0; i < 10; i++ {
		commPath := filepath.Join("/proc", strconv.Itoa(pid), "comm")
		comm, err := os.ReadFile(commPath)
		if err != nil {
			break
		}
		if strings.Contains(strings.ToLower(strings.TrimSpace(string(comm))), "claude") {
			return strconv.Itoa(pid), true
		}

		statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
		status, err := os.ReadFile(statusPath)
		if err != nil {
			break
		}
		ppid := 0
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "PPid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					ppid, _ = strconv.Atoi(fields[1])
				}
				break
			}
		}
		if ppid <= 1 {
			break
		}
		pid = ppid
	}

	// Fallback: use PPID (unstable)
	return strconv.Itoa(os.Getppid()), false
}
