#!/bin/bash
# Claude Code Statusline v2.0.3
# https://github.com/Benniphx/claude-statusline
# Cross-platform support: macOS + Linux/WSL
VERSION="2.0.3"

export LC_NUMERIC=C
input=$(cat)

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

# === Update Check (once daily) ===
UPDATE_CACHE="/tmp/claude_statusline_update.txt"
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
INPUT_TOKENS=$(echo "$input" | jq -r '.context_window.total_input_tokens // 0')
OUTPUT_TOKENS=$(echo "$input" | jq -r '.context_window.total_output_tokens // 0')
CONTEXT_SIZE=$(echo "$input" | jq -r '.context_window.context_window_size // 200000')
MODEL=$(echo "$input" | jq -r '.model.display_name // "Claude"')
COST=$(echo "$input" | jq -r '.cost.total_cost_usd // 0')
DURATION_MS=$(echo "$input" | jq -r '.cost.total_duration_ms // 0')
LINES_ADDED=$(echo "$input" | jq -r '.cost.total_lines_added // 0')
LINES_REMOVED=$(echo "$input" | jq -r '.cost.total_lines_removed // 0')

# === Calculations ===
TOTAL_TOKENS=$((INPUT_TOKENS + OUTPUT_TOKENS))
# Cap tokens at context size for display (API sometimes reports more)
if [ "$TOTAL_TOKENS" -gt "$CONTEXT_SIZE" ]; then
    DISPLAY_TOKENS="$CONTEXT_SIZE"
else
    DISPLAY_TOKENS="$TOTAL_TOKENS"
fi
PERCENT_USED=$((DISPLAY_TOKENS * 100 / CONTEXT_SIZE))
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

# Context Color
if [ "$PERCENT_USED" -lt 50 ]; then
    CTX_COLOR="$GREEN"
elif [ "$PERCENT_USED" -lt 80 ]; then
    CTX_COLOR="$YELLOW"
else
    CTX_COLOR="$RED"
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

    RATE_CACHE="/tmp/claude_rate_limit_cache.json"
    DISPLAY_CACHE="/tmp/claude_display_cache.json"
    RATE_CACHE_AGE=60

    fetch_rate_limits() {
        local TOKEN
        TOKEN=$(echo "$CREDS" | jq -r '.claudeAiOauth.accessToken' 2>/dev/null)
        if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
            curl -s --max-time 3 "https://api.anthropic.com/api/oauth/usage" \
                -H "Authorization: Bearer $TOKEN" \
                -H "anthropic-beta: oauth-2025-04-20" \
                -H "Content-Type: application/json" 2>/dev/null
        fi
    }

    # Check cache
    RATE_DATA=""
    if [ -f "$RATE_CACHE" ]; then
        CACHE_MOD=$(get_file_mtime "$RATE_CACHE")
        NOW_SECS=$(date +%s)
        if [ $((NOW_SECS - CACHE_MOD)) -lt "$RATE_CACHE_AGE" ]; then
            RATE_DATA=$(cat "$RATE_CACHE" 2>/dev/null)
        fi
    fi

    if [ -z "$RATE_DATA" ]; then
        if [ ! -f "$RATE_CACHE" ]; then
            RATE_DATA=$(fetch_rate_limits)
            echo "$RATE_DATA" > "$RATE_CACHE" 2>/dev/null
        else
            (fetch_rate_limits > "$RATE_CACHE" 2>/dev/null) &
            RATE_DATA=$(cat "$RATE_CACHE" 2>/dev/null)
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
            RATE_DATA=$(fetch_rate_limits)
            echo "$RATE_DATA" > "$RATE_CACHE" 2>/dev/null
            FIVE_HOUR_PERCENT=$(echo "$RATE_DATA" | jq -r '.five_hour.utilization // empty' 2>/dev/null)
            FIVE_HOUR_RESET=$(echo "$RATE_DATA" | jq -r '.five_hour.resets_at // empty' 2>/dev/null)
            SEVEN_DAY_PERCENT=$(echo "$RATE_DATA" | jq -r '.seven_day.utilization // empty' 2>/dev/null)
            SEVEN_DAY_RESET=$(echo "$RATE_DATA" | jq -r '.seven_day.resets_at // empty' 2>/dev/null)
        fi
    fi

    # Fallback to display cache
    if [ -z "$FIVE_HOUR_PERCENT" ] && [ -f "$DISPLAY_CACHE" ]; then
        FIVE_HOUR_PERCENT=$(jq -r '.five_hour_percent // empty' "$DISPLAY_CACHE" 2>/dev/null)
        FIVE_HOUR_RESET=$(jq -r '.five_hour_reset // empty' "$DISPLAY_CACHE" 2>/dev/null)
        SEVEN_DAY_PERCENT=$(jq -r '.seven_day_percent // empty' "$DISPLAY_CACHE" 2>/dev/null)
        SEVEN_DAY_RESET=$(jq -r '.seven_day_reset // empty' "$DISPLAY_CACHE" 2>/dev/null)
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

    # 7d Reset date
    SEVEN_DAY_RESET_DATE=""
    SEVEN_DAY_DAYS_LEFT=""
    if [ -n "$SEVEN_DAY_RESET" ] && [ "$SEVEN_DAY_RESET" != "null" ] && [ "$SEVEN_DAY_RESET" != "" ]; then
        SEVEN_DAY_EPOCH=$(parse_iso_date "$SEVEN_DAY_RESET")
        if [ "$SEVEN_DAY_EPOCH" -gt 0 ]; then
            SEVEN_DAY_RESET_DATE=$(format_time "$SEVEN_DAY_EPOCH" "%d.%m")
            NOW_EPOCH=$(date +%s)
            DAYS_LEFT=$(( (SEVEN_DAY_EPOCH - NOW_EPOCH) / 86400 ))
            if [ "$DAYS_LEFT" -gt 0 ]; then
                SEVEN_DAY_DAYS_LEFT="${DAYS_LEFT}d"
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

    # Format rate display
    if [ -n "$FIVE_HOUR_PERCENT" ] && [ "$FIVE_HOUR_PERCENT" != "null" ]; then
        FIVE_HOUR_FMT=$(awk "BEGIN {printf \"%.0f\", $FIVE_HOUR_PERCENT}" 2>/dev/null)
        SEVEN_DAY_FMT=$(awk "BEGIN {printf \"%.0f\", $SEVEN_DAY_PERCENT}" 2>/dev/null)
        if [ -n "$RESET_INFO" ] && [ -n "$RESET_TIME_LOCAL" ]; then
            RATE_DISPLAY="${RATE_5H_COLOR}${FIVE_HOUR_FMT}%${RESET} ${DIM}→${RESET}${BLUE}${RESET_INFO}${RESET} ${DIM}@${RESET}${CYAN}${RESET_TIME_LOCAL}${RESET}"
        elif [ -n "$RESET_INFO" ]; then
            RATE_DISPLAY="${RATE_5H_COLOR}${FIVE_HOUR_FMT}%${RESET} ${DIM}→${RESET}${BLUE}${RESET_INFO}${RESET}"
        else
            RATE_DISPLAY="${RATE_5H_COLOR}${FIVE_HOUR_FMT}%${RESET}"
        fi
    else
        RATE_DISPLAY="${DIM}--${RESET}"
        SEVEN_DAY_FMT="--"
        FIVE_HOUR_INT=0
        SEVEN_DAY_INT=0
    fi

    # Progress Bars
    RATE_5H_BAR=$(make_bar "${FIVE_HOUR_INT:-0}" 8)
    RATE_7D_BAR=$(make_bar "${SEVEN_DAY_INT:-0}" 8)

    # Burn Rate + ETA
    ETA_DISPLAY=""
    if [ "$TOKENS_PER_MIN" != "--" ] && [ -n "$FIVE_HOUR_PERCENT" ] && [ "$FIVE_HOUR_PERCENT" != "null" ]; then
        # Format tokens/min
        TPM_INT=${TOKENS_PER_MIN%.*}
        if [ "${TPM_INT:-0}" -gt 1000 ]; then
            TPM_FMT=$(awk "BEGIN {printf \"%.1f\", $TOKENS_PER_MIN / 1000}")K
        else
            TPM_FMT="${TOKENS_PER_MIN}"
        fi

        # Calculate %/h
        FIVE_HOUR_FLT=$(awk "BEGIN {print $FIVE_HOUR_PERCENT + 0}" 2>/dev/null)
        PERCENT_PER_HOUR="--"
        if [ "$FIVE_HOUR_FLT" != "0" ] && [ -n "$RESET_EPOCH" ] && [ "$RESET_EPOCH" -gt 0 ]; then
            NOW_EPOCH_CALC=$(date +%s)
            SECS_UNTIL_RESET=$((RESET_EPOCH - NOW_EPOCH_CALC))
            SECS_SINCE_START=$((18000 - SECS_UNTIL_RESET))  # 5h = 18000 sec
            if [ "$SECS_SINCE_START" -gt 60 ]; then
                HOURS_SINCE_START=$(awk "BEGIN {print $SECS_SINCE_START / 3600}" 2>/dev/null)
                PERCENT_PER_HOUR=$(awk "BEGIN {printf \"%.0f\", $FIVE_HOUR_FLT / $HOURS_SINCE_START}" 2>/dev/null)
            fi
        fi

        # ETA calculation
        REMAINING_PERCENT=$(awk "BEGIN {print 100 - $FIVE_HOUR_FLT}" 2>/dev/null)
        ETA_TIME=""
        ETA_COLOR="$CYAN"
        if [ "$(awk "BEGIN {print ($REMAINING_PERCENT > 0 && $PERCENT_PER_HOUR > 0) ? 1 : 0}")" = "1" ] && [ "$PERCENT_PER_HOUR" != "--" ]; then
            ETA_MIN=$(awk "BEGIN {printf \"%.0f\", ($REMAINING_PERCENT / $PERCENT_PER_HOUR) * 60}" 2>/dev/null)
            if [ "${ETA_MIN:-0}" -gt 0 ]; then
                NOW_EPOCH=$(date +%s)
                ETA_EPOCH=$((NOW_EPOCH + ETA_MIN * 60))
                ETA_MIN_RAW=$(format_time "$ETA_EPOCH" "%M")
                ETA_MIN_ROUNDED=$(( ((ETA_MIN_RAW + 2) / 5) * 5 ))
                if [ "$ETA_MIN_ROUNDED" -eq 60 ]; then
                    ETA_TIME=$(format_time "$((ETA_EPOCH + 300))" "%H:00")
                else
                    ETA_HOUR=$(format_time "$ETA_EPOCH" "%H")
                    ETA_TIME=$(printf "%s:%02d" "$ETA_HOUR" "$ETA_MIN_ROUNDED")
                fi
                # Color: Red if before reset, Green if after
                if [ -n "$RESET_EPOCH" ] && [ "$RESET_EPOCH" -gt 0 ]; then
                    if [ "$ETA_EPOCH" -lt "$RESET_EPOCH" ]; then
                        ETA_COLOR="$RED"
                    else
                        ETA_COLOR="$GREEN"
                    fi
                fi
            fi
        fi

        # Format burn display
        if [ -n "$ETA_TIME" ] && [ "${ETA_MIN:-0}" -gt 0 ] && [ "$ETA_COLOR" = "$RED" ]; then
            if [ "$ETA_MIN" -ge 60 ]; then
                ETA_DUR_H=$((ETA_MIN / 60))
                ETA_DUR_M=$((ETA_MIN % 60))
                ETA_DURATION="${ETA_DUR_H}h${ETA_DUR_M}m"
            else
                ETA_DURATION="${ETA_MIN}m"
            fi
            BURN_DISPLAY="${MAGENTA}${TPM_FMT}${RESET} ${DIM}t/m${RESET} ${RED}⚠️ ${ETA_DURATION} @${ETA_TIME}${RESET}"
        else
            BURN_DISPLAY="${MAGENTA}${TPM_FMT}${RESET} ${DIM}t/m${RESET}"
        fi
    else
        BURN_DISPLAY="${DIM}--${RESET}"
    fi

    # 7d warning
    SEVEN_DAY_WARNING=""
    if [ -n "$SEVEN_DAY_DAYS_LEFT" ] && [ -n "$SEVEN_DAY_FMT" ] && [ "$SEVEN_DAY_FMT" != "--" ]; then
        DAYS_ELAPSED=$((7 - ${SEVEN_DAY_DAYS_LEFT%d}))
        if [ "$DAYS_ELAPSED" -gt 0 ]; then
            EXPECTED_MAX=$(awk "BEGIN {printf \"%.0f\", ($DAYS_ELAPSED / 7) * 100}" 2>/dev/null)
            if [ "${SEVEN_DAY_FMT:-0}" -gt "${EXPECTED_MAX:-100}" ]; then
                SEVEN_DAY_WARNING=" ${RED}⚠️${RESET}"
            fi
        fi
    fi

    if [ -n "$SEVEN_DAY_RESET_DATE" ] && [ -n "$SEVEN_DAY_DAYS_LEFT" ]; then
        SEVEN_DAY_DISPLAY="${RATE_7D_COLOR}${SEVEN_DAY_FMT:-0}%${RESET}${SEVEN_DAY_WARNING} ${DIM}→${RESET}${CYAN}${SEVEN_DAY_DAYS_LEFT}${RESET} ${DIM}@${RESET}${CYAN}${SEVEN_DAY_RESET_DATE}${RESET}"
    elif [ -n "$SEVEN_DAY_RESET_DATE" ]; then
        SEVEN_DAY_DISPLAY="${RATE_7D_COLOR}${SEVEN_DAY_FMT:-0}%${RESET}${SEVEN_DAY_WARNING} ${DIM}→${RESET}${CYAN}${SEVEN_DAY_RESET_DATE}${RESET}"
    else
        SEVEN_DAY_DISPLAY="${RATE_7D_COLOR}${SEVEN_DAY_FMT:-0}%${RESET}${SEVEN_DAY_WARNING}"
    fi

    # OUTPUT: Subscription
    echo -e "${CTX_COLOR}${MODEL}${RESET}  ${DIM}│${RESET}  Ctx: ${CTX_BAR} ${CTX_COLOR}${PERCENT_USED}%${RESET} ${DIM}(${TOKENS_FMT}/${MAX_FMT})${RESET}  ${DIM}│${RESET}  5h: ${RATE_5H_BAR} ${RATE_DISPLAY}  ${DIM}│${RESET}  🔥 ${BURN_DISPLAY}  ${DIM}│${RESET}  7d: ${RATE_7D_BAR} ${SEVEN_DAY_DISPLAY}  ${DIM}│${RESET}  ${DIM}${DURATION_MIN}m${RESET}${LINES_INFO}${UPDATE_NOTICE}"

else
    # ==========================================
    # API-KEY MODE - Costs
    # ==========================================

    TODAY=$(date +%Y-%m-%d)
    COST_TRACKER="/tmp/claude_daily_cost_${TODAY}.txt"

    # Clean old trackers
    find /tmp -name "claude_daily_cost_*.txt" ! -name "claude_daily_cost_${TODAY}.txt" -delete 2>/dev/null

    SESSION_COST="$COST"

    # Load or init daily costs
    if [ -f "$COST_TRACKER" ]; then
        CURRENT_SESSION_ID="${CLAUDE_SESSION_ID:-$PPID}"

        if grep -q "^${CURRENT_SESSION_ID}:" "$COST_TRACKER" 2>/dev/null; then
            sed_inplace "s/^${CURRENT_SESSION_ID}:.*/${CURRENT_SESSION_ID}:${SESSION_COST}/" "$COST_TRACKER" 2>/dev/null
        else
            echo "${CURRENT_SESSION_ID}:${SESSION_COST}" >> "$COST_TRACKER"
        fi

        DAILY_COST=$(awk -F: '{sum += $2} END {printf "%.4f", sum}' "$COST_TRACKER" 2>/dev/null)
    else
        CURRENT_SESSION_ID="${CLAUDE_SESSION_ID:-$PPID}"
        echo "${CURRENT_SESSION_ID}:${SESSION_COST}" > "$COST_TRACKER"
        DAILY_COST="$SESSION_COST"
    fi

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
    echo -e "${CTX_COLOR}${MODEL}${RESET}  ${DIM}│${RESET}  Ctx: ${CTX_BAR} ${CTX_COLOR}${PERCENT_USED}%${RESET} ${DIM}(${TOKENS_FMT}/${MAX_FMT})${RESET}  ${DIM}│${RESET}  💰 ${COST_COLOR}\$${SESSION_FMT}${RESET}  ${DIM}│${RESET}  📅 ${DAILY_COLOR}\$${DAILY_FMT}${RESET}  ${DIM}│${RESET}  🔥 ${BURN_DISPLAY}  ${DIM}│${RESET}  ${DIM}${DURATION_MIN}m${RESET}${LINES_INFO}${UPDATE_NOTICE}"
fi
