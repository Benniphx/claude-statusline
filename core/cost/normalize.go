package cost

import "github.com/Benniphx/claude-statusline/core/types"

// CostMultiplier returns the cost weight for a given model tier.
func CostMultiplier(tier types.CostTier, cfg types.Config) float64 {
	switch tier {
	case types.CostTierOpus:
		return cfg.CostWeightOpus
	case types.CostTierHaiku:
		return cfg.CostWeightHaiku
	default:
		return cfg.CostWeightSonnet
	}
}

// IsNonBaseline returns true if the tier is not the Sonnet baseline.
func IsNonBaseline(tier types.CostTier) bool {
	return tier != types.CostTierSonnet
}

// NormalizeBurnRate applies cost normalization to a burn rate (tokens/min).
// Returns the normalized value and whether normalization was applied.
func NormalizeBurnRate(tpm float64, tier types.CostTier, cfg types.Config) (float64, bool) {
	if !cfg.CostNormalize {
		return tpm, false
	}
	multiplier := CostMultiplier(tier, cfg)
	return tpm * multiplier, IsNonBaseline(tier)
}

// NormalizePace applies cost normalization to a pace value.
// Returns the normalized value and whether normalization was applied.
func NormalizePace(pace float64, tier types.CostTier, cfg types.Config) (float64, bool) {
	if !cfg.CostNormalize {
		return pace, false
	}
	multiplier := CostMultiplier(tier, cfg)
	return pace * multiplier, IsNonBaseline(tier)
}

// NormalizeCostPerHour applies cost normalization to cost/hour.
// Returns the normalized value and whether normalization was applied.
func NormalizeCostPerHour(cph float64, tier types.CostTier, cfg types.Config) (float64, bool) {
	if !cfg.CostNormalize {
		return cph, false
	}
	multiplier := CostMultiplier(tier, cfg)
	return cph * multiplier, IsNonBaseline(tier)
}
