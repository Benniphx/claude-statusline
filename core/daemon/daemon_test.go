package daemon

import (
	"testing"
)

func TestUpdateBurnRate_NoPrevious(t *testing.T) {
	state := burnState{}
	result := updateBurnRate(state, 50.0)

	if result.LastPct != 50.0 {
		t.Errorf("LastPct = %f, want 50.0", result.LastPct)
	}
	if result.TokensPerMin != 0 {
		t.Errorf("TokensPerMin = %f, want 0 (no previous data)", result.TokensPerMin)
	}
}

func TestUpdateBurnRate_ActiveIncrease(t *testing.T) {
	prev := burnState{
		TokensPerMin: 0,
		LastPct:      50.0,
		LastTime:     1000,
	}
	// 15 seconds later, 1% increase
	result := updateBurnRate(prev, 51.0)

	if result.TokensPerMin <= 0 {
		t.Error("TokensPerMin should be positive for active increase")
	}

	// Expected: (1% / 15s) * 60 * 5000 = 20000 t/m
	// With no previous TPM, no smoothing
	expected := ((1.0 / float64(result.LastTime-prev.LastTime)) * 60.0) * tokensPerPct
	diff := result.TokensPerMin - expected
	if diff < -1 || diff > 1 {
		t.Errorf("TokensPerMin = %f, want ≈%f", result.TokensPerMin, expected)
	}
}

func TestUpdateBurnRate_Decay(t *testing.T) {
	prev := burnState{
		TokensPerMin: 10000,
		LastPct:      50.0,
		LastTime:     1000,
	}
	// No change in percentage → decay
	result := updateBurnRate(prev, 50.0)

	expected := 10000.0 * 0.5
	if result.TokensPerMin != expected {
		t.Errorf("TokensPerMin = %f, want %f (50%% decay)", result.TokensPerMin, expected)
	}
}

func TestUpdateBurnRate_Smoothing(t *testing.T) {
	prev := burnState{
		TokensPerMin: 10000,
		LastPct:      50.0,
		LastTime:     1000,
	}

	// Simulate 15s delta with 2% increase
	result := updateBurnRate(prev, 52.0)

	// Should use 80/20 smoothing
	if result.TokensPerMin <= 0 {
		t.Error("TokensPerMin should be positive")
	}

	// Verify it's between raw new value and pure old value
	deltaSecs := float64(result.LastTime - prev.LastTime)
	rawNew := (2.0 / deltaSecs) * 60.0 * tokensPerPct
	smoothed := rawNew*0.8 + 10000*0.2

	diff := result.TokensPerMin - smoothed
	if diff < -1 || diff > 1 {
		t.Errorf("TokensPerMin = %f, want ≈%f (smoothed)", result.TokensPerMin, smoothed)
	}
}

func TestUpdateBurnRate_NegativeDelta(t *testing.T) {
	prev := burnState{
		TokensPerMin: 10000,
		LastPct:      50.0,
		LastTime:     1000,
	}
	// Decrease in percentage (reset)
	result := updateBurnRate(prev, 40.0)

	// Should decay
	if result.TokensPerMin != 5000 {
		t.Errorf("TokensPerMin = %f, want 5000 (decay on negative delta)", result.TokensPerMin)
	}
}
