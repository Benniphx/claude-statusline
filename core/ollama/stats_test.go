package ollama

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestStats(t *testing.T, dir string, stats Stats) string {
	t.Helper()
	path := filepath.Join(dir, "claude_ollama_stats.json")
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadStats(t *testing.T) {
	dir := t.TempDir()
	path := writeTestStats(t, dir, Stats{
		Requests:              42,
		TotalPromptTokens:     156000,
		TotalCompletionTokens: 28000,
		LastUpdated:           time.Now().Unix(),
	})

	stats, err := ReadStats(path)
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}
	if stats.Requests != 42 {
		t.Errorf("Requests = %d, want 42", stats.Requests)
	}
	if stats.TotalPromptTokens != 156000 {
		t.Errorf("TotalPromptTokens = %d, want 156000", stats.TotalPromptTokens)
	}
	if stats.TotalCompletionTokens != 28000 {
		t.Errorf("TotalCompletionTokens = %d, want 28000", stats.TotalCompletionTokens)
	}
}

func TestReadStatsStale(t *testing.T) {
	dir := t.TempDir()
	path := writeTestStats(t, dir, Stats{
		Requests:    10,
		LastUpdated: time.Now().Add(-10 * time.Minute).Unix(),
	})

	_, err := ReadStats(path)
	if err == nil {
		t.Error("expected error for stale stats, got nil")
	}
}

func TestReadStatsFileNotFound(t *testing.T) {
	_, err := ReadStats("/nonexistent/path/stats.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestReadStatsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadStats(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestSavings(t *testing.T) {
	tests := []struct {
		name  string
		stats *Stats
		want  float64
	}{
		{
			name:  "nil stats",
			stats: nil,
			want:  0,
		},
		{
			name: "typical usage",
			stats: &Stats{
				TotalPromptTokens:     1_000_000,
				TotalCompletionTokens: 200_000,
			},
			// 1M * $0.25/M + 200K * $1.25/M = $0.25 + $0.25 = $0.50
			want: 0.50,
		},
		{
			name: "zero tokens",
			stats: &Stats{
				TotalPromptTokens:     0,
				TotalCompletionTokens: 0,
			},
			want: 0,
		},
		{
			name: "large usage",
			stats: &Stats{
				TotalPromptTokens:     10_000_000,
				TotalCompletionTokens: 2_000_000,
			},
			// 10M * $0.25/M + 2M * $1.25/M = $2.50 + $2.50 = $5.00
			want: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Savings(tt.stats)
			if got != tt.want {
				t.Errorf("Savings() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name  string
		stats *Stats
		want  string
	}{
		{
			name:  "nil stats",
			stats: nil,
			want:  "",
		},
		{
			name:  "zero requests",
			stats: &Stats{Requests: 0},
			want:  "",
		},
		{
			name: "requests with savings",
			stats: &Stats{
				Requests:              42,
				TotalPromptTokens:     1_000_000,
				TotalCompletionTokens: 200_000,
			},
			want: "42 req | saved ~$0.50",
		},
		{
			name: "requests without meaningful savings",
			stats: &Stats{
				Requests:              3,
				TotalPromptTokens:     100,
				TotalCompletionTokens: 50,
			},
			want: "3 req",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.stats)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}
