# Changelog

All notable changes to this project will be documented in this file.

## [2.0.2] - 2026-01-08

### Fixed
- SessionStart hook now includes matcher for reliable execution
- Added timeout to hook command

## [2.0.1] - 2026-01-08

### Added
- Marketplace manifest for plugin distribution via `claude plugin marketplace add`

## [2.0.0] - 2026-01-08

### Added
- **Plugin Support**: Install as Claude Code plugin for automatic updates
- **Cross-Platform**: Works on macOS and Linux/WSL
- **CI Pipeline**: GitHub Actions for automated testing on macOS and Linux
- **Test Suite**: Comprehensive tests for all major functionality
- `/statusline:config` command for troubleshooting

### Changed
- Restructured project as Claude Code plugin
- Moved scripts to `scripts/` directory
- Cross-platform credential access (Keychain on macOS, credentials file on Linux)
- Cross-platform `stat` and `date` commands

### Technical
- `get_credentials()` - unified credential access
- `get_file_mtime()` - cross-platform file modification time
- `parse_iso_date()` - cross-platform ISO date parsing
- `format_time()` - cross-platform time formatting
- `sed_inplace()` - cross-platform sed in-place edit

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
