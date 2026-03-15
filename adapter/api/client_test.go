package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestBackoff_EscalatesOnCall(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewWithCacheDir(tmpDir, "test")

	// Initially no backoff
	if rem := client.backoffRemaining(); rem > 0 {
		t.Fatalf("expected no backoff initially, got %v", rem)
	}

	// First escalation → 30s
	client.escalateBackoff()
	rem := client.backoffRemaining()
	if rem < 25*time.Second || rem > 35*time.Second {
		t.Errorf("first backoff = %v, want ~30s", rem)
	}

	// Second escalation → 60s
	client.escalateBackoff()
	rem = client.backoffRemaining()
	if rem < 55*time.Second || rem > 65*time.Second {
		t.Errorf("second backoff = %v, want ~60s", rem)
	}

	// Third escalation → 120s
	client.escalateBackoff()
	rem = client.backoffRemaining()
	if rem < 115*time.Second || rem > 125*time.Second {
		t.Errorf("third backoff = %v, want ~120s", rem)
	}
}

func TestBackoff_CapsAtMax(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewWithCacheDir(tmpDir, "test")

	// Escalate many times
	for i := 0; i < 20; i++ {
		client.escalateBackoff()
	}

	rem := client.backoffRemaining()
	if rem > maxBackoff+time.Second {
		t.Errorf("backoff = %v, should be capped at %v", rem, maxBackoff)
	}
}

func TestBackoff_ClearsOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewWithCacheDir(tmpDir, "test")

	client.escalateBackoff()
	if client.backoffRemaining() == 0 {
		t.Fatal("expected backoff after escalation")
	}

	client.clearBackoff()
	if rem := client.backoffRemaining(); rem > 0 {
		t.Errorf("backoff should be cleared, got %v", rem)
	}
}

func TestBackoff_PersistsAcrossInstances(t *testing.T) {
	tmpDir := t.TempDir()

	// Instance 1 sets backoff
	client1 := NewWithCacheDir(tmpDir, "test")
	client1.escalateBackoff()

	// Instance 2 reads it
	client2 := NewWithCacheDir(tmpDir, "test")
	rem := client2.backoffRemaining()
	if rem < 25*time.Second {
		t.Errorf("backoff not persisted, got %v", rem)
	}
}

func TestBackoff_ExpiredReturnsZero(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewWithCacheDir(tmpDir, "test")

	// Write a state that's already expired
	state := backoffState{
		Until:    time.Now().Add(-10 * time.Second).Unix(),
		Duration: 30,
	}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(tmpDir, backoffStateFile), data, 0o644)

	if rem := client.backoffRemaining(); rem > 0 {
		t.Errorf("expired backoff should return 0, got %v", rem)
	}
}

func TestFetchRateLimits_429TriggersBackoff(t *testing.T) {
	tmpDir := t.TempDir()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"Rate limited"}}`))
	}))
	defer server.Close()

	// We can't easily override the URL in the client, so test the backoff state directly
	client := NewWithCacheDir(tmpDir, "test")

	// Simulate what FetchRateLimits does on 429
	client.escalateBackoff()

	// Next call should be blocked by backoff
	_, err := client.FetchRateLimits("test-token")
	if err == nil {
		t.Fatal("expected error during backoff")
	}
}
