package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// TestOAuthUsageEndpointLive checks whether the /api/oauth/usage endpoint is
// currently functional. This is a live integration test — it requires a valid
// OAuth token in the environment.
//
// Run manually to check if the known 429 issue (anthropics/claude-code#30930)
// has been resolved:
//
//	go test ./adapter/api/... -run TestOAuthUsageEndpointLive -v
//
// The test is skipped automatically in CI (no CLAUDE_CODE_OAUTH_TOKEN set).
func TestOAuthUsageEndpointLive(t *testing.T) {
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if token == "" {
		t.Skip("CLAUDE_CODE_OAUTH_TOKEN not set — skipping live test")
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	t.Logf("HTTP %d — body: %v", resp.StatusCode, body)

	if resp.StatusCode == http.StatusTooManyRequests {
		t.Errorf("KNOWN ISSUE STILL ACTIVE: endpoint returns 429 (anthropics/claude-code#30930)\n%s",
			"See docs/known-issues.md for details and workaround.")
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status %d: %v", resp.StatusCode, body)
		return
	}

	// Verify expected fields are present
	for _, field := range []string{"five_hour", "seven_day"} {
		if _, ok := body[field]; !ok {
			t.Errorf("response missing field %q — API shape may have changed", field)
		}
	}

	fmt.Printf("✅ Endpoint is WORKING — known issue appears to be fixed!\n")
	fmt.Printf("   five_hour: %v\n", body["five_hour"])
	fmt.Printf("   seven_day: %v\n", body["seven_day"])
}
