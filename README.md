# Claude Code Statusline

A rich status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that shows context usage, rate limits, and cost tracking.

![Version](https://img.shields.io/badge/version-1.0.1-blue)
![Platform](https://img.shields.io/badge/platform-macOS-lightgrey)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

### Subscription Mode
![Subscription Mode](./screenshots/subscription.png)

- **Context Window**: Progress bar + percentage + tokens used/max
- **5h Rate Limit**: Usage + time until reset + local reset time (rounded to 5min)
- **Burn Rate**: Tokens/min with warning if limit will be hit before reset
- **7d Rate Limit**: Usage + days until reset + reset date

### API-Key Mode
![API-Key Mode](./screenshots/api-key.png)

- **Context Window**: Same as above
- **Session Cost**: Current session spending
- **Daily Cost**: Cumulative daily spending across all sessions
- **Burn Rate**: Tokens/min + $/hour

### Color Coding
- 🟢 **Green**: < 50% (all good)
- 🟡 **Yellow**: 50-80% (attention)
- 🔴 **Red**: > 80% (critical)

### Warning System
Warnings only appear when there's a problem:
- `🔥 16K t/m ⚠️ 45m @12:40` - Will hit limit BEFORE reset
- `7d: 60% ⚠️` - Usage exceeds expected rate

## Requirements

- macOS (uses `security` command for keychain access)
- `jq` for JSON parsing
- `curl` for API calls

```bash
brew install jq
```

## Installation

### Quick Install
```bash
curl -fsSL https://raw.githubusercontent.com/Benniphx/claude-statusline/main/install.sh | bash
```

### Manual Install
```bash
# Download the script
curl -o ~/.claude/statusline.sh https://raw.githubusercontent.com/Benniphx/claude-statusline/main/statusline.sh
chmod +x ~/.claude/statusline.sh

# Add to Claude Code settings (~/.claude/settings.json)
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
```

## Configuration

The script auto-detects your account type:
- **Subscription (OAuth)**: Shows rate limits from Anthropic API
- **API-Key**: Shows cost tracking

No configuration needed - just install and restart Claude Code.

## Cache Files

The script uses temporary cache files to minimize API calls:
- `/tmp/claude_rate_limit_cache.json` - Rate limit data (60s TTL)
- `/tmp/claude_display_cache.json` - Fallback display values
- `/tmp/claude_daily_cost_YYYY-MM-DD.txt` - Daily cost tracking

Cache is automatically invalidated when rate limit resets.

## Troubleshooting

### No rate limit data showing
```bash
# Check if OAuth credentials exist
security find-generic-password -s "Claude Code-credentials" -w | jq '.claudeAiOauth'

# Clear cache and retry
rm -f /tmp/claude_rate_limit_cache.json /tmp/claude_display_cache.json
```

### Wrong timezone
The script converts UTC timestamps from the API to your local timezone automatically.

## License

MIT License - feel free to use and modify.

## Contributing

Issues and PRs welcome!
