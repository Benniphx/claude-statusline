#!/bin/bash
# Claude Code Statusline v3.1.0-beta.4
# https://github.com/Benniphx/claude-statusline
# Cross-platform support: macOS + Linux/WSL
VERSION="3.1.0-beta.4"

export LC_NUMERIC=C

# === Daemon Mode ===
# Start with: statusline.sh --daemon
# Keeps rate limit cache fresh for all sessions
if [ "$1" = "--daemon" ]; then
    DAEMON_LOCK="/tmp/claude_statusline_daemon.lock"
    DAEMON_PID="/tmp/claude_statusline_daemon.pid"
    DAEMON_LOG="/tmp/claude_statusline_daemon.log"

    # Check if already running
    if [ -f "$DAEMON_PID" ]; then
        OLD_PID=$(cat "$DAEMON_PID" 2>/dev/null)
        if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
            exit 0  # Already running
        fi
    fi

    # Daemonize
    (
        # Get actual subshell PID (portable - works in all bash/zsh)
        MY_PID=$(sh -c 'echo $PPID')
        echo "$MY_PID" > "$DAEMON_PID"
        trap "rm -f '$DAEMON_LOCK' '$DAEMON_PID'" EXIT
        touch "$DAEMON_LOCK"

        # Config
        CACHE_DIR="${CLAUDE_CODE_TMPDIR:-/tmp}"
        RATE_CACHE="$CACHE_DIR/claude_rate_limit_cache.json"
        REFRESH_INTERVAL=15
        IDLE_CHECKS=0
        MAX_IDLE_CHECKS=4  # Exit after 4 checks (~60s) with no Claude processes

        # OS detection
        IS_MACOS=false
        [[ "$OSTYPE" == "darwin"* ]] && IS_MACOS=true

        # Get credentials (same logic as main script)
        get_creds() {
            if [ -n "$CLAUDE_CODE_OAUTH_TOKEN" ]; then
                echo "{\"claudeAiOauth\":{\"accessToken\":\"$CLAUDE_CODE_OAUTH_TOKEN\"}}"
                return
            fi
            if $IS_MACOS; then
                security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null
                return
            fi
            local CREDS_FILE="$HOME/.claude/.credentials.json"
            [ -f "$CREDS_FILE" ] && cat "$CREDS_FILE" 2>/dev/null
        }

        echo "[$(date '+%H:%M:%S')] Daemon started (PID $MY_PID)" >> "$DAEMON_LOG"

        while true; do
            # Check for Claude processes
            if pgrep -f "[c]laude" > /dev/null 2>&1; then
                IDLE_CHECKS=0

                # Refresh rate limit cache
                CREDS=$(get_creds)
                TOKEN=$(echo "$CREDS" | jq -r '.claudeAiOauth.accessToken' 2>/dev/null)

                if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
                    RESULT=$(curl -s --max-time 3 "https://api.anthropic.com/api/oauth/usage" \
                        -H "Authorization: Bearer $TOKEN" \
                        -H "anthropic-beta: oauth-2025-04-20" \
                        -H "Content-Type: application/json" 2>/dev/null)

                    if [ -n "$RESULT" ] && echo "$RESULT" | jq -e '.five_hour' >/dev/null 2>&1; then
                        TEMP_FILE="${RATE_CACHE}.tmp.$MY_PID"
                        echo "$RESULT" > "$TEMP_FILE" 2>/dev/null
                        mv "$TEMP_FILE" "$RATE_CACHE" 2>/dev/null

                        # Calculate global burn rate from 5h% delta
                        CURRENT_5H=$(echo "$RESULT" | jq -r '.five_hour.utilization' 2>/dev/null)
                        CURRENT_TS=$(date +%s)
                        BURN_CACHE="$CACHE_DIR/claude_global_burn.json"
                        TOKENS_PER_MIN=0

                        if [ -f "$BURN_CACHE" ]; then
                            PREV_5H=$(jq -r '.five_hour_percent // 0' "$BURN_CACHE" 2>/dev/null)
                            PREV_TS=$(jq -r '.timestamp // 0' "$BURN_CACHE" 2>/dev/null)
                            PREV_TPM=$(jq -r '.tokens_per_min // 0' "$BURN_CACHE" 2>/dev/null)

                            if [ -n "$PREV_5H" ] && [ -n "$PREV_TS" ] && [ "$PREV_TS" -gt 0 ]; then
                                DELTA_PCT=$(awk "BEGIN {printf \"%.1f\", $CURRENT_5H - $PREV_5H}" 2>/dev/null)
                                DELTA_SECS=$((CURRENT_TS - PREV_TS))

                                if [ "$DELTA_PCT" = "0.0" ] || [ "$DELTA_PCT" = "0" ]; then
                                    # No change in 5h% = no activity = fast decay
                                    # Decay by 50% each cycle when idle
                                    TOKENS_PER_MIN=$(awk "BEGIN {printf \"%.0f\", $PREV_TPM * 0.5}" 2>/dev/null)
                                elif [ "$DELTA_SECS" -gt 5 ] && [ "$(awk "BEGIN {print ($DELTA_PCT > 0)}")" = "1" ]; then
                                    # Positive delta = activity
                                    # Convert %/sec to tokens/min (1% ≈ 5000 tokens)
                                    PCT_PER_MIN=$(awk "BEGIN {printf \"%.4f\", ($DELTA_PCT / $DELTA_SECS) * 60}" 2>/dev/null)
                                    TOKENS_PER_MIN=$(awk "BEGIN {printf \"%.0f\", $PCT_PER_MIN * 5000}" 2>/dev/null)

                                    # Light smoothing only when active (80% new, 20% old)
                                    if [ "${PREV_TPM:-0}" -gt 0 ] 2>/dev/null; then
                                        TOKENS_PER_MIN=$(awk "BEGIN {printf \"%.0f\", $TOKENS_PER_MIN * 0.8 + $PREV_TPM * 0.2}" 2>/dev/null)
                                    fi
                                fi
                            fi
                        fi

                        # Write burn rate cache
                        jq -n \
                            --arg fh "$CURRENT_5H" \
                            --arg ts "$CURRENT_TS" \
                            --arg tpm "${TOKENS_PER_MIN:-0}" \
                            '{five_hour_percent: ($fh|tonumber), timestamp: ($ts|tonumber), tokens_per_min: ($tpm|tonumber)}' \
                            > "$BURN_CACHE" 2>/dev/null

                        echo "[$(date '+%H:%M:%S')] Cache refreshed (5h: ${CURRENT_5H}%, burn: ${TOKENS_PER_MIN:-0} t/m)" >> "$DAEMON_LOG"
                    fi
                fi
            else
                IDLE_CHECKS=$((IDLE_CHECKS + 1))
                if [ "$IDLE_CHECKS" -ge "$MAX_IDLE_CHECKS" ]; then
                    echo "[$(date '+%H:%M:%S')] No Claude processes, exiting" >> "$DAEMON_LOG"
                    exit 0
                fi
            fi

            sleep "$REFRESH_INTERVAL"
        done
    ) &

    disown 2>/dev/null
    exit 0
fi

# === Normal Statusline Mode ===
input=$(cat)

# === User Configuration ===
# Config file locations (in order of priority):
#   1. $XDG_CONFIG_HOME/claude-statusline/config (or ~/.config/claude-statusline/config)
#   2. ~/.claude-statusline.conf (legacy fallback)
# Example content:
#   CONTEXT_WARNING_THRESHOLD=75
#   RATE_CACHE_TTL=30
#   WORK_DAYS_PER_WEEK=5
#
XDG_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/claude-statusline"
CONFIG_FILE=""
if [ -f "$XDG_CONFIG_DIR/config" ]; then
    CONFIG_FILE="$XDG_CONFIG_DIR/config"
elif [ -f "$HOME/.claude-statusline.conf" ]; then
    CONFIG_FILE="$HOME/.claude-statusline.conf"
fi
CONTEXT_WARNING_THRESHOLD=""  # Empty = disabled (uses default color scheme)
RATE_CACHE_TTL=15             # Seconds between API refreshes (10-120)
WORK_DAYS_PER_WEEK=5          # 5 = Mon-Fri, 7 = all days (for 7d pace calculation)

if [ -n "$CONFIG_FILE" ] && [ -f "$CONFIG_FILE" ]; then
    # Source config file (only specific variables for security)
    while IFS='=' read -r key value; do
        key=$(echo "$key" | tr -d '[:space:]')
        value=$(echo "$value" | tr -d '[:space:]')
        case "$key" in
            CONTEXT_WARNING_THRESHOLD)
                if [[ "$value" =~ ^[0-9]+$ ]] && [ "$value" -ge 1 ] && [ "$value" -le 100 ]; then
                    CONTEXT_WARNING_THRESHOLD="$value"
                fi
                ;;
            RATE_CACHE_TTL)
                if [[ "$value" =~ ^[0-9]+$ ]] && [ "$value" -ge 10 ] && [ "$value" -le 120 ]; then
                    RATE_CACHE_TTL="$value"
                fi
                ;;
            WORK_DAYS_PER_WEEK)
                if [[ "$value" =~ ^[0-9]+$ ]] && [ "$value" -ge 1 ] && [ "$value" -le 7 ]; then
                    WORK_DAYS_PER_WEEK="$value"
                fi
                ;;
        esac
    done < "$CONFIG_FILE"
fi

# Cache directory (respects CLAUDE_CODE_TMPDIR)
CACHE_DIR="${CLAUDE_CODE_TMPDIR:-/tmp}"

# === Cross-Platform Helpers ===

# Detect OS
IS_MACOS=false
[[ "$OSTYPE" == "darwin"* ]] && IS_MACOS=true

# Get file modification time (seconds since epoch)
get_file_mtime() {
    local file="$1"
    if $IS_MACOS; then
        stat -f %m "$file" 2>/dev/null || echo 0
    else
        stat -c %Y "$file" 2>/dev/null || echo 0
    fi
}

# Parse ISO date to epoch (UTC)
parse_iso_date() {
    local iso_date="$1"
    local date_part="${iso_date%%.*}"  # Remove fractional seconds
    date_part="${date_part%%+*}"       # Remove timezone if present

    if $IS_MACOS; then
        date -j -u -f "%Y-%m-%dT%H:%M:%S" "$date_part" +%s 2>/dev/null || echo 0
    else
        date -u -d "$date_part" +%s 2>/dev/null || echo 0
    fi
}

# Format epoch to local time
format_time() {
    local epoch="$1"
    local format="$2"
    if $IS_MACOS; then
        date -j -f "%s" "$epoch" "+$format" 2>/dev/null
    else
        date -d "@$epoch" "+$format" 2>/dev/null
    fi
}

# Get credentials (cross-platform)
get_credentials() {
    # 1. Environment variable (highest priority)
    if [ -n "$CLAUDE_CODE_OAUTH_TOKEN" ]; then
        echo "{\"claudeAiOauth\":{\"accessToken\":\"$CLAUDE_CODE_OAUTH_TOKEN\"}}"
        return
    fi

    # 2. macOS Keychain
    if $IS_MACOS; then
        security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null
        return
    fi

    # 3. Linux/WSL credentials file
    local creds_file="$HOME/.claude/.credentials.json"
    [ ! -f "$creds_file" ] && creds_file="$HOME/.claude/credentials.json"

    if [ -f "$creds_file" ]; then
        cat "$creds_file" 2>/dev/null
    fi
}

# sed in-place (cross-platform)
sed_inplace() {
    if $IS_MACOS; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# Count work days between two epochs (excludes weekends if WORK_DAYS_PER_WEEK=5)
count_work_days() {
    local start_epoch="$1"
    local end_epoch="$2"
    local total_days=$(( (end_epoch - start_epoch) / 86400 ))

    if [ "$WORK_DAYS_PER_WEEK" -eq 7 ]; then
        echo "$total_days"
        return
    fi

    # Count weekends in the range
    local work_days=0
    local current_epoch="$start_epoch"
    while [ "$current_epoch" -lt "$end_epoch" ]; do
        local dow
        if $IS_MACOS; then
            dow=$(date -j -f "%s" "$current_epoch" "+%u" 2>/dev/null)
        else
            dow=$(date -d "@$current_epoch" "+%u" 2>/dev/null)
        fi
        # %u: Monday=1, Sunday=7
        if [ "${dow:-1}" -le 5 ]; then
            work_days=$((work_days + 1))
        fi
        current_epoch=$((current_epoch + 86400))
    done
    echo "$work_days"
}

# Get stable session ID by walking process tree to find Claude Code
# Returns: "stable:<pid>" or "unstable:<ppid>"
get_stable_session_id() {
    # 1. Use CLAUDE_SESSION_ID if set (ideal case)
    if [ -n "$CLAUDE_SESSION_ID" ]; then
        echo "stable:$CLAUDE_SESSION_ID"
        return
    fi

    # 2. Walk process tree to find claude/node ancestor
    if [ -d /proc ]; then
        # Linux: walk up /proc
        local pid=$$
        while [ "$pid" -gt 1 ]; do
            local comm
            comm=$(cat /proc/"$pid"/comm 2>/dev/null)
            if [[ "$comm" == "claude" ]]; then
                echo "stable:$pid"
                return
            fi
            # Move to parent
            pid=$(awk '{print $4}' /proc/"$pid"/stat 2>/dev/null)
            [ -z "$pid" ] && break
        done
    elif $IS_MACOS; then
        # macOS: use ps to walk ancestry
        local pid=$$
        while [ "$pid" -gt 1 ]; do
            local comm
            comm=$(ps -p "$pid" -o comm= 2>/dev/null)
            if [[ "$comm" == "claude" ]]; then
                echo "stable:$pid"
                return
            fi
            # Move to parent
            pid=$(ps -p "$pid" -o ppid= 2>/dev/null | tr -d ' ')
            [ -z "$pid" ] && break
        done
    fi

    # 3. Fallback: unstable PPID
    echo "unstable:$PPID"
}

# === Update Check (once daily) ===
UPDATE_CACHE="$CACHE_DIR/claude_statusline_update.txt"
UPDATE_NOTICE=""

# Version comparison helper: returns 0 if v1 > v2, 1 otherwise
version_gt() {
    local v1="$1" v2="$2"
    # Simple comparison: if they're equal, return 1 (not greater)
    [ "$v1" = "$v2" ] && return 1
    # Sort versions and check if v1 comes second (meaning it's greater)
    local highest
    highest=$(printf '%s\n%s\n' "$v1" "$v2" | sort -V | tail -n1)
    [ "$highest" = "$v1" ]
}

check_update() {
    local LATEST
    LATEST=$(curl -s --max-time 2 "https://api.github.com/repos/Benniphx/claude-statusline/releases/latest" 2>/dev/null | jq -r '.tag_name // empty' 2>/dev/null)
    if [ -n "$LATEST" ]; then
        LATEST="${LATEST#v}"  # Remove 'v' prefix
        echo "$LATEST" > "$UPDATE_CACHE"
        # Only show update if remote version is NEWER than local
        if version_gt "$LATEST" "$VERSION"; then
            echo "1"  # Update available
        else
            echo "0"  # Up to date (or local is newer/equal)
        fi
    fi
}

# Check cache (24h TTL)
UPDATE_AVAILABLE=""
if [ -f "$UPDATE_CACHE" ]; then
    CACHE_AGE=$(( $(date +%s) - $(get_file_mtime "$UPDATE_CACHE") ))
    CACHED_VERSION=$(cat "$UPDATE_CACHE" 2>/dev/null)

    if [ "$CACHE_AGE" -gt 86400 ]; then
        # Cache expired, refresh
        UPDATE_AVAILABLE=$(check_update)
    elif [ "$CACHED_VERSION" != "$VERSION" ]; then
        # Cache version differs - check if cached is actually newer
        if version_gt "$CACHED_VERSION" "$VERSION"; then
            # Cached version is newer = update available
            UPDATE_AVAILABLE="1"
        else
            # User upgraded or versions are same prefix - refresh to be sure
            UPDATE_AVAILABLE=$(check_update)
        fi
    fi
    # If CACHED_VERSION == VERSION, no update needed (don't set UPDATE_AVAILABLE)
else
    (check_update > /dev/null 2>&1) &  # Background check on first run
fi

# === Account Type Detection ===
CREDS=$(get_credentials)
HAS_OAUTH=$(echo "$CREDS" | jq -r '.claudeAiOauth.accessToken // empty' 2>/dev/null)
IS_SUBSCRIPTION=false
if [ -n "$HAS_OAUTH" ] && [ "$HAS_OAUTH" != "null" ]; then
    IS_SUBSCRIPTION=true
fi

# === Extract Data ===
CONTEXT_SIZE=$(echo "$input" | jq -r '.context_window.context_window_size // 200000')
MODEL=$(echo "$input" | jq -r '.model.display_name // "Claude"')
COST=$(echo "$input" | jq -r '.cost.total_cost_usd // 0')
DURATION_MS=$(echo "$input" | jq -r '.cost.total_duration_ms // 0')
LINES_ADDED=$(echo "$input" | jq -r '.cost.total_lines_added // 0')
LINES_REMOVED=$(echo "$input" | jq -r '.cost.total_lines_removed // 0')

# Context tokens: use current_usage (actual context) not cumulative totals
# current_usage contains the actual tokens in the context window
CURRENT_INPUT=$(echo "$input" | jq -r '.context_window.current_usage.input_tokens // empty' 2>/dev/null)
CACHE_CREATE=$(echo "$input" | jq -r '.context_window.current_usage.cache_creation_input_tokens // 0' 2>/dev/null)
CACHE_READ=$(echo "$input" | jq -r '.context_window.current_usage.cache_read_input_tokens // 0' 2>/dev/null)

if [ -n "$CURRENT_INPUT" ] && [ "$CURRENT_INPUT" != "null" ]; then
    # Use current_usage for accurate context window display
    TOTAL_TOKENS=$((CURRENT_INPUT + CACHE_CREATE + CACHE_READ))
else
    # Fallback to cumulative (old behavior) if current_usage not available
    INPUT_TOKENS=$(echo "$input" | jq -r '.context_window.total_input_tokens // 0')
    OUTPUT_TOKENS=$(echo "$input" | jq -r '.context_window.total_output_tokens // 0')
    TOTAL_TOKENS=$((INPUT_TOKENS + OUTPUT_TOKENS))
fi

# === Calculations ===
# Cap tokens at context size for display
if [ "$TOTAL_TOKENS" -gt "$CONTEXT_SIZE" ]; then
    DISPLAY_TOKENS="$CONTEXT_SIZE"
else
    DISPLAY_TOKENS="$TOTAL_TOKENS"
fi

# Use API-provided percentage if available (v2.1.6+), otherwise calculate
CTX_PERCENT=$(echo "$input" | jq -r '.context_window.used_percentage // empty' 2>/dev/null)
if [ -n "$CTX_PERCENT" ] && [ "$CTX_PERCENT" != "null" ]; then
    PERCENT_USED=${CTX_PERCENT%.*}  # Convert to integer
    PERCENT_USED=${PERCENT_USED:-0}
else
    PERCENT_USED=$((DISPLAY_TOKENS * 100 / CONTEXT_SIZE))
fi
DURATION_MIN=$((DURATION_MS / 60000))

# Burn Rate
if [ "$DURATION_MS" -gt 60000 ]; then
    TOKENS_PER_MIN=$(awk "BEGIN {printf \"%.1f\", ($TOTAL_TOKENS / ($DURATION_MS / 60000))}" 2>/dev/null)
    COST_PER_HOUR=$(awk "BEGIN {printf \"%.2f\", ($COST / ($DURATION_MS / 3600000))}" 2>/dev/null)
else
    TOKENS_PER_MIN="--"
    COST_PER_HOUR="0.00"
fi

# === Colors ===
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
CYAN="\033[36m"
BLUE="\033[34m"
MAGENTA="\033[35m"
DIM="\033[2m"
RESET="\033[0m"

# Update Notice
UPDATE_NOTICE=""
[ "$UPDATE_AVAILABLE" = "1" ] && UPDATE_NOTICE=" ${YELLOW}[Update]${RESET}"

# === Progress Bar Function ===
make_bar() {
    local percent=${1:-0}
    local width=${2:-10}

    [ "$percent" -lt 0 ] && percent=0
    [ "$percent" -gt 100 ] && percent=100

    local filled=$((percent * width / 100))
    local empty=$((width - filled))

    local color
    if [ "$percent" -lt 50 ]; then
        color="$GREEN"
    elif [ "$percent" -lt 80 ]; then
        color="$YELLOW"
    else
        color="$RED"
    fi

    local bar="${color}"
    for ((i=0; i<filled; i++)); do
        bar+="█"
    done
    bar+="${DIM}"
    for ((i=0; i<empty; i++)); do
        bar+="░"
    done
    bar+="${RESET}"

    echo -e "$bar"
}

# Initial context fix: Don't show misleading "0%" at session start
# Show "--" if no meaningful data yet (0 tokens or very early in session)
SHOW_CTX_PERCENT=true
if [ "$TOTAL_TOKENS" -eq 0 ] || ([ "$PERCENT_USED" -eq 0 ] && [ "$DURATION_MIN" -eq 0 ]); then
    SHOW_CTX_PERCENT=false
fi

# Context Color
if [ "$PERCENT_USED" -lt 50 ]; then
    CTX_COLOR="$GREEN"
elif [ "$PERCENT_USED" -lt 80 ]; then
    CTX_COLOR="$YELLOW"
else
    CTX_COLOR="$RED"
fi

# Context Warning (configurable threshold)
CONTEXT_WARNING=""
if [ -n "$CONTEXT_WARNING_THRESHOLD" ] && [ "$PERCENT_USED" -ge "$CONTEXT_WARNING_THRESHOLD" ]; then
    CONTEXT_WARNING=" ⚠️"
fi

# Context percent display (handle initial "0%" case)
if [ "$SHOW_CTX_PERCENT" = true ]; then
    CTX_PERCENT_DISPLAY="${PERCENT_USED}%"
else
    CTX_PERCENT_DISPLAY="--"
fi

# Format tokens
if [ "$DISPLAY_TOKENS" -gt 1000 ]; then
    TOKENS_FMT=$(awk "BEGIN {printf \"%.1f\", $DISPLAY_TOKENS / 1000}")K
else
    TOKENS_FMT="$DISPLAY_TOKENS"
fi

if [ "$CONTEXT_SIZE" -gt 1000 ]; then
    MAX_FMT=$(awk "BEGIN {printf \"%.0f\", $CONTEXT_SIZE / 1000}")K
else
    MAX_FMT="$CONTEXT_SIZE"
fi

# Lines changed
if [ "$LINES_ADDED" -gt 0 ] || [ "$LINES_REMOVED" -gt 0 ]; then
    LINES_INFO=" ${DIM}│${RESET} ${GREEN}+${LINES_ADDED}${RESET}/${RED}-${LINES_REMOVED}${RESET}"
else
    LINES_INFO=""
fi

# Context Bar
CTX_BAR=$(make_bar "$PERCENT_USED" 8)

# === SUBSCRIPTION vs API-KEY ===
if [ "$IS_SUBSCRIPTION" = true ]; then
    # ==========================================
    # SUBSCRIPTION MODE - Rate Limits
    # ==========================================

    RATE_CACHE="$CACHE_DIR/claude_rate_limit_cache.json"
    DISPLAY_CACHE="$CACHE_DIR/claude_display_cache.json"

    # Atomic write: write to temp, then rename (prevents empty reads)
    fetch_rate_limits_atomic() {
        local TOKEN
        TOKEN=$(echo "$CREDS" | jq -r '.claudeAiOauth.accessToken' 2>/dev/null)
        if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
            local TEMP_FILE="${RATE_CACHE}.tmp.$$"
            local RESULT
            RESULT=$(curl -s --max-time 3 "https://api.anthropic.com/api/oauth/usage" \
                -H "Authorization: Bearer $TOKEN" \
                -H "anthropic-beta: oauth-2025-04-20" \
                -H "Content-Type: application/json" 2>/dev/null)
            if [ -n "$RESULT" ] && echo "$RESULT" | jq -e '.five_hour' >/dev/null 2>&1; then
                echo "$RESULT" > "$TEMP_FILE" 2>/dev/null
                mv "$TEMP_FILE" "$RATE_CACHE" 2>/dev/null
            fi
        fi
    }

    # Check cache - always read existing data first
    RATE_DATA=""
    CACHE_EXPIRED=false
    if [ -f "$RATE_CACHE" ]; then
        RATE_DATA=$(cat "$RATE_CACHE" 2>/dev/null)
        CACHE_MOD=$(get_file_mtime "$RATE_CACHE")
        NOW_SECS=$(date +%s)
        if [ $((NOW_SECS - CACHE_MOD)) -ge "$RATE_CACHE_TTL" ]; then
            CACHE_EXPIRED=true
        fi
    fi

    # Refresh in background if expired (but keep using old data)
    if [ "$CACHE_EXPIRED" = true ] || [ -z "$RATE_DATA" ]; then
        if [ -z "$RATE_DATA" ]; then
            # No data at all - fetch synchronously
            fetch_rate_limits_atomic
            RATE_DATA=$(cat "$RATE_CACHE" 2>/dev/null)
        else
            # Have old data - refresh in background, keep using old
            (fetch_rate_limits_atomic) &
        fi
    fi

    # Extract rate data
    FIVE_HOUR_PERCENT=$(echo "$RATE_DATA" | jq -r '.five_hour.utilization // empty' 2>/dev/null)
    FIVE_HOUR_RESET=$(echo "$RATE_DATA" | jq -r '.five_hour.resets_at // empty' 2>/dev/null)
    SEVEN_DAY_PERCENT=$(echo "$RATE_DATA" | jq -r '.seven_day.utilization // empty' 2>/dev/null)
    SEVEN_DAY_RESET=$(echo "$RATE_DATA" | jq -r '.seven_day.resets_at // empty' 2>/dev/null)

    # Check if reset time passed → invalidate cache
    if [ -n "$FIVE_HOUR_RESET" ] && [ "$FIVE_HOUR_RESET" != "null" ]; then
        RESET_CHECK_EPOCH=$(parse_iso_date "$FIVE_HOUR_RESET")
        NOW_CHECK=$(date +%s)
        if [ "$RESET_CHECK_EPOCH" -gt 0 ] && [ "$RESET_CHECK_EPOCH" -lt "$NOW_CHECK" ]; then
            rm -f "$RATE_CACHE" "$DISPLAY_CACHE" 2>/dev/null
            fetch_rate_limits_atomic
            RATE_DATA=$(cat "$RATE_CACHE" 2>/dev/null)
            FIVE_HOUR_PERCENT=$(echo "$RATE_DATA" | jq -r '.five_hour.utilization // empty' 2>/dev/null)
            FIVE_HOUR_RESET=$(echo "$RATE_DATA" | jq -r '.five_hour.resets_at // empty' 2>/dev/null)
            SEVEN_DAY_PERCENT=$(echo "$RATE_DATA" | jq -r '.seven_day.utilization // empty' 2>/dev/null)
            SEVEN_DAY_RESET=$(echo "$RATE_DATA" | jq -r '.seven_day.resets_at // empty' 2>/dev/null)
        fi
    fi

    # Fallback to display cache (only if fresh to prevent flash from stale data)
    if [ -z "$FIVE_HOUR_PERCENT" ] && [ -f "$DISPLAY_CACHE" ]; then
        DISPLAY_CACHE_AGE=$(( $(date +%s) - $(get_file_mtime "$DISPLAY_CACHE") ))
        if [ "$DISPLAY_CACHE_AGE" -lt "$RATE_CACHE_TTL" ]; then
            FIVE_HOUR_PERCENT=$(jq -r '.five_hour_percent // empty' "$DISPLAY_CACHE" 2>/dev/null)
            FIVE_HOUR_RESET=$(jq -r '.five_hour_reset // empty' "$DISPLAY_CACHE" 2>/dev/null)
            SEVEN_DAY_PERCENT=$(jq -r '.seven_day_percent // empty' "$DISPLAY_CACHE" 2>/dev/null)
            SEVEN_DAY_RESET=$(jq -r '.seven_day_reset // empty' "$DISPLAY_CACHE" 2>/dev/null)
        fi
        # If cache too old, we'll show "--" until fresh data arrives
    fi

    # Save to display cache
    if [ -n "$FIVE_HOUR_PERCENT" ]; then
        jq -n \
            --arg fh "$FIVE_HOUR_PERCENT" \
            --arg fhr "$FIVE_HOUR_RESET" \
            --arg sd "$SEVEN_DAY_PERCENT" \
            --arg sdr "$SEVEN_DAY_RESET" \
            '{five_hour_percent: $fh, five_hour_reset: $fhr, seven_day_percent: $sd, seven_day_reset: $sdr}' \
            > "$DISPLAY_CACHE" 2>/dev/null
    fi

    # 5h Reset time
    RESET_INFO=""
    RESET_TIME_LOCAL=""
    RESET_EPOCH=0
    RESET_EPOCH_ROUNDED=0
    if [ -n "$FIVE_HOUR_RESET" ] && [ "$FIVE_HOUR_RESET" != "null" ] && [ "$FIVE_HOUR_RESET" != "" ]; then
        RESET_EPOCH=$(parse_iso_date "$FIVE_HOUR_RESET")
        NOW_EPOCH=$(date +%s)
        if [ "$RESET_EPOCH" -gt "$NOW_EPOCH" ]; then
            # Round to 5 min
            RESET_EPOCH_ROUNDED=$(( ((RESET_EPOCH + 150) / 300) * 300 ))
            RESET_TIME_LOCAL=$(format_time "$RESET_EPOCH_ROUNDED" "%H:%M")
            # Remaining time
            REMAINING_SECS=$((RESET_EPOCH_ROUNDED - NOW_EPOCH))
            [ "$REMAINING_SECS" -lt 0 ] && REMAINING_SECS=0
            RESET_H=$((REMAINING_SECS / 3600))
            RESET_M=$(((REMAINING_SECS % 3600) / 60))
            if [ "$RESET_H" -gt 0 ]; then
                RESET_INFO="${RESET_H}h${RESET_M}m"
            else
                RESET_INFO="${RESET_M}m"
            fi
        fi
    fi

    # 7d Reset date (with time if ≤2 days remaining)
    SEVEN_DAY_RESET_DATE=""
    SEVEN_DAY_DAYS_LEFT=""
    if [ -n "$SEVEN_DAY_RESET" ] && [ "$SEVEN_DAY_RESET" != "null" ] && [ "$SEVEN_DAY_RESET" != "" ]; then
        SEVEN_DAY_EPOCH=$(parse_iso_date "$SEVEN_DAY_RESET")
        if [ "$SEVEN_DAY_EPOCH" -gt 0 ]; then
            NOW_EPOCH=$(date +%s)
            DAYS_LEFT=$(( (SEVEN_DAY_EPOCH - NOW_EPOCH) / 86400 ))
            if [ "$DAYS_LEFT" -gt 0 ]; then
                SEVEN_DAY_DAYS_LEFT="${DAYS_LEFT}d"
            fi
            # Show time when ≤2 days remaining (more precision needed)
            if [ "$DAYS_LEFT" -le 2 ]; then
                SEVEN_DAY_RESET_DATE=$(format_time "$SEVEN_DAY_EPOCH" "%d.%m %H:%M")
            else
                SEVEN_DAY_RESET_DATE=$(format_time "$SEVEN_DAY_EPOCH" "%d.%m")
            fi
        fi
    fi

    # Rate colors
    FIVE_HOUR_INT=${FIVE_HOUR_PERCENT%.*}
    FIVE_HOUR_INT=${FIVE_HOUR_INT:-0}
    if [ "$FIVE_HOUR_INT" -lt 50 ]; then
        RATE_5H_COLOR="$GREEN"
    elif [ "$FIVE_HOUR_INT" -lt 80 ]; then
        RATE_5H_COLOR="$YELLOW"
    else
        RATE_5H_COLOR="$RED"
    fi

    SEVEN_DAY_INT=${SEVEN_DAY_PERCENT%.*}
    SEVEN_DAY_INT=${SEVEN_DAY_INT:-0}
    if [ "$SEVEN_DAY_INT" -lt 50 ]; then
        RATE_7D_COLOR="$GREEN"
    elif [ "$SEVEN_DAY_INT" -lt 80 ]; then
        RATE_7D_COLOR="$YELLOW"
    else
        RATE_7D_COLOR="$RED"
    fi

    # Progress Bars
    RATE_5H_BAR=$(make_bar "${FIVE_HOUR_INT:-0}" 8)
    RATE_7D_BAR=$(make_bar "${SEVEN_DAY_INT:-0}" 8)

    # Calculate Pace and check if hitting limit before reset
    PACE_DISPLAY=""
    PACE_COLOR="$GREEN"
    HITTING_LIMIT=false
    REMAINING_SECS_UNTIL_RESET=0

    if [ -n "$FIVE_HOUR_PERCENT" ] && [ "$FIVE_HOUR_PERCENT" != "null" ]; then
        FIVE_HOUR_FMT=$(awk "BEGIN {printf \"%.0f\", $FIVE_HOUR_PERCENT}" 2>/dev/null)
        SEVEN_DAY_FMT=$(awk "BEGIN {printf \"%.0f\", $SEVEN_DAY_PERCENT}" 2>/dev/null)
        FIVE_HOUR_FLT=$(awk "BEGIN {print $FIVE_HOUR_PERCENT + 0}" 2>/dev/null)

        # Calculate %/h and Pace
        if [ "$FIVE_HOUR_FLT" != "0" ] && [ -n "$RESET_EPOCH" ] && [ "$RESET_EPOCH" -gt 0 ]; then
            NOW_EPOCH_CALC=$(date +%s)
            REMAINING_SECS_UNTIL_RESET=$((RESET_EPOCH - NOW_EPOCH_CALC))
            SECS_SINCE_START=$((18000 - REMAINING_SECS_UNTIL_RESET))  # 5h = 18000 sec

            if [ "$SECS_SINCE_START" -gt 60 ]; then
                HOURS_SINCE_START=$(awk "BEGIN {print $SECS_SINCE_START / 3600}" 2>/dev/null)
                PERCENT_PER_HOUR=$(awk "BEGIN {printf \"%.1f\", $FIVE_HOUR_FLT / $HOURS_SINCE_START}" 2>/dev/null)

                # Pace: 20%/h = sustainable (1.0x)
                PACE=$(awk "BEGIN {printf \"%.1f\", $PERCENT_PER_HOUR / 20}" 2>/dev/null)
                PACE_INT=$(awk "BEGIN {printf \"%.0f\", $PACE * 10}" 2>/dev/null)

                if [ "${PACE_INT:-0}" -lt 10 ]; then
                    PACE_COLOR="$GREEN"   # < 1.0x = sustainable
                elif [ "${PACE_INT:-0}" -lt 15 ]; then
                    PACE_COLOR="$YELLOW"  # 1.0-1.5x = elevated
                else
                    PACE_COLOR="$RED"     # > 1.5x = burning fast
                fi
                PACE_DISPLAY="${PACE}x"

                # Check if hitting limit before reset
                REMAINING_PERCENT=$(awk "BEGIN {print 100 - $FIVE_HOUR_FLT}" 2>/dev/null)
                if [ "$(awk "BEGIN {print ($REMAINING_PERCENT > 0 && $PERCENT_PER_HOUR > 0) ? 1 : 0}")" = "1" ]; then
                    RUNWAY_MIN=$(awk "BEGIN {printf \"%.0f\", ($REMAINING_PERCENT / $PERCENT_PER_HOUR) * 60}" 2>/dev/null)
                    RUNWAY_SECS=$((RUNWAY_MIN * 60))
                    if [ "$RUNWAY_SECS" -lt "$REMAINING_SECS_UNTIL_RESET" ]; then
                        HITTING_LIMIT=true
                    fi
                fi
            fi
        fi

        # Build 5h display: Percent + Pace + Time (dynamic based on reset proximity)
        # Format: "72% 1.3x" or "85% 1.3x →45m" or "92% 1.3x →12m @14:30"
        RATE_DISPLAY="${RATE_5H_COLOR}${FIVE_HOUR_FMT}%${RESET}"

        if [ -n "$PACE_DISPLAY" ]; then
            RATE_DISPLAY+=" ${PACE_COLOR}${PACE_DISPLAY}${RESET}"
        fi

        # Add warning if hitting limit before reset
        if [ "$HITTING_LIMIT" = true ]; then
            RATE_DISPLAY+=" ${RED}⚠️${RESET}"
        fi

        # Add time info based on proximity to reset
        if [ "$REMAINING_SECS_UNTIL_RESET" -gt 0 ]; then
            REMAINING_MIN=$((REMAINING_SECS_UNTIL_RESET / 60))
            if [ "$REMAINING_MIN" -le 30 ] && [ -n "$RESET_INFO" ] && [ -n "$RESET_TIME_LOCAL" ]; then
                # ≤30min: show both duration and time
                RATE_DISPLAY+=" ${DIM}→${RESET}${CYAN}${RESET_INFO}${RESET} ${DIM}@${RESET}${CYAN}${RESET_TIME_LOCAL}${RESET}"
            elif [ "$REMAINING_MIN" -le 60 ] && [ -n "$RESET_INFO" ]; then
                # ≤1h: show just duration
                RATE_DISPLAY+=" ${DIM}→${RESET}${CYAN}${RESET_INFO}${RESET}"
            fi
            # >1h: no time shown
        fi
    else
        RATE_DISPLAY="${DIM}--${RESET}"
        SEVEN_DAY_FMT="--"
        FIVE_HOUR_INT=0
        SEVEN_DAY_INT=0
    fi

    # 7d Pace calculation (based on work days)
    # Simpler approach: scale calendar days by work day ratio
    SEVEN_DAY_PACE=""
    SEVEN_DAY_PACE_COLOR="$GREEN"
    SEVEN_DAY_WARNING=""

    if [ -n "$SEVEN_DAY_FMT" ] && [ "$SEVEN_DAY_FMT" != "--" ] && [ "${SEVEN_DAY_FMT:-0}" -gt 0 ]; then
        DAYS_LEFT=${SEVEN_DAY_DAYS_LEFT%d}
        DAYS_LEFT=${DAYS_LEFT:-0}
        CALENDAR_DAYS_ELAPSED=$((7 - DAYS_LEFT))

        if [ "$CALENDAR_DAYS_ELAPSED" -gt 0 ]; then
            # Scale to work days: elapsed_work_days = calendar_days * (work_days_per_week / 7)
            WORK_DAYS_ELAPSED=$(awk "BEGIN {printf \"%.2f\", $CALENDAR_DAYS_ELAPSED * ($WORK_DAYS_PER_WEEK / 7)}" 2>/dev/null)

            # Sustainable rate = 100% / work_days_per_week
            # Actual rate = usage% / work_days_elapsed
            # Pace = actual / sustainable
            if [ "$(awk "BEGIN {print ($WORK_DAYS_ELAPSED > 0.1) ? 1 : 0}")" = "1" ]; then
                SUSTAINABLE_PER_DAY=$(awk "BEGIN {print 100 / $WORK_DAYS_PER_WEEK}" 2>/dev/null)
                ACTUAL_PER_DAY=$(awk "BEGIN {print $SEVEN_DAY_FMT / $WORK_DAYS_ELAPSED}" 2>/dev/null)
                SEVEN_DAY_PACE_VAL=$(awk "BEGIN {printf \"%.1f\", $ACTUAL_PER_DAY / $SUSTAINABLE_PER_DAY}" 2>/dev/null)

                PACE_7D_INT=$(awk "BEGIN {printf \"%.0f\", $SEVEN_DAY_PACE_VAL * 10}" 2>/dev/null)
                if [ "${PACE_7D_INT:-0}" -lt 10 ]; then
                    SEVEN_DAY_PACE_COLOR="$GREEN"
                elif [ "${PACE_7D_INT:-0}" -lt 15 ]; then
                    SEVEN_DAY_PACE_COLOR="$YELLOW"
                else
                    SEVEN_DAY_PACE_COLOR="$RED"
                fi
                SEVEN_DAY_PACE="${SEVEN_DAY_PACE_VAL}x"

                # Warning if pace > 1.0 (will exceed limit)
                if [ "${PACE_7D_INT:-0}" -gt 10 ]; then
                    SEVEN_DAY_WARNING=" ${RED}⚠️${RESET}"
                fi
            fi
        fi
    fi

    # Build 7d display: Percent + Pace + Days (only when ≤3d)
    SEVEN_DAY_DISPLAY="${RATE_7D_COLOR}${SEVEN_DAY_FMT:-0}%${RESET}"

    if [ -n "$SEVEN_DAY_PACE" ]; then
        SEVEN_DAY_DISPLAY+=" ${SEVEN_DAY_PACE_COLOR}${SEVEN_DAY_PACE}${RESET}"
    fi

    if [ -n "$SEVEN_DAY_WARNING" ]; then
        SEVEN_DAY_DISPLAY+="${SEVEN_DAY_WARNING}"
    fi

    # Show days only when ≤3d remaining
    if [ -n "$SEVEN_DAY_DAYS_LEFT" ]; then
        DAYS_LEFT_NUM=${SEVEN_DAY_DAYS_LEFT%d}
        if [ "${DAYS_LEFT_NUM:-7}" -le 3 ]; then
            SEVEN_DAY_DISPLAY+=" ${DIM}→${RESET}${CYAN}${SEVEN_DAY_DAYS_LEFT}${RESET}"
        fi
    fi

    # Burn rate display (global + local tokens/min)
    GLOBAL_BURN_CACHE="$CACHE_DIR/claude_global_burn.json"
    GLOBAL_TPM=0
    if [ -f "$GLOBAL_BURN_CACHE" ]; then
        GLOBAL_TPM=$(jq -r '.tokens_per_min // 0' "$GLOBAL_BURN_CACHE" 2>/dev/null)
    fi

    # Format local burn rate
    LOCAL_TPM_FMT="--"
    if [ "$TOKENS_PER_MIN" != "--" ]; then
        TPM_INT=${TOKENS_PER_MIN%.*}
        if [ "${TPM_INT:-0}" -gt 1000 ]; then
            LOCAL_TPM_FMT=$(awk "BEGIN {printf \"%.1f\", $TOKENS_PER_MIN / 1000}")K
        else
            LOCAL_TPM_FMT="${TOKENS_PER_MIN%.*}"
        fi
    fi

    # Format global burn rate
    GLOBAL_TPM_FMT="--"
    GLOBAL_TPM_INT=${GLOBAL_TPM%.*}
    if [ "${GLOBAL_TPM_INT:-0}" -gt 100 ]; then
        if [ "${GLOBAL_TPM_INT:-0}" -gt 1000 ]; then
            GLOBAL_TPM_FMT=$(awk "BEGIN {printf \"%.1f\", $GLOBAL_TPM / 1000}")K
        else
            GLOBAL_TPM_FMT="${GLOBAL_TPM_INT}"
        fi
    fi

    # Build burn display - always show local rate (accurate), add indicator if other sessions active
    # Global burn rate from 5h% is too coarse (batched API updates), so we only use it as activity indicator
    LOCAL_TPM_INT=${TOKENS_PER_MIN%.*}
    LOCAL_TPM_INT=${LOCAL_TPM_INT:-0}

    # Check if there's significant other activity (global > local + 5000)
    OTHER_ACTIVITY=false
    # Ensure LOCAL_TPM_INT is numeric (default to 0 if --)
    [[ "$LOCAL_TPM_INT" =~ ^[0-9]+$ ]] || LOCAL_TPM_INT=0
    if [ "${GLOBAL_TPM_INT:-0}" -gt 0 ] && [ "$LOCAL_TPM_INT" -gt 0 ]; then
        DIFF=$((GLOBAL_TPM_INT - LOCAL_TPM_INT))
        if [ "$DIFF" -gt 5000 ]; then
            OTHER_ACTIVITY=true
        fi
    elif [ "${GLOBAL_TPM_INT:-0}" -gt 5000 ] && [ "$LOCAL_TPM_INT" -eq 0 ]; then
        # No local activity but significant global = other sessions
        OTHER_ACTIVITY=true
    fi

    if [ "$LOCAL_TPM_FMT" != "--" ]; then
        if [ "$OTHER_ACTIVITY" = true ]; then
            # Show local rate with "others active" indicator
            BURN_DISPLAY="🔥 ${MAGENTA}${LOCAL_TPM_FMT}${RESET} ${DIM}t/m${RESET} ${YELLOW}⚡${RESET}"
        else
            BURN_DISPLAY="🔥 ${MAGENTA}${LOCAL_TPM_FMT}${RESET} ${DIM}t/m${RESET}"
        fi
    elif [ "$OTHER_ACTIVITY" = true ]; then
        # No local activity but others are active
        BURN_DISPLAY="🔥 ${DIM}--${RESET} ${YELLOW}⚡${RESET}"
    else
        BURN_DISPLAY="🔥 ${DIM}--${RESET}"
    fi

    # OUTPUT: Subscription
    echo -e "${CTX_COLOR}${MODEL}${RESET}  ${DIM}│${RESET}  Ctx: ${CTX_BAR} ${CTX_COLOR}${CTX_PERCENT_DISPLAY}${RESET}${CONTEXT_WARNING} ${DIM}(${TOKENS_FMT}/${MAX_FMT})${RESET}  ${DIM}│${RESET}  5h: ${RATE_5H_BAR} ${RATE_DISPLAY}  ${DIM}│${RESET}  ${BURN_DISPLAY}  ${DIM}│${RESET}  7d: ${RATE_7D_BAR} ${SEVEN_DAY_DISPLAY}  ${DIM}│${RESET}  ${DIM}${DURATION_MIN}m${RESET}${LINES_INFO}${UPDATE_NOTICE}"

else
    # ==========================================
    # API-KEY MODE - Costs
    # ==========================================

    TODAY=$(date +%Y-%m-%d)
    COST_TRACKER="$CACHE_DIR/claude_daily_cost_${TODAY}.txt"

    # Get session ID (stable if we found Claude process, unstable if using PPID)
    SESSION_ID_RESULT=$(get_stable_session_id)
    SESSION_ID_TYPE="${SESSION_ID_RESULT%%:*}"
    CURRENT_SESSION_ID="${SESSION_ID_RESULT#*:}"
    SESSION_TOTAL_FILE="$CACHE_DIR/claude_session_total_${CURRENT_SESSION_ID}.txt"

    # Clean old daily trackers (keep today only)
    find "$CACHE_DIR" -name "claude_daily_cost_*.txt" ! -name "claude_daily_cost_${TODAY}.txt" -delete 2>/dev/null

    # Session cost display = total session cost (unchanged)
    SESSION_COST="$COST"

    if [ "$SESSION_ID_TYPE" = "stable" ]; then
        # === STABLE ID: Use delta-based accounting (handles --resume) ===

        # Get last known total for this session
        LAST_KNOWN_TOTAL=0
        if [ -f "$SESSION_TOTAL_FILE" ]; then
            LAST_KNOWN_TOTAL=$(cat "$SESSION_TOTAL_FILE" 2>/dev/null)
            LAST_KNOWN_TOTAL=${LAST_KNOWN_TOTAL:-0}
        fi

        # Calculate delta (new spending since last update)
        COST_DELTA=$(awk "BEGIN {printf \"%.6f\", $COST - $LAST_KNOWN_TOTAL}" 2>/dev/null)
        # Ensure delta is not negative (can happen on session restart)
        if awk "BEGIN {exit ($COST_DELTA < 0) ? 0 : 1}" 2>/dev/null; then
            COST_DELTA="$COST"
        fi

        # Save current total for next comparison
        echo "$COST" > "$SESSION_TOTAL_FILE"

        # Update daily tracker with delta
        if [ -f "$COST_TRACKER" ]; then
            if grep -q "^${CURRENT_SESSION_ID}:" "$COST_TRACKER" 2>/dev/null; then
                CURRENT_DAILY=$(grep "^${CURRENT_SESSION_ID}:" "$COST_TRACKER" | cut -d: -f2)
                CURRENT_DAILY=${CURRENT_DAILY:-0}
                NEW_DAILY=$(awk "BEGIN {printf \"%.6f\", $CURRENT_DAILY + $COST_DELTA}" 2>/dev/null)
                sed_inplace "s/^${CURRENT_SESSION_ID}:.*/${CURRENT_SESSION_ID}:${NEW_DAILY}/" "$COST_TRACKER" 2>/dev/null
            else
                echo "${CURRENT_SESSION_ID}:${COST_DELTA}" >> "$COST_TRACKER"
            fi
        else
            echo "${CURRENT_SESSION_ID}:${COST_DELTA}" > "$COST_TRACKER"
        fi
    else
        # === UNSTABLE ID: Use replace-based accounting (safer, avoids inflation) ===
        # Since PPID changes frequently, we use the session cost directly.
        # This may slightly overcount with --resume across days, but avoids
        # the massive inflation bug from delta-based with unstable IDs.

        if [ -f "$COST_TRACKER" ]; then
            if grep -q "^${CURRENT_SESSION_ID}:" "$COST_TRACKER" 2>/dev/null; then
                # Update existing entry (replace, not add)
                sed_inplace "s/^${CURRENT_SESSION_ID}:.*/${CURRENT_SESSION_ID}:${COST}/" "$COST_TRACKER" 2>/dev/null
            else
                # New session entry
                echo "${CURRENT_SESSION_ID}:${COST}" >> "$COST_TRACKER"
            fi
        else
            echo "${CURRENT_SESSION_ID}:${COST}" > "$COST_TRACKER"
        fi
    fi

    DAILY_COST=$(awk -F: '{sum += $2} END {printf "%.4f", sum}' "$COST_TRACKER" 2>/dev/null)

    # Cost colors (session)
    COST_CENTS=$(awk "BEGIN {printf \"%.0f\", $SESSION_COST * 100}" 2>/dev/null)
    COST_CENTS=${COST_CENTS:-0}
    if [ "$COST_CENTS" -lt 50 ]; then
        COST_COLOR="$GREEN"
    elif [ "$COST_CENTS" -lt 200 ]; then
        COST_COLOR="$YELLOW"
    else
        COST_COLOR="$RED"
    fi

    # Cost colors (daily)
    DAILY_CENTS=$(awk "BEGIN {printf \"%.0f\", $DAILY_COST * 100}" 2>/dev/null)
    DAILY_CENTS=${DAILY_CENTS:-0}
    if [ "$DAILY_CENTS" -lt 500 ]; then
        DAILY_COLOR="$GREEN"
    elif [ "$DAILY_CENTS" -lt 2000 ]; then
        DAILY_COLOR="$YELLOW"
    else
        DAILY_COLOR="$RED"
    fi

    # Burn rate color
    CPH_CENTS=$(awk "BEGIN {printf \"%.0f\", $COST_PER_HOUR * 100}" 2>/dev/null)
    CPH_CENTS=${CPH_CENTS:-0}
    if [ "$CPH_CENTS" -lt 100 ]; then
        BURN_COLOR="$GREEN"
    elif [ "$CPH_CENTS" -lt 500 ]; then
        BURN_COLOR="$YELLOW"
    else
        BURN_COLOR="$RED"
    fi

    # Format
    SESSION_FMT=$(awk "BEGIN {printf \"%.2f\", $SESSION_COST}" 2>/dev/null)
    DAILY_FMT=$(awk "BEGIN {printf \"%.2f\", $DAILY_COST}" 2>/dev/null)

    # Burn rate display
    if [ "$TOKENS_PER_MIN" != "--" ]; then
        TPM_INT=${TOKENS_PER_MIN%.*}
        if [ "${TPM_INT:-0}" -gt 1000 ]; then
            TPM_FMT=$(awk "BEGIN {printf \"%.1f\", $TOKENS_PER_MIN / 1000}")K
        else
            TPM_FMT="${TOKENS_PER_MIN}"
        fi
        BURN_DISPLAY="${MAGENTA}${TPM_FMT}${RESET} ${DIM}t/m${RESET} ${BURN_COLOR}\$${COST_PER_HOUR}${RESET}${DIM}/h${RESET}"
    else
        BURN_DISPLAY="${DIM}--${RESET}"
    fi

    # OUTPUT: API-Key
    echo -e "${CTX_COLOR}${MODEL}${RESET}  ${DIM}│${RESET}  Ctx: ${CTX_BAR} ${CTX_COLOR}${CTX_PERCENT_DISPLAY}${RESET}${CONTEXT_WARNING} ${DIM}(${TOKENS_FMT}/${MAX_FMT})${RESET}  ${DIM}│${RESET}  💰 ${COST_COLOR}\$${SESSION_FMT}${RESET}  ${DIM}│${RESET}  📅 ${DAILY_COLOR}\$${DAILY_FMT}${RESET}  ${DIM}│${RESET}  🔥 ${BURN_DISPLAY}  ${DIM}│${RESET}  ${DIM}${DURATION_MIN}m${RESET}${LINES_INFO}${UPDATE_NOTICE}"
fi
