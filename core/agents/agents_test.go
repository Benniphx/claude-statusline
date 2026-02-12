package agents

import "testing"

func TestIsClaudeProcess(t *testing.T) {
	tests := []struct {
		cmdline string
		want    bool
	}{
		{"/usr/local/bin/claude --session abc", true},
		{"/opt/homebrew/bin/node /Users/user/.claude/bin/claude", true},
		{"claude-code --daemon", true},
		{"git -c diff.mnemonicprefix=false log", false},
		{"pgrep -af claude", false},
		{"rg pattern src/", false},
		{"/usr/bin/node /path/to/node_modules/.bin/something", false},
		{"", false},
		{"some random process", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmdline, func(t *testing.T) {
			got := isClaudeProcess(tt.cmdline)
			if got != tt.want {
				t.Errorf("isClaudeProcess(%q) = %v, want %v", tt.cmdline, got, tt.want)
			}
		})
	}
}

func TestCountReturnsAgentInfo(t *testing.T) {
	// Count() calls pgrep which may or may not find processes,
	// but it should not panic and should return valid AgentInfo.
	info := Count()
	if info.Total < 0 {
		t.Errorf("Total should be >= 0, got %d", info.Total)
	}
	if info.Total <= 1 && info.HasSubagents {
		t.Errorf("HasSubagents should be false when Total <= 1")
	}
}
