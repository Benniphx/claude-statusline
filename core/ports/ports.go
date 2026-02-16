package ports

import (
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

// CredentialProvider retrieves OAuth/API credentials.
type CredentialProvider interface {
	GetCredentials() (types.Credentials, error)
}

// CacheStore handles file-based caching with atomic writes.
type CacheStore interface {
	AtomicWrite(path string, data []byte) error
	ReadIfFresh(path string, ttl time.Duration) ([]byte, bool)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	FileMTime(path string) (time.Time, error)
	CleanOld(dir, pattern, keep string) error
}

// APIClient communicates with external APIs.
type APIClient interface {
	FetchRateLimits(token string) (*types.RateLimitResponse, error)
	FetchLatestRelease(repo string) (string, error)
}

// Renderer produces ANSI-formatted output.
type Renderer interface {
	Colorize(text string, percent int) string
	Color(text, color string) string
	Dim(text string) string
	MakeBar(percent, width int) string
	FormatTokens(n int) string  // 0 decimals: "200K"
	FormatTokensF(n int) string // 1 decimal:  "100.0K"
	FormatCost(f float64) string
}

// PlatformInfo provides OS-specific operations.
type PlatformInfo interface {
	ParseISODate(s string) (time.Time, error)
	FormatTime(t time.Time, format string) string
	CountWorkDays(start, end time.Time, workDaysPerWeek int) float64
	GetStableSessionID() (id string, stable bool)
}

// ProcessDetector checks for running Claude processes.
type ProcessDetector interface {
	HasClaudeProcesses() bool
}

// AgentCounter counts running Claude agent processes.
type AgentCounter interface {
	CountAgents() (total int, hasSubagents bool)
}

// OllamaClient queries the local Ollama API.
type OllamaClient interface {
	GetContextSize(model string) (int, error)
}
