---
description: Check statusline configuration and troubleshoot issues
---

# Statusline Config

Check the statusline installation and configuration. Run diagnostics to identify any issues.

## Diagnostics to perform:

1. **Check if statusline.sh exists** at `~/.claude/statusline.sh`
2. **Check if settings.json** has the `statusLine` configuration
3. **Check dependencies**: `jq` and `curl` must be installed
4. **Check credentials access**:
   - macOS: Can read from Keychain
   - Linux: Can read from `~/.claude/.credentials.json`
5. **Test API connectivity** to Anthropic rate limit endpoint

## Output format:

Show a clear status for each check with pass/fail indicators.
If any check fails, provide the fix command.
