package ratelimit

import (
	"fmt"
	"math"
	"time"

	"github.com/Benniphx/claude-statusline/adapter/render"
	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

const fiveHourSecs = 18000 // 5 hours in seconds

// CalculatePace computes pace metrics for 5-hour and 7-day windows.
func CalculatePace(data types.RateLimitData, cfg types.Config, plat ports.PlatformInfo) types.PaceInfo {
	now := time.Now()
	pace := types.PaceInfo{}

	// 5-hour pace
	pace.FiveHourPace, pace.HittingLimit, pace.ResetInfo = calcFiveHourPace(data, now)

	// 7-day pace
	pace.SevenDayPace, pace.SevenDayResetFmt = calcSevenDayPace(data, cfg, plat, now)

	return pace
}

func calcFiveHourPace(data types.RateLimitData, now time.Time) (pace float64, hitting bool, resetInfo string) {
	remainingSecs := data.FiveHourReset.Sub(now).Seconds()
	if remainingSecs < 0 {
		remainingSecs = 0
	}

	// Round reset to nearest 5 minutes
	resetEpoch := data.FiveHourReset.Unix()
	resetRounded := ((resetEpoch + 150) / 300) * 300
	remainingSecsRounded := float64(resetRounded) - float64(now.Unix())
	if remainingSecsRounded < 0 {
		remainingSecsRounded = 0
	}

	secsSinceStart := float64(fiveHourSecs) - remainingSecs
	if secsSinceStart <= 0 {
		return 0, false, ""
	}

	hoursSinceStart := secsSinceStart / 3600.0
	if hoursSinceStart <= 0 {
		return 0, false, ""
	}

	percentPerHour := data.FiveHourPercent / hoursSinceStart
	pace = percentPerHour / 20.0 // 20% per hour = 1.0x sustainable

	// Hitting limit check
	if percentPerHour > 0 {
		remainingPercent := 100.0 - data.FiveHourPercent
		runwayMin := (remainingPercent / percentPerHour) * 60.0
		runwaySecs := runwayMin * 60.0
		if runwaySecs < remainingSecs {
			hitting = true
		}
	}

	// Reset time display
	resetH := int(remainingSecsRounded) / 3600
	resetM := (int(remainingSecsRounded) % 3600) / 60

	var resetTimeStr string
	if resetH > 0 {
		resetTimeStr = fmt.Sprintf("%dh%dm", resetH, resetM)
	} else {
		resetTimeStr = fmt.Sprintf("%dm", resetM)
	}

	// Show reset info based on remaining time
	// Format: DIM→RESETCYAN{time}RESET [DIM@RESETCYAN{local}RESET]
	remainingMin := remainingSecsRounded / 60.0
	if remainingMin <= 30 {
		localTime := time.Unix(resetRounded, 0).Format("15:04")
		resetInfo = fmt.Sprintf("%s→%s%s%s %s@%s%s%s",
			render.DimCode, render.Reset,
			render.Cyan, resetTimeStr+render.Reset,
			render.DimCode, render.Reset,
			render.Cyan, localTime+render.Reset)
	} else if remainingMin <= 60 {
		resetInfo = fmt.Sprintf("%s→%s%s%s%s",
			render.DimCode, render.Reset,
			render.Cyan, resetTimeStr, render.Reset)
	}

	return pace, hitting, resetInfo
}

func calcSevenDayPace(data types.RateLimitData, cfg types.Config, plat ports.PlatformInfo, now time.Time) (pace float64, resetFmt string) {
	remainingSecs := data.SevenDayReset.Sub(now).Seconds()
	if remainingSecs < 0 {
		remainingSecs = 0
	}

	daysLeft := int(remainingSecs / 86400.0)
	calendarDaysElapsed := 7 - daysLeft
	if calendarDaysElapsed <= 0 {
		return 0, ""
	}

	workDaysElapsed := float64(calendarDaysElapsed) * float64(cfg.WorkDaysPerWeek) / 7.0
	if workDaysElapsed <= 0.1 {
		return 0, ""
	}

	sustainablePerDay := 100.0 / float64(cfg.WorkDaysPerWeek)
	actualPerDay := data.SevenDayPercent / workDaysElapsed
	pace = actualPerDay / sustainablePerDay
	pace = math.Round(pace*10) / 10

	// Format reset: "→Xd" with dim/cyan coloring, only when ≤3 days
	if daysLeft <= 3 {
		var daysStr string
		if daysLeft > 0 {
			daysStr = fmt.Sprintf("%dd", daysLeft)
		} else {
			daysStr = "<1d"
		}
		resetFmt = fmt.Sprintf("%s→%s%s%s%s",
			render.DimCode, render.Reset,
			render.Cyan, daysStr, render.Reset)
	}

	return pace, resetFmt
}
