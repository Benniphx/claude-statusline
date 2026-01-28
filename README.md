# Claude Code Statusline

![Stable](https://img.shields.io/badge/stable-v3.0.4-blue)
![Beta](https://img.shields.io/badge/beta-v3.1.0--beta.5-orange)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)
![License](https://img.shields.io/badge/license-MIT-green)
[![Tests](https://github.com/Benniphx/claude-statusline/actions/workflows/test.yml/badge.svg)](https://github.com/Benniphx/claude-statusline/actions/workflows/test.yml)

Rich status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) showing context usage, rate limits, and costs at a glance.

## Features

- **Context Window** - Progress bar with percentage and token count
- **5h Rate Limit** - Usage + pace indicator + smart time display
- **7d Rate Limit** - Usage + pace (work-day aware) + time display
- **Pace Indicator** - Shows if you're on track (`1.0x`) or burning fast (`2.0x`)
- **Cross-Tab Sync** - All sessions share rate limit data (10s refresh)
- **API-Key Mode** - Session + daily cost tracking with burn rate

---

## Quick Start

**Requires:** `jq` (`brew install jq` / `apt install jq`)

```bash
# Add marketplace & install
claude plugin marketplace add Benniphx/claude-statusline
claude plugin install statusline

# Restart Claude Code
```

---

## What You'll See

### Subscription Mode

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Claude 3.5  │  Ctx: ████░░░░ 45% (90K/200K)  │  5h: ██████░░ 72% 1.3x  │ ... │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Context Window:**
| Example | Meaning |
|---------|---------|
| `Ctx: ████░░░░ 45%` | 45% of context used |
| `Ctx: ██████░░ 75% ⚠️` | Warning at threshold (configurable) |

**5h Rate Limit (Pace = usage speed vs sustainable):**
| Example | Meaning |
|---------|---------|
| `5h: ████░░░░ 40% 0.8x` | 🟢 Under budget (sustainable) |
| `5h: ██████░░ 72% 1.3x` | 🟡 30% over sustainable pace |
| `5h: ████████ 95% 2.1x ⚠️` | 🔴 Will hit limit before reset |
| `5h: ██████░░ 85% 1.3x →45m` | Reset in 45min (shown when ≤1h) |
| `5h: ███████░ 90% 1.5x →12m @14:30` | Reset at 14:30 (shown when ≤30m) |

**7d Rate Limit (Work-day aware):**
| Example | Meaning |
|---------|---------|
| `7d: ███░░░░░ 35% 0.8x` | 🟢 On track for the week |
| `7d: █████░░░ 60% 1.4x ⚠️` | 🔴 Over budget, pace warning |
| `7d: ██████░░ 72% 0.9x →2d` | Reset in 2 days (shown only when ≤3d) |

### API-Key Mode

```
┌───────────────────────────────────────────────────────────────────────────────────┐
│ Claude 3.5  │  Ctx: ████░░░░ 45%  │  💰 $0.42  │  📅 $3.85  │  🔥 8.2K t/m $1.20/h │
└───────────────────────────────────────────────────────────────────────────────────┘
```

| Element | Meaning |
|---------|---------|
| `💰 $0.42` | Current session cost |
| `📅 $3.85` | Today's total (all sessions) |
| `🔥 8.2K t/m $1.20/h` | Burn rate: tokens/min + cost/hour |

---

## Understanding Pace

The **pace indicator** (`1.3x`) shows how fast you're consuming your limit compared to a sustainable rate.

| Pace | Color | Meaning |
|------|-------|---------|
| `0.5x` | 🟢 Green | Half the sustainable rate - very conservative |
| `1.0x` | 🟢 Green | Exactly sustainable - will use 100% by reset |
| `1.3x` | 🟡 Yellow | 30% faster than sustainable |
| `2.0x` | 🔴 Red | Double speed - will hit limit at 50% time |

**5h Pace:** Based on `usage% / hours_elapsed`. Sustainable = 20%/hour.

**7d Pace:** Based on work days (Mon-Fri by default). If you've used 40% after 2 work days, and have 3 work days left, that's `(40%/2) / (100%/5) = 1.0x`.

---

## Configuration

Create `~/.config/claude-statusline/config`:

```bash
mkdir -p ~/.config/claude-statusline
cat > ~/.config/claude-statusline/config << 'EOF'
# ─────────────────────────────────────────────────────────
# Context Warning Threshold
# ─────────────────────────────────────────────────────────
# Show ⚠️ when context usage exceeds this percentage
# Useful to get a heads-up before hitting context limits
# Range: 1-100, Default: disabled
CONTEXT_WARNING_THRESHOLD=75

# ─────────────────────────────────────────────────────────
# Rate Limit Cache TTL
# ─────────────────────────────────────────────────────────
# How often to refresh rate limit data from API (seconds)
# Lower = faster cross-tab sync, slightly more API calls
# Range: 10-120, Default: 15
RATE_CACHE_TTL=15

# ─────────────────────────────────────────────────────────
# Work Days Per Week
# ─────────────────────────────────────────────────────────
# Used for 7d pace calculation
# 5 = Monday-Friday (excludes weekends)
# 7 = All days (if you work weekends)
# Range: 1-7, Default: 5
WORK_DAYS_PER_WEEK=5
EOF
```

---

## Manual Installation

If you prefer not to use the plugin system:

```bash
# Download
curl -o ~/.claude/statusline.sh \
  https://raw.githubusercontent.com/Benniphx/claude-statusline/main/scripts/statusline.sh
chmod +x ~/.claude/statusline.sh
```

Add to `~/.claude/settings.json`:
```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
```

---

## Troubleshooting

Run `/statusline:config` in Claude Code to check:
- Account type detection (subscription vs API-key)
- Credential access
- Cache file locations
- Current rate limit data

**Cache files** (in `/tmp/` or `$CLAUDE_CODE_TMPDIR`):
- `claude_rate_limit_cache.json` - API data (shared across tabs)
- `claude_display_cache.json` - Display fallback
- `claude_daily_cost_YYYY-MM-DD.txt` - Daily cost tracking
- `claude_session_total_*.txt` - Per-session tracking

**Credentials:**
| Platform | Location |
|----------|----------|
| macOS | Keychain (`Claude Code-credentials`) |
| Linux | `~/.claude/.credentials.json` |
| Override | `$CLAUDE_CODE_OAUTH_TOKEN` env var |

---

## Beta Testing

Want to try new features before they're stable? Install the beta version:

```bash
# Install specific beta version
claude plugins install Benniphx/claude-statusline@v3.1.0-beta.5

# Or install latest from main (includes unreleased changes)
claude plugins install Benniphx/claude-statusline@main
```

**Current Beta (v3.1.0-beta.5):**
- Cross-session burn rate tracking
- Background daemon for sync
- XDG config path support (`~/.config/claude-statusline/config`)

**Feedback:** [Open an issue](https://github.com/Benniphx/claude-statusline/issues) with `[beta]` in the title.

---

## License

MIT
