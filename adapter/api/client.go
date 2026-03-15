package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Benniphx/claude-statusline/core/types"
)

const (
	backoffStateFile = "claude_statusline_backoff.json"
	initialBackoff   = 30 * time.Second
	maxBackoff       = 5 * time.Minute
)

// backoffState persists across invocations via filesystem.
type backoffState struct {
	Until    int64 `json:"until"`     // Unix timestamp until which to skip API calls
	Duration int64 `json:"duration"`  // Current backoff duration in seconds
}

// Client implements the ports.APIClient interface.
type Client struct {
	httpClient *http.Client
	cacheDir   string
	version    string
}

// New creates a new API client.
func New() *Client {
	return NewWithCacheDir("/tmp", "")
}

// NewWithCacheDir creates a new API client with a specific cache directory and version.
func NewWithCacheDir(cacheDir, version string) *Client {
	return &Client{
		httpClient: &http.Client{},
		cacheDir:   cacheDir,
		version:    version,
	}
}

// FetchRateLimits retrieves rate limit data from the Anthropic OAuth usage API.
// Implements exponential backoff on 429 responses, persisted across invocations.
func (c *Client) FetchRateLimits(token string) (*types.RateLimitResponse, error) {
	// Check backoff state — skip API call if still in cooldown
	if remaining := c.backoffRemaining(); remaining > 0 {
		return nil, fmt.Errorf("backoff active: %v remaining (429 cooldown)", remaining.Round(time.Second))
	}

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	// Use claude-code User-Agent for more generous rate limit bucket
	ccVersion := c.version
	if ccVersion == "" {
		ccVersion = os.Getenv("CLAUDE_CODE_VERSION")
	}
	if ccVersion == "" {
		ccVersion = "1.0.0"
	}
	req.Header.Set("User-Agent", "claude-code/"+ccVersion)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching rate limits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		c.escalateBackoff()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned 429 (rate limited, backoff escalated): %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	// Success — clear backoff
	c.clearBackoff()

	var result types.RateLimitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

// backoffRemaining returns how long until the backoff expires.
func (c *Client) backoffRemaining() time.Duration {
	state := c.loadBackoff()
	if state.Until == 0 {
		return 0
	}
	remaining := time.Until(time.Unix(state.Until, 0))
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// escalateBackoff doubles the backoff duration (capped at maxBackoff).
func (c *Client) escalateBackoff() {
	state := c.loadBackoff()

	dur := time.Duration(state.Duration) * time.Second
	if dur < initialBackoff {
		dur = initialBackoff
	} else {
		dur *= 2
		if dur > maxBackoff {
			dur = maxBackoff
		}
	}

	state.Duration = int64(dur.Seconds())
	state.Until = time.Now().Add(dur).Unix()
	c.saveBackoff(state)
}

// clearBackoff resets the backoff state after a successful request.
func (c *Client) clearBackoff() {
	c.saveBackoff(backoffState{})
}

func (c *Client) backoffPath() string {
	return filepath.Join(c.cacheDir, backoffStateFile)
}

func (c *Client) loadBackoff() backoffState {
	var state backoffState
	data, err := os.ReadFile(c.backoffPath())
	if err != nil {
		return state
	}
	json.Unmarshal(data, &state)
	return state
}

func (c *Client) saveBackoff(state backoffState) {
	data, _ := json.Marshal(state)
	os.WriteFile(c.backoffPath(), data, 0o644)
}

// FetchLatestRelease gets the latest release tag from a GitHub repository.
func (c *Client) FetchLatestRelease(repo string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding release: %w", err)
	}

	return release.TagName, nil
}
