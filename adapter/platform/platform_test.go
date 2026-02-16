package platform

import (
	"testing"
	"time"
)

func TestParseISODate(t *testing.T) {
	p := Detect()

	tests := []struct {
		input string
		valid bool
	}{
		{"2025-02-06T14:30:45Z", true},
		{"2025-02-06T14:30:45+01:00", true},
		{"not-a-date", false},
		{"", false},
	}

	for _, tt := range tests {
		_, err := p.ParseISODate(tt.input)
		if (err == nil) != tt.valid {
			t.Errorf("ParseISODate(%q): err=%v, wantValid=%v", tt.input, err, tt.valid)
		}
	}
}

func TestCountWorkDays(t *testing.T) {
	p := Detect()

	tests := []struct {
		days            float64
		workDaysPerWeek int
		expected        float64
	}{
		{7, 5, 5},   // Full week, 5 work days
		{7, 7, 7},   // Full week, 7 work days
		{14, 5, 10}, // Two weeks, 5 work days
		{1, 5, 0.7142857142857143}, // 1 day with 5-day week
	}

	for _, tt := range tests {
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		end := start.Add(time.Duration(tt.days*24) * time.Hour)
		got := p.CountWorkDays(start, end, tt.workDaysPerWeek)
		diff := got - tt.expected
		if diff < -0.01 || diff > 0.01 {
			t.Errorf("CountWorkDays(%v days, %d work days) = %f, want %f",
				tt.days, tt.workDaysPerWeek, got, tt.expected)
		}
	}
}

func TestFormatTime(t *testing.T) {
	p := Detect()
	ts := time.Date(2025, 2, 6, 14, 30, 0, 0, time.UTC)

	got := p.FormatTime(ts, "02.01 15:04")
	if got != "06.02 14:30" {
		t.Errorf("FormatTime = %q, want %q", got, "06.02 14:30")
	}
}
