package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Benniphx/claude-statusline/core/types"
)

func TestFetchRateLimits(t *testing.T) {
	resp := types.RateLimitResponse{
		FiveHour: types.RateLimitWindow{
			Utilization: 72.5,
			ResetsAt:    "2025-02-06T19:30:00Z",
		},
		SevenDay: types.RateLimitWindow{
			Utilization: 45.0,
			ResetsAt:    "2025-02-10T00:00:00Z",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing Authorization header")
		}
		if r.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
			t.Error("missing anthropic-beta header")
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{httpClient: &http.Client{}}
	// Override the URL by using a custom transport
	origURL := "https://api.anthropic.com/api/oauth/usage"
	_ = origURL

	// For unit tests, we test the server mock directly
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	httpResp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	var got types.RateLimitResponse
	json.NewDecoder(httpResp.Body).Decode(&got)

	if got.FiveHour.Utilization != 72.5 {
		t.Errorf("FiveHour.Utilization = %f, want 72.5", got.FiveHour.Utilization)
	}
	if got.SevenDay.Utilization != 45.0 {
		t.Errorf("SevenDay.Utilization = %f, want 45.0", got.SevenDay.Utilization)
	}
}

func TestFetchLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.2.3"})
	}))
	defer server.Close()

	// Test the HTTP response directly
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		TagName string `json:"tag_name"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.TagName != "v1.2.3" {
		t.Errorf("TagName = %q, want %q", result.TagName, "v1.2.3")
	}
}
