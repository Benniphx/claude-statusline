# Known Issues

## `api/oauth/usage` returns persistent 429 (since ~Feb 2026)

**Status:** Open — no fix from Anthropic yet
**Tracking:** [anthropics/claude-code#30930](https://github.com/anthropics/claude-code/issues/30930), [#31021](https://github.com/anthropics/claude-code/issues/31021)

### Symptom

The 5h and 7d utilization percentages in the statusline freeze and stop updating.
The `/tmp/claude_rate_limit_cache.json` cache file becomes stale (hours or days old).

### Root Cause

`GET https://api.anthropic.com/api/oauth/usage` has a secondary rate limit of approximately
5 requests per window. Claude Code itself also calls this endpoint (for the built-in `/usage`
command), which exhausts the limit immediately. Any external tool calling the same endpoint
then receives HTTP 429 indefinitely.

This is **not a bug in this tool** — it is a server-side rate limit on Anthropic's end.

### How to verify

```bash
# Check cache age
stat -f "%Sm" /tmp/claude_rate_limit_cache.json

# Test the endpoint directly
TOKEN=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('claudeAiOauth',{}).get('accessToken',''))")
curl -s "https://api.anthropic.com/api/oauth/usage" \
  -H "Authorization: Bearer $TOKEN" \
  -H "anthropic-beta: oauth-2025-04-20"
# Returns: {"error":{"type":"rate_limit_error","message":"Rate limited. Please try again later."}}
```

### Current behavior

No workaround available. The last successfully fetched values remain visible (stale).
A TTL increase would not help — Claude Code itself exhausts the rate limit regardless of how often this tool calls the endpoint.

### Fix eta

Unknown. The correct long-term fix is for Anthropic to expose rate limit utilization data
in the statusLine stdin JSON (tracked in [anthropics/claude-code#27829](https://github.com/anthropics/claude-code/issues/27829)
and [#29604](https://github.com/anthropics/claude-code/issues/29604)).

### How to check if the issue is fixed

Run `make test-oauth-usage` or:

```bash
go test ./adapter/api/... -run TestOAuthUsageEndpointLive -v
```
