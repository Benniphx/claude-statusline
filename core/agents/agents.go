package agents

import (
	"os/exec"
	"strconv"
	"strings"
)

// AgentInfo holds the result of counting Claude agent processes.
type AgentInfo struct {
	Total         int
	HasSubagents  bool
}

// CountClaude counts running Claude Code processes using pgrep.
// Returns total unique PIDs matching the claude pattern.
func CountClaude() int {
	out, err := exec.Command("pgrep", "-af", "[c]laude").Output()
	if err != nil {
		return 0
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
		// Only count processes that look like the Claude Code node process
		cmdline := strings.Join(fields[1:], " ")
		if isClaudeProcess(cmdline) {
			pids[pid] = true
		}
	}

	return len(pids)
}

// isClaudeProcess checks if a command line looks like a Claude Code main process
// (not a helper/child process like git, node modules, etc.)
func isClaudeProcess(cmdline string) bool {
	lower := strings.ToLower(cmdline)
	// Match the actual Claude CLI binary or node process running claude
	if strings.Contains(lower, "claude") {
		// Exclude common child processes
		excludes := []string{
			"git", "pgrep", "rg ", "grep", "find ",
			"node_modules", "eslint", "prettier",
		}
		for _, ex := range excludes {
			if strings.Contains(lower, ex) {
				return false
			}
		}
		return true
	}
	return false
}

// Count returns AgentInfo with the current count of Claude processes.
func Count() AgentInfo {
	total := CountClaude()
	return AgentInfo{
		Total:        total,
		HasSubagents: total > 1,
	}
}
