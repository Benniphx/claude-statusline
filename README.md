# Claude Code Statusline

> [!WARNING]
> **5h/7d values currently frozen (Anthropic-side bug)**
> The `api/oauth/usage` endpoint returns HTTP 429 persistently since ~Feb 2026.
> This affects all Claude Code statusline tools — not specific to this plugin.
> See [Known Issues](docs/known-issues.md) · [Upstream #30930](https://github.com/anthropics/claude-code/issues/30930)

![Stable](https://img.shields.io/badge/stable-v4.0.0-blue)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)
![License](https://img.shields.io/badge/license-MIT-green)
[![Tests](https://github.com/Benniphx/claude-statusline/actions/workflows/test.yml/badge.svg)](https://github.com/Benniphx/claude-statusline/actions/workflows/test.yml)

Rich status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) showing context usage, rate limits, and costs at a glance.

## Features

- **Context Window** - Progress bar with percentage and token count (color-coded: green → yellow → red)
- **5h Rate Limit** - Usage + pace indicator + smart time display
- **7d Rate Limit** - Usage + pace (work-day aware) + time display
- **Pace Indicator** - Shows if you're on track (`1.0x`) or burning fast (`2.0x`)
- **Agent Count** - Shows number of active Claude processes when running with subagents
- **Ollama Savings Tracker** - Tracks local model usage and shows estimated Haiku-equivalent cost savings
- **Stale Context Detection** - Ignores stale API percentages after clear/compact
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
Opus 4.6 (3)  │  Ctx: ██░░░░░░ 30% (60K/200K)  │  5h: ███░░░░░ 46% 0.5x →47m  │  🔥 5.0K t/m  │  7d: ██░░░░░░ 27% 0.6x  │  12m │ +142/-38  │  🦙 saved ~$1.71 (387 req · 3.3M tok)
```

| Section | Description |
|---------|-------------|
| `Opus 4.6` | Model name, color-coded by context usage (green <50%, yellow 50-80%, red >80%) |
| `(3)` | Active Claude processes — only shown when >1 (teams/subagents) |
| `Ctx: ██░░░░░░ 30% (60K/200K)` | Context window: progress bar + percentage + tokens used/total |
| `5h: ███░░░░░ 46% 0.5x →47m` | 5-hour rate limit: usage, pace (0.5x = half speed), time until reset |
| `🔥 5.0K t/m` | Burn rate: tokens per minute (current consumption speed) |
| `7d: ██░░░░░░ 27% 0.6x` | 7-day rate limit: weekly usage + pace |
| `12m` | Session duration in minutes |
| `+142/-38` | Lines added (green) / removed (red) in this session |
| `🦙 saved ~$1.71 (387 req · 3.3M tok)` | Ollama savings: estimated Haiku-equivalent cost saved by running locally |

**Pace details:**
| Example | Meaning |
|---------|---------|
| `5h: ████░░░░ 40% 0.8x` | Under budget (sustainable) |
| `5h: ██████░░ 72% 1.3x` | 30% over sustainable pace |
| `5h: ████████ 95% 2.1x` | Will hit limit before reset |
| `5h: ██████░░ 85% 1.3x →45m` | Reset in 45min (shown when ≤1h) |
| `5h: ███████░ 90% 1.5x →12m @14:30` | Reset at 14:30 (shown when ≤30m) |

### API-Key Mode

```
Opus 4.6  │  Ctx: ████░░░░ 45% (90K/200K)  │  💰 $0.42  │  📅 $3.85  │  🔥 8.2K t/m $1.20/h  │  12m │ +142/-38
```

| Section | Description |
|---------|-------------|
| `💰 $0.42` | Session cost (green <$0.50, yellow <$2.00, red >$2.00) |
| `📅 $3.85` | Daily cost across all sessions today (green <$5, yellow <$20, red >$20) |
| `🔥 8.2K t/m $1.20/h` | Burn rate: tokens/min + cost per hour (green <$1/h, yellow <$5/h, red >$5/h) |

### Local Models

Ollama/local models display with the llama icon and detected context size:

```
🦙 Qwen3  │  Ctx: █░░░░░░░ 15% (3K/32K)  │  ...
```

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

# ─────────────────────────────────────────────────────────
# Background Daemon (opt-in)
# ─────────────────────────────────────────────────────────
# Keeps rate limit cache fresh in the background
# Useful if you have multiple Claude sessions open
# Starts automatically on session start, stops when idle
# Default: false (disabled)
ENABLE_DAEMON=true
EOF
```

---

## Background Daemon (Details)

The optional background daemon keeps your rate limit cache fresh across all Claude sessions. Here's exactly what it does:

### What the Daemon Does

1. **Refreshes rate limit data** every 15 seconds from the Anthropic API
2. **Writes to a shared cache file** (`/tmp/claude_rate_limit_cache.json`)
3. **Calculates global burn rate** by tracking 5h% changes over time
4. **Auto-stops** after ~60 seconds of no Claude processes running

### What the Daemon Does NOT Do

- ❌ Does not send any data to external servers (only reads from Anthropic API)
- ❌ Does not track your conversations or prompts
- ❌ Does not run when Claude Code is not running
- ❌ Does not use significant CPU or memory (~0.1% CPU when active)

### Why Use It?

| Scenario | Without Daemon | With Daemon |
|----------|----------------|-------------|
| Multiple tabs | Each tab refreshes independently | Shared cache, less API calls |
| Rate limit accuracy | May be stale (up to 15s) | Always fresh |
| Burn rate display | Only shows when actively typing | Shows even during pauses |

### Technical Details

```
Location:     /tmp/claude_statusline_daemon.pid
Log file:     /tmp/claude_statusline_daemon.log
Cache file:   /tmp/claude_rate_limit_cache.json
Refresh:      Every 15 seconds
Idle timeout: 60 seconds (4 checks × 15s)
```

### Future Plans

The daemon is currently **opt-in** (`ENABLE_DAEMON=false` by default). We may change this to **opt-out** in a future version once it has been thoroughly tested by the community. Any such change will be clearly documented in the changelog.

---

## Ollama / Local Models

This statusline supports local models running via [Ollama](https://ollama.com/). When you use Ollama models with Claude Code, the statusline will:

1. **Display the model name** with a 🦙 icon (e.g., `🦙 Qwen3`, `🦙 Llama3`)
2. **Auto-detect your ACTUAL context size** (not just max capacity):
   - If model is running: reads your configured `num_ctx` from `/api/ps`
   - If not running: shows max capacity from `/api/show`
3. **Cache the context size** for 30 seconds (short cache since config can change)
4. **Track savings** when routing tasks locally instead of using the Haiku API

### Requirements

- [Ollama](https://ollama.com/) installed and running
- Model pulled (e.g., `ollama pull qwen3-coder:30b`)

### Using Ollama with Claude Code

To use local models with Claude Code, you can integrate Ollama via MCP:

- **[ollama-mcp](https://github.com/ollama/ollama-mcp)** - Official Ollama MCP server
- **[Claude Code MCP Docs](https://docs.anthropic.com/en/docs/claude-code/mcp)** - How to configure MCP servers

### Supported Models

The statusline recognizes and shortens common model names:

| Ollama Model | Displayed As |
|--------------|--------------|
| `qwen3-coder:30b` | `🦙 Qwen3` |
| `llama3:8b` | `🦙 Llama3` |
| `codellama:13b` | `🦙 CodeLlama` |
| `mistral:7b` | `🦙 Mistral` |
| `deepseek-coder:6.7b` | `🦙 DeepSeek` |
| Other models | `🦙 <ModelName>` |

### Ollama Savings Tracker

When you run tasks locally via Ollama instead of the Haiku API, the statusline tracks usage and shows estimated savings:

```
🦙 saved ~$1.71 (387 req · 3.3M tok)
```

**How it works:**
- Your Ollama agent writes stats to `/tmp/claude_ollama_stats.json` after each API call
- The statusline reads this file and calculates what those tokens would have cost using Haiku pricing ($0.25/1M input, $1.25/1M output)
- Stats are only shown when the file exists and is fresh (< 5 minutes old)

**Stats file format:**
```json
{
  "requests": 387,
  "total_prompt_tokens": 2400000,
  "total_completion_tokens": 890000,
  "last_updated": 1739367600
}
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

## Known Issues

### 5h/7d values frozen (since ~Feb 2026)

The `api/oauth/usage` endpoint returns persistent HTTP 429. This is a server-side issue at Anthropic — not a bug here. The last fetched values remain visible until Anthropic fixes it.

See [`docs/known-issues.md`](docs/known-issues.md) for details and a fixture test to detect when it's resolved. Upstream: [anthropics/claude-code#30930](https://github.com/anthropics/claude-code/issues/30930)

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

## Architecture (v4.0.0)

v4.0.0 is a complete rewrite from Bash to Go with hexagonal architecture:

```
core/                    Domain logic (pure, no I/O)
  context/               Context window calculations
  ratelimit/             Rate limits, burn rate, pace
  cost/                  Session cost tracking
  agents/                Claude process counting
  ollama/                Ollama stats reader + savings calculation
  model/                 Model detection + Ollama context
  update/                Update check
  daemon/                Background daemon
adapter/                 Implementations
  api/                   HTTP client for Anthropic + GitHub APIs
  cache/                 File-based cache with TTL
  config/                Config file parsing
  platform/              OS-specific (macOS/Linux) process detection
  render/                ANSI color output + progress bars
cmd/statusline/          Entry point
```

## What's New in v4.0.0

- **Go rewrite** — Full statusline rewritten in Go for speed and maintainability
- **Agent count display** — Shows active Claude process count when running subagents: `Opus 4.6 (3)`
- **Ollama savings tracker** — Tracks local model usage and calculates Haiku-equivalent cost savings: `🦙 saved ~$1.71 (387 req · 3.3M tok)`
- **Stale context fix** — Detects and ignores stale API percentages after context clear/compact
- **Hooks on more events** — Now triggers on `startup|resume|clear|compact` (was only startup)
- All v3.x features preserved: pace indicators, work-day aware 7d, Ollama model display, daemon, cross-tab sync

**Feedback:** [Open an issue](https://github.com/Benniphx/claude-statusline/issues).

---

## License

MIT
