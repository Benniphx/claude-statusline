# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0] - 2026-01-07

### Added
- Warning-only approach: Only show ⚠️ when hitting limits
- 5h rate limit: Local reset time (rounded to 5min)
- 7d rate limit: Days until reset + date
- Auto-invalidate cache when reset time passes
- Display cache for stable values during API timeouts

### Fixed
- UTC timezone parsing for API timestamps
- Context overflow: Cap display at 100%
- Session ID tracking: Use PPID for correct daily cost

### Changed
- Reordered sections: Ctx | 5h | 🔥 Burn | 7d | Duration
- Burn rate shows warning only when limit before reset

## [1.0.0] - 2025-12-17

### Added
- Initial release
- Dual mode: Subscription (rate limits) vs API-Key (costs)
- Context window progress bar
- Color-coded usage indicators
- Session and daily cost tracking
