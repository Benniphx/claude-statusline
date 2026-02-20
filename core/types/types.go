package types

import "time"

// Input represents the JSON structure read from stdin (provided by Claude Code).
type Input struct {
	ContextWindow ContextWindow `json:"context_window"`
	Model         Model         `json:"model"`
	Cost          Cost          `json:"cost"`
}

// ContextWindow holds context window sizing and usage data.
type ContextWindow struct {
	ContextWindowSize int          `json:"context_window_size"`
	UsedPercentage    *float64     `json:"used_percentage,omitempty"`
	CurrentUsage      CurrentUsage `json:"current_usage"`
	TotalInputTokens  int          `json:"total_input_tokens"`
	TotalOutputTokens int          `json:"total_output_tokens"`
}

// CurrentUsage holds granular token counts for the current context.
type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// Model identifies the model being used.
type Model struct {
	DisplayName string `json:"display_name"`
	ModelID     string `json:"model_id"`
}

// Cost holds session cost and duration data.
type Cost struct {
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalDurationMS   int     `json:"total_duration_ms"`
	TotalLinesAdded   int     `json:"total_lines_added"`
	TotalLinesRemoved int     `json:"total_lines_removed"`
}

// Config holds user configuration settings.
type Config struct {
	ContextWarningThreshold int           // 0 = disabled, 1-100 = show warning at this %
	RateCacheTTL            time.Duration // How often to refresh rate limit data
	WorkDaysPerWeek         int           // 1-7, used for 7-day pace scaling
	CacheDir                string        // Directory for cache files
	Version                 string        // Current binary version
	CostNormalize           bool          // Enable cost-normalized burn rate
	CostWeightOpus          float64       // Cost weight for Opus models (default 5.0)
	CostWeightSonnet        float64       // Cost weight for Sonnet models (default 1.0, baseline)
	CostWeightHaiku         float64       // Cost weight for Haiku models (default 0.25)
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ContextWarningThreshold: 0,
		RateCacheTTL:            15 * time.Second,
		WorkDaysPerWeek:         5,
		CacheDir:                "/tmp",
		Version:                 "dev",
		CostNormalize:           true,
		CostWeightOpus:          5.0,
		CostWeightSonnet:        1.0,
		CostWeightHaiku:         0.25,
	}
}

// Credentials holds authentication credentials for the Anthropic API.
type Credentials struct {
	OAuthToken string
	APIKey     string
}

// HasOAuth returns true if OAuth credentials are available.
func (c Credentials) HasOAuth() bool {
	return c.OAuthToken != ""
}

// ModelInfo holds resolved model information.
type ModelInfo struct {
	ShortName      string
	DefaultContext int
	IsLocal        bool
	CostTier       CostTier // Model cost tier for normalization
}

// CostTier represents the pricing tier of a model.
type CostTier int

const (
	CostTierSonnet CostTier = iota // Default/baseline
	CostTierOpus
	CostTierHaiku
)

// ContextDisplay holds computed context window display data.
type ContextDisplay struct {
	PercentUsed  int
	TokensUsed   int
	TokensTotal  int
	IsInitial    bool // True when no data yet (0 tokens, 0 duration)
	LinesAdded   int
	LinesRemoved int
	DurationMin  int
}

// RateLimitResponse holds the API response for rate limits.
type RateLimitResponse struct {
	FiveHour RateLimitWindow `json:"five_hour"`
	SevenDay RateLimitWindow `json:"seven_day"`
}

// RateLimitWindow holds data for a single rate limit window.
type RateLimitWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

// RateLimitData holds processed rate limit information.
type RateLimitData struct {
	FiveHourPercent float64
	FiveHourReset   time.Time
	SevenDayPercent float64
	SevenDayReset   time.Time
	FromCache       bool
}

// PaceInfo holds calculated pace information.
type PaceInfo struct {
	FiveHourPace    float64
	SevenDayPace    float64
	HittingLimit    bool
	ResetInfo       string // e.g., "→45m @14:30"
	SevenDayResetFmt string
}

// BurnInfo holds burn rate metrics.
type BurnInfo struct {
	LocalTPM       float64 // Local tokens per minute
	GlobalTPM      float64 // Global tokens per minute (from daemon)
	IsHighActivity bool    // True when global >> local
}

// CostDisplay holds computed cost display data.
type CostDisplay struct {
	SessionCost float64
	DailyCost   float64
	CostPerHour float64
	SessionID   string
}
