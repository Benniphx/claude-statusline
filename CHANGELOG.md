# Changelog

All notable changes to this project will be documented in this file.

## [3.0.3] - 2026-01-26

### Fixed
- **Cache invalidation bug**: Fixed undefined function call `fetch_rate_limits` → `fetch_rate_limits_atomic` when 5h reset time passes

## [3.0.2] - 2026-01-26

### Fixed
- **Initial context display**: No longer shows "0%" or "null" at session start when tokens not yet loaded

## [3.0.1] - 2026-01-26

### Added
- **Burn rate display** back between 5h and 7d: `🔥 12.5K t/m`

## [3.0.0] - 2026-01-26

### Added
- **Pace indicator** - Shows consumption speed vs sustainable rate (`1.0x` = on budget)
  - 5h: `72% 1.3x` means 30% faster than sustainable (20%/hour)
  - 7d: `35% 0.8x` means 20% under budget (work-day aware)
- **Work-day aware 7d calculation** - Excludes weekends by default
  - Configure with `WORK_DAYS_PER_WEEK=5` (Mon-Fri) or `7` (all days)
- **Smart time display** - Only shows when relevant
  - 5h: `→45m` when ≤1h, `→12m @14:30` when ≤30m
  - 7d: `→2d` only shown when ≤3 days remaining
- **Configurable context warning**: `CONTEXT_WARNING_THRESHOLD=75` shows ⚠️
- **Configurable cache TTL**: `RATE_CACHE_TTL=15` for fast cross-tab sync
- **Atomic cache writes** - Prevents empty statusline during refresh

### Changed
- **Completely redesigned rate limit display** - Cleaner, more informative
- Removed separate 🔥 burn rate section - integrated into 5h/7d displays
- More compact statusline with less visual clutter
- Improved README with tables and pace explanation

### Fixed
- Initial context no longer shows misleading "0%" at session start
- Cache race condition that caused empty statusline during refresh

## [2.0.7] - 2026-01-14

### Fixed
- **Session ID stability**: PPID could change between statusline invocations, causing each call to be treated as a new session and inflating daily costs by 70x+
- Now walks process tree to find the actual Claude process PID for stable session tracking
- Falls back to safer replace-based accounting when stable ID unavailable

### Credits
- Fix contributed by [@omarkohl](https://github.com/omarkohl) in [#2](https://github.com/Benniphx/claude-statusline/pull/2)

## [2.0.6] - 2026-01-13

### Fixed
- **Daily cost tracking with `--resume`**: Sessions resumed across multiple days no longer show inflated daily costs
- Previously, `--resume` sessions would display the entire accumulated session cost as "today's" cost
- Now tracks cost deltas per session, so daily cost only reflects spending since midnight
- Example: A 3-day resumed session with $30 total now correctly shows ~$10/day instead of $30 on day 3

### Technical
- Added `SESSION_TOTAL_FILE` to track last known total cost per session
- Daily tracker now stores delta-based costs instead of total session costs
- Handles negative deltas gracefully (session restart detection)

## [2.0.5] - 2026-01-13

### Added
- Use `context_window.used_percentage` from Claude Code v2.1.6+ API when available
- Respect `CLAUDE_CODE_TMPDIR` environment variable for all cache files
- Version consistency check now includes README badge

### Fixed
- Display cache TTL reduced to 60s to prevent flash of stale rate limit data on session start

## [2.0.4] - 2026-01-08

### Fixed
- Context window now shows **actual usage** instead of cumulative tokens
- Previously showed 200K/200K after long sessions even when context was only ~40% full
- Now uses `current_usage.input_tokens` from Claude Code API for accurate display

## [2.0.3] - 2026-01-08

### Fixed
- False `[Update]` notification after upgrading: now uses proper version comparison
- Added `version_gt()` helper for semantic version comparison
- Update check now correctly handles: user upgrades, equal versions, and actual new releases

### Added
- Tests for update notification logic (version comparison, no false positives)

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
