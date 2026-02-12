# Claude Code Statusline

![Stable](https://img.shields.io/badge/stable-v4.0.0-blue)
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

## What's New in v4.0.0

- **Ollama/Local model support** - Shows `🦙 Qwen3`, `🦙 Llama3`, etc. with auto-detected context size
- **Shorter model names** - `Opus 4.5` instead of long display names
- **Robust error handling** - No more crashes on startup
- **Background daemon** - Opt-in via `ENABLE_DAEMON=true` in config
- **Pace indicators** - See if you're on track (`1.0x`) or burning fast (`2.0x`)
- **Work-day aware 7d** - Excludes weekends from pace calculation

**Feedback:** [Open an issue](https://github.com/Benniphx/claude-statusline/issues) with `[beta]` in the title.

---

## License

MIT
