//go:build darwin

package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Benniphx/claude-statusline/core/types"
)

// GetCredentials retrieves OAuth credentials on macOS.
// Priority: env var → keychain → error.
func (p *Platform) GetCredentials() (types.Credentials, error) {
	// 1. Environment variable
	if token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); token != "" {
		return types.Credentials{OAuthToken: token}, nil
	}

	// 2. macOS Keychain
	out, err := exec.Command("security", "find-generic-password",
		"-s", "Claude Code-credentials", "-w").Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		var creds struct {
			ClaudeAiOauth struct {
				AccessToken string `json:"accessToken"`
			} `json:"claudeAiOauth"`
		}
		if err := json.Unmarshal([]byte(raw), &creds); err == nil && creds.ClaudeAiOauth.AccessToken != "" {
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

// CountAgents counts running Claude Code agent processes.
// Returns total count and whether subagents are active (total > 1).
func (p *Platform) CountAgents() (int, bool) {
	out, err := exec.Command("pgrep", "-af", "[c]laude").Output()
	if err != nil {
		return 0, false
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	pids := make(map[int]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		cmdline := strings.Join(fields[1:], " ")
		if isClaudeAgentProcess(cmdline) {
			pids[pid] = true
		}
	}

	total := len(pids)
	return total, total > 1
}

// isClaudeAgentProcess checks if a command line is a Claude Code main process.
func isClaudeAgentProcess(cmdline string) bool {
	lower := strings.ToLower(cmdline)
	if !strings.Contains(lower, "claude") {
		return false
	}
	// Exclude child helper processes
	excludes := []string{"git", "pgrep", "rg ", "grep", "find ", "node_modules"}
	for _, ex := range excludes {
		if strings.Contains(lower, ex) {
			return false
		}
	}
	return true
}

// getSessionFromProcessTree walks the process tree on macOS using ps.
func (p *Platform) getSessionFromProcessTree() (string, bool) {
	pid := os.Getppid()
	for i := 0; i < 10; i++ {
		out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=,comm=").Output()
		if err != nil {
			break
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) < 2 {
			break
		}
		comm := strings.Join(fields[1:], " ")
		if strings.Contains(strings.ToLower(comm), "claude") {
			return strconv.Itoa(pid), true
		}
		ppid, err := strconv.Atoi(fields[0])
		if err != nil || ppid <= 1 {
			break
		}
		pid = ppid
	}

	// Fallback: use PPID (unstable)
	return strconv.Itoa(os.Getppid()), false
}
