package ratelimit

import (
	"testing"
	"time"

	"github.com/Benniphx/claude-statusline/adapter/platform"
	"github.com/Benniphx/claude-statusline/core/types"
)

func TestCalcFiveHourPace(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		percent    float64
		resetIn    time.Duration
		wantPace   float64 // approximate
		wantHit    bool
	}{
		{
			name:     "sustainable pace",
			percent:  20,
			resetIn:  4 * time.Hour, // 1h elapsed, 20% → 1.0x
			wantPace: 1.0,
		},
		{
			name:     "fast pace",
			percent:  60,
			resetIn:  4 * time.Hour, // 1h elapsed, 60% → 3.0x, hitting limit
			wantPace: 3.0,
			wantHit:  true,
		},
		{
			name:     "slow pace",
			percent:  10,
			resetIn:  3 * time.Hour, // 2h elapsed, 5%/h → 0.25x
			wantPace: 0.25,
		},
		{
			name:     "hitting limit",
			percent:  90,
			resetIn:  3 * time.Hour, // 2h elapsed, 90% → will hit 100%
			wantPace: 2.25,
			wantHit:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := types.RateLimitData{
				FiveHourPercent: tt.percent,
				FiveHourReset:   now.Add(tt.resetIn),
			}

			pace, hitting, _ := calcFiveHourPace(data, now)

			diff := pace - tt.wantPace
			if diff < -0.1 || diff > 0.1 {
				t.Errorf("pace = %.2f, want ≈%.2f", pace, tt.wantPace)
			}
			if hitting != tt.wantHit {
				t.Errorf("hitting = %v, want %v", hitting, tt.wantHit)
			}
		})
	}
}

func TestCalcSevenDayPace(t *testing.T) {
	now := time.Now()
	plat := platform.Detect()

	tests := []struct {
		name    string
		percent float64
		resetIn time.Duration
		wdpw    int
		wantPace float64
	}{
		{
			name:     "sustainable 5-day week",
			percent:  40,
			resetIn:  5 * 24 * time.Hour, // 2 calendar days elapsed
			wdpw:     5,
			wantPace: 1.4, // workDaysElapsed=2*5/7≈1.43, actual=40/1.43≈28, sustainable=20, pace=28/20=1.4
		},
		{
			name:     "fast pace",
			percent:  60,
			resetIn:  5 * 24 * time.Hour, // 2 days, 60%
			wdpw:     5,
			wantPace: 2.1, // workDaysElapsed=1.43, actual=60/1.43≈42, sustainable=20, pace≈2.1
		},
	}

	cfg := types.DefaultConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.WorkDaysPerWeek = tt.wdpw
			data := types.RateLimitData{
				SevenDayPercent: tt.percent,
				SevenDayReset:   now.Add(tt.resetIn),
			}

			pace, _ := calcSevenDayPace(data, cfg, plat, now)

			diff := pace - tt.wantPace
			if diff < -0.2 || diff > 0.2 {
				t.Errorf("pace = %.2f, want ≈%.2f", pace, tt.wantPace)
			}
		})
	}
}

func TestResetTimeDisplay(t *testing.T) {
	now := time.Now()

	// ≤30 min: shows time and clock
	data30 := types.RateLimitData{
		FiveHourPercent: 50,
		FiveHourReset:   now.Add(25 * time.Minute),
	}
	_, _, info30 := calcFiveHourPace(data30, now)
	if info30 == "" {
		t.Error("should show reset info for ≤30 min")
	}
	if len(info30) < 5 { // At minimum "→Xm @HH:MM"
		t.Errorf("reset info too short: %q", info30)
	}

	// >60 min: no time shown
	data90 := types.RateLimitData{
		FiveHourPercent: 50,
		FiveHourReset:   now.Add(90 * time.Minute),
	}
	_, _, info90 := calcFiveHourPace(data90, now)
	if info90 != "" {
		t.Errorf("should not show reset info for >60 min: %q", info90)
	}
}
