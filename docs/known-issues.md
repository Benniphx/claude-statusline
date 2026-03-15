# Known Issues

## `api/oauth/usage` rate limiting (mitigated since v4.3.0)

**Status:** Mitigated — solved by using `User-Agent: claude-code/<version>` header + exponential backoff
**Tracking:** [anthropics/claude-code#30930](https://github.com/anthropics/claude-code/issues/30930), [#31021](https://github.com/anthropics/claude-code/issues/31021)

### Background

The `/api/oauth/usage` endpoint uses different rate limit buckets depending on the `User-Agent` header. Without the correct UA, the endpoint returns HTTP 429 after just 1-2 requests, with bans lasting 7-15+ minutes.

### What we found (March 2026)

| User-Agent | Behavior |
|---|---|
| Default (curl) | 429 after 1-2 requests, 7-15min ban |
| `claude-code/<version>` | Stable at 30s/45s/60s intervals (tested 5 requests each) |

Polling during an active ban **extends the ban duration**. This is why exponential backoff is critical.

### How v4.3.x handles it

1. **`User-Agent: claude-code/<version>`** header on all API requests → uses the generous rate limit bucket
2. **60s cache TTL** → only 1 API request per minute per machine
3. **Exponential backoff on 429** → 30s → 60s → 120s → max 5min, persisted to `/tmp/claude_statusline_backoff.json` and shared across all instances
4. **Stale cache fallback** → if API is unavailable, last successful data is displayed
5. **`STATUSLINE_NO_POLL=1`** → emergency kill-switch to disable all API calls

### Multi-machine usage

Rate limits are per OAuth token (= per account), not per machine. With the default 60s TTL, 2 machines create at most 1 request per 30 seconds — well within tested safe limits.

### If you still see frozen values

```bash
# Check backoff state
cat /tmp/claude_statusline_backoff.json
# If "until" is in the future, backoff is active

# Clear backoff manually
rm /tmp/claude_statusline_backoff.json

# Check cache freshness
stat -f "%Sm" /tmp/claude_rate_limit_cache.json

# Emergency: disable all polling
export STATUSLINE_NO_POLL=1
```

### Long-term fix

The ideal solution is for Anthropic to include rate limit data in the statusLine stdin JSON, eliminating the need for separate API calls entirely. Tracked in [anthropics/claude-code#27829](https://github.com/anthropics/claude-code/issues/27829) and [#29604](https://github.com/anthropics/claude-code/issues/29604).
