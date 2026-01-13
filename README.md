# Claude Code Statusline

![Version](https://img.shields.io/badge/version-2.0.6-blue)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)
![License](https://img.shields.io/badge/license-MIT-green)
[![Tests](https://github.com/Benniphx/claude-statusline/actions/workflows/test.yml/badge.svg)](https://github.com/Benniphx/claude-statusline/actions/workflows/test.yml)

Rich status line for [Claude Code](https://code.claude.com) showing context, rate limits, and costs.

**Features:**
- Context window usage with progress bar
- 5h/7d rate limits with reset times (subscription)
- Session + daily cost tracking (API-key)
- Burn rate warnings when approaching limits
- Auto-update notifications
- Cross-platform (macOS + Linux/WSL)

![Subscription Mode](./screenshots/subscription.png)

## Installation

**Requires:** `jq` (`brew install jq` / `apt install jq`)

```bash
# Add marketplace
claude plugin marketplace add Benniphx/claude-statusline

# Install
claude plugin install statusline

# Update
claude plugin update statusline@claude-statusline
```

Restart Claude Code after install.

## Usage

Works automatically after install. Auto-detects account type:
- **Subscription**: Shows rate limits
- **API-Key**: Shows costs

Run `/statusline:config` to troubleshoot.

## Screenshots

### Subscription Mode
![Subscription](./screenshots/subscription.png)
- Context: `Ctx: ████░░░░ 45% (90K/200K)`
- 5h limit: `5h: ██████░░ 72% →1h23m @14:30`
- Burn rate: `🔥 12.5K t/m` (warning if hitting limit before reset)
- 7d limit: `7d: ███░░░░░ 35% →4d @17.01`

### API-Key Mode
![API-Key](./screenshots/api-key.png)
- Session cost: `💰 $0.42`
- Daily cost: `📅 $3.85`
- Burn rate: `🔥 8.2K t/m $1.20/h`

### Color Coding
- 🟢 Green: < 50%
- 🟡 Yellow: 50-80%
- 🔴 Red: > 80%

## Manual Installation

If you prefer not to use the plugin system:

```bash
# Download
curl -o ~/.claude/statusline.sh https://raw.githubusercontent.com/Benniphx/claude-statusline/main/scripts/statusline.sh
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

## Cache & Credentials

**Cache files** (in `/tmp/` or `$CLAUDE_CODE_TMPDIR`):
- `claude_rate_limit_cache.json` (60s TTL)
- `claude_display_cache.json`
- `claude_daily_cost_YYYY-MM-DD.txt`
- `claude_session_total_*.txt` (for `--resume` support)

**Credentials:**
| Platform | Location |
|----------|----------|
| macOS | Keychain |
| Linux | `~/.claude/.credentials.json` |
| Override | `$CLAUDE_CODE_OAUTH_TOKEN` |

## License

MIT
