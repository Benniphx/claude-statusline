# Changelog

All notable changes to this project will be documented in this file.

## [1.0.1] - 2026-01-07

### Fixed
- Remaining time now matches rounded reset time (e.g., `→15m @14:00` instead of `→13m @14:00`)

## [1.0.0] - 2026-01-07

### Added
- Dual mode: Subscription (rate limits) vs API-Key (costs)
- Context window progress bar with color coding
- 5h rate limit: Usage + time until reset + local reset time (rounded to 5min)
- 7d rate limit: Usage + days until reset + reset date
- Burn rate: Tokens/min with warning if limit will be hit before reset
- Warning-only approach: Only show ⚠️ when hitting limits
- Auto-invalidate cache when reset time passes
- Display cache for stable values during API timeouts
- Update notification when new version available

### Technical
- UTC timezone parsing for API timestamps
- Context overflow cap at 100%
- Session ID tracking via PPID for correct daily cost
