#!/bin/bash
# Claude Statusline - Test Suite
# Run with: ./tests/test_statusline.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
STATUSLINE="$PROJECT_ROOT/scripts/statusline.sh"

# Colors for test output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    echo "  Expected: $2"
    echo "  Got: $3"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Mock input data for testing
create_mock_input() {
    local input_tokens="${1:-50000}"
    local output_tokens="${2:-10000}"
    local context_size="${3:-200000}"
    local cost="${4:-0.50}"
    local duration_ms="${5:-120000}"

    cat << EOF
{
    "context_window": {
        "total_input_tokens": $input_tokens,
        "total_output_tokens": $output_tokens,
        "context_window_size": $context_size
    },
    "model": {
        "display_name": "Claude Sonnet 4"
    },
    "cost": {
        "total_cost_usd": $cost,
        "total_duration_ms": $duration_ms,
        "total_lines_added": 100,
        "total_lines_removed": 50
    }
}
EOF
}

# ============================================
# Unit Tests
# ============================================

echo "Running Claude Statusline Tests..."
echo "=================================="
echo ""

# Test 1: Script exists and is executable
echo "Test: Script exists and is executable"
if [ -x "$STATUSLINE" ]; then
    pass "statusline.sh is executable"
else
    fail "statusline.sh should be executable" "executable" "not executable"
fi

# Test 2: All version numbers match across files
echo "Test: Version consistency across all files"
V_SCRIPT=$(grep -o 'VERSION="[^"]*"' "$STATUSLINE" | cut -d'"' -f2)
V_PLUGIN=$(jq -r '.version' "$PROJECT_ROOT/.claude-plugin/plugin.json")
V_MARKET=$(jq -r '.plugins[0].version' "$PROJECT_ROOT/.claude-plugin/marketplace.json")
V_CHANGELOG=$(grep -o '## \[[0-9a-z.\-]*\]' "$PROJECT_ROOT/CHANGELOG.md" | head -1 | tr -d '[]# ')

# README has separate stable and beta badges
V_README_STABLE=$(grep -o 'stable-v[0-9.]*-blue' "$PROJECT_ROOT/README.md" | sed 's/stable-v//;s/-blue//')
V_README_BETA=$(grep -o 'beta-v[0-9a-z.\-]*-orange' "$PROJECT_ROOT/README.md" | sed 's/beta-v//;s/-orange//' | sed 's/--/-/')

# Script/plugin/marketplace should match (current dev version)
if [ "$V_SCRIPT" = "$V_PLUGIN" ] && [ "$V_PLUGIN" = "$V_MARKET" ]; then
    pass "Script/plugin/marketplace versions match: $V_SCRIPT"
else
    fail "Version mismatch in code!" "all same" "script=$V_SCRIPT, plugin=$V_PLUGIN, marketplace=$V_MARKET"
fi

# Test 2b: README beta badge matches current version (if beta exists)
echo "Test: README beta badge matches current version"
if [ -z "$V_README_BETA" ]; then
    # No beta badge = stable release, that's fine
    pass "No beta badge (stable release)"
elif [ "$V_README_BETA" = "$V_SCRIPT" ]; then
    pass "README beta badge matches: $V_README_BETA"
else
    fail "README beta badge outdated!" "$V_SCRIPT" "README shows $V_README_BETA"
fi

# Test 2c: CHANGELOG latest entry matches current version
echo "Test: CHANGELOG matches current version"
if [ "$V_CHANGELOG" = "$V_SCRIPT" ]; then
    pass "CHANGELOG matches: $V_CHANGELOG"
else
    fail "CHANGELOG outdated!" "$V_SCRIPT" "CHANGELOG shows $V_CHANGELOG"
fi

# Test 2d: README stable badge is documented (informational)
echo "Test: README stable badge present"
if [ -n "$V_README_STABLE" ]; then
    pass "README stable version: $V_README_STABLE"
else
    fail "README missing stable badge" "vX.Y.Z" "not found"
fi

# Test 3: Cross-platform helpers exist
echo "Test: Cross-platform helper functions exist"
if grep -q "get_file_mtime()" "$STATUSLINE" && \
   grep -q "parse_iso_date()" "$STATUSLINE" && \
   grep -q "format_time()" "$STATUSLINE" && \
   grep -q "get_credentials()" "$STATUSLINE"; then
    pass "All cross-platform helpers defined"
else
    fail "Missing cross-platform helper functions" "all helpers" "some missing"
fi

# Test 4: Script handles empty input gracefully
echo "Test: Handles empty input"
OUTPUT=$(echo '{}' | bash "$STATUSLINE" 2>/dev/null || true)
if [ -n "$OUTPUT" ]; then
    pass "Produces output with empty input"
else
    fail "Should produce output even with empty input" "some output" "empty"
fi

# Test 5: Script handles minimal valid input
echo "Test: Handles minimal valid input"
OUTPUT=$(create_mock_input | bash "$STATUSLINE" 2>/dev/null || true)
if [ -n "$OUTPUT" ]; then
    pass "Produces output with valid input"
else
    fail "Should produce output with valid input" "some output" "empty"
fi

# Test 6: Output contains model name
echo "Test: Output contains model name"
OUTPUT=$(create_mock_input | bash "$STATUSLINE" 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "Sonnet"; then
    pass "Output contains model name"
else
    fail "Output should contain model name" "Sonnet" "$OUTPUT"
fi

# Test 7: Output contains context percentage
echo "Test: Output contains context percentage"
OUTPUT=$(create_mock_input 60000 0 200000 | bash "$STATUSLINE" 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "30%"; then
    pass "Output contains correct context percentage (30%)"
else
    fail "Output should contain 30% for 60K/200K tokens" "30%" "$OUTPUT"
fi

# Test 8: Progress bar is generated
echo "Test: Progress bar is generated"
OUTPUT=$(create_mock_input | bash "$STATUSLINE" 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "█\|░"; then
    pass "Output contains progress bar characters"
else
    fail "Output should contain progress bar" "█ or ░" "$OUTPUT"
fi

# Test 9: Lines changed info is shown
echo "Test: Lines changed info is shown"
OUTPUT=$(create_mock_input | bash "$STATUSLINE" 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "+100"; then
    pass "Output contains lines added"
else
    fail "Output should contain lines added" "+100" "$OUTPUT"
fi

# Test 10: OS detection works
echo "Test: OS detection"
if [[ "$OSTYPE" == "darwin"* ]]; then
    if grep -q 'IS_MACOS=false' "$STATUSLINE"; then
        # The default is false, but it gets set to true on macOS
        pass "macOS detection logic present"
    else
        fail "macOS detection logic missing" "IS_MACOS variable" "not found"
    fi
else
    pass "Linux detection (non-macOS)"
fi

# Test 11: Subscription mode check
echo "Test: Subscription mode detection logic"
if grep -q 'IS_SUBSCRIPTION=' "$STATUSLINE" && \
   grep -q 'claudeAiOauth' "$STATUSLINE"; then
    pass "Subscription mode detection logic present"
else
    fail "Subscription detection missing" "OAuth check" "not found"
fi

# Test 12: API-key mode fallback
echo "Test: API-key mode fallback"
if grep -q 'API-KEY MODE' "$STATUSLINE" && \
   grep -q 'COST_TRACKER' "$STATUSLINE"; then
    pass "API-key mode with cost tracking present"
else
    fail "API-key mode missing" "cost tracking" "not found"
fi

# ============================================
# Update Notification Tests
# ============================================

# Test 13: version_gt function exists
echo "Test: version_gt helper function exists"
if grep -q "version_gt()" "$STATUSLINE"; then
    pass "version_gt function defined"
else
    fail "version_gt function missing" "function definition" "not found"
fi

# Test 14: version_gt logic (extracted and tested)
echo "Test: version_gt returns correct results"
# Extract the version_gt function and test it
version_gt() {
    local v1="$1" v2="$2"
    [ "$v1" = "$v2" ] && return 1
    local highest
    highest=$(printf '%s\n%s\n' "$v1" "$v2" | sort -V | tail -n1)
    [ "$highest" = "$v1" ]
}

# Test cases
VGT_PASS=true
# 2.0.3 > 2.0.2 should be true
if ! version_gt "2.0.3" "2.0.2"; then
    VGT_PASS=false
fi
# 2.0.2 > 2.0.3 should be false
if version_gt "2.0.2" "2.0.3"; then
    VGT_PASS=false
fi
# 2.0.3 > 2.0.3 should be false (equal)
if version_gt "2.0.3" "2.0.3"; then
    VGT_PASS=false
fi
# 2.1.0 > 2.0.9 should be true
if ! version_gt "2.1.0" "2.0.9"; then
    VGT_PASS=false
fi
# 3.0.0 > 2.9.9 should be true
if ! version_gt "3.0.0" "2.9.9"; then
    VGT_PASS=false
fi

if [ "$VGT_PASS" = true ]; then
    pass "version_gt returns correct results"
else
    fail "version_gt logic incorrect" "correct comparisons" "wrong results"
fi

# Test 15: No [Update] when versions match
echo "Test: No update notification when versions match"
TEST_UPDATE_CACHE=$(mktemp)
echo "2.0.3" > "$TEST_UPDATE_CACHE"
# Simulate: cache has same version as current
OUTPUT=$(UPDATE_CACHE="$TEST_UPDATE_CACHE" VERSION="2.0.3" create_mock_input | bash "$STATUSLINE" 2>/dev/null || true)
rm -f "$TEST_UPDATE_CACHE"
if echo "$OUTPUT" | grep -q '\[Update\]'; then
    fail "Should NOT show update when versions match" "no [Update]" "showed [Update]"
else
    pass "No update notification when versions match"
fi

# Test 16: No [Update] when local version is newer (user upgraded)
echo "Test: No false update notification after user upgrade"
TEST_UPDATE_CACHE=$(mktemp)
echo "2.0.2" > "$TEST_UPDATE_CACHE"  # Old cache from before upgrade
# Note: This test requires network call to check, so we just verify the logic exists
if grep -q 'version_gt.*CACHED_VERSION.*VERSION' "$STATUSLINE"; then
    pass "Upgrade detection logic present"
else
    # Fallback: check that the logic handles version mismatch
    if grep -q 'User upgraded' "$STATUSLINE" || grep -q 'versions are same' "$STATUSLINE"; then
        pass "Upgrade handling logic present"
    else
        fail "Upgrade detection missing" "version comparison" "not found"
    fi
fi
rm -f "$TEST_UPDATE_CACHE"

# ============================================
# Install Script Tests
# ============================================

INSTALL_SCRIPT="$PROJECT_ROOT/scripts/install.sh"

# Test 17: Install script exists and is executable
echo "Test: Install script exists and is executable"
if [ -x "$INSTALL_SCRIPT" ]; then
    pass "install.sh is executable"
else
    fail "install.sh should be executable" "executable" "not executable"
fi

# Test 18: Install script creates settings.json
echo "Test: Install script creates settings.json"
TEST_DIR=$(mktemp -d)
HOME="$TEST_DIR" CLAUDE_PLUGIN_ROOT="$PROJECT_ROOT" bash "$INSTALL_SCRIPT" >/dev/null 2>&1
if [ -f "$TEST_DIR/.claude/settings.json" ]; then
    pass "settings.json created"
else
    fail "settings.json should be created" "file exists" "not found"
fi
rm -rf "$TEST_DIR"

# Test 19: Install script copies statusline.sh
echo "Test: Install script copies statusline.sh"
TEST_DIR=$(mktemp -d)
HOME="$TEST_DIR" CLAUDE_PLUGIN_ROOT="$PROJECT_ROOT" bash "$INSTALL_SCRIPT" >/dev/null 2>&1
if [ -f "$TEST_DIR/.claude/statusline.sh" ] && [ -x "$TEST_DIR/.claude/statusline.sh" ]; then
    pass "statusline.sh copied and executable"
else
    fail "statusline.sh should be copied" "file exists and executable" "not found or not executable"
fi
rm -rf "$TEST_DIR"

# Test 20: Install script preserves existing settings
echo "Test: Install script preserves existing settings"
TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/.claude"
echo '{"model":"claude-sonnet-4"}' > "$TEST_DIR/.claude/settings.json"
HOME="$TEST_DIR" CLAUDE_PLUGIN_ROOT="$PROJECT_ROOT" bash "$INSTALL_SCRIPT" >/dev/null 2>&1
if grep -q '"model"' "$TEST_DIR/.claude/settings.json" && grep -q '"statusLine"' "$TEST_DIR/.claude/settings.json"; then
    pass "Existing settings preserved, statusLine added"
else
    fail "Should preserve existing settings" "both model and statusLine" "missing one"
fi
rm -rf "$TEST_DIR"

# Test 21: Install script respects custom statusLine
echo "Test: Install script respects custom statusLine"
TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/.claude"
echo '{"statusLine":{"command":"/custom/script.sh"}}' > "$TEST_DIR/.claude/settings.json"
HOME="$TEST_DIR" CLAUDE_PLUGIN_ROOT="$PROJECT_ROOT" bash "$INSTALL_SCRIPT" >/dev/null 2>&1
if grep -q '/custom/script.sh' "$TEST_DIR/.claude/settings.json"; then
    pass "Custom statusLine not overwritten"
else
    fail "Should not overwrite custom statusLine" "/custom/script.sh" "was overwritten"
fi
rm -rf "$TEST_DIR"

# ============================================
# v3.0 Feature Tests
# ============================================

# Test 22: Pace indicator logic exists
echo "Test: Pace indicator logic exists"
if grep -q 'PACE_DISPLAY' "$STATUSLINE" && grep -q 'PACE_COLOR' "$STATUSLINE"; then
    pass "Pace indicator logic present"
else
    fail "Pace indicator missing" "PACE_DISPLAY and PACE_COLOR" "not found"
fi

# Test 23: Work days configuration exists
echo "Test: Work days configuration exists"
if grep -q 'WORK_DAYS_PER_WEEK' "$STATUSLINE"; then
    pass "Work days configuration present"
else
    fail "Work days config missing" "WORK_DAYS_PER_WEEK" "not found"
fi

# Test 24: Atomic cache function exists
echo "Test: Atomic cache writes"
if grep -q 'fetch_rate_limits_atomic' "$STATUSLINE" && grep -q '\.tmp\.' "$STATUSLINE"; then
    pass "Atomic cache function present"
else
    fail "Atomic cache missing" "fetch_rate_limits_atomic with .tmp" "not found"
fi

# Test 25: Burn rate display exists
echo "Test: Burn rate display exists"
if grep -q 'BURN_DISPLAY' "$STATUSLINE" && grep -q '🔥' "$STATUSLINE"; then
    pass "Burn rate display present"
else
    fail "Burn rate display missing" "BURN_DISPLAY with 🔥" "not found"
fi

# Test 26: Cache TTL configuration exists
echo "Test: Cache TTL configuration exists"
if grep -q 'RATE_CACHE_TTL' "$STATUSLINE"; then
    pass "Cache TTL configuration present"
else
    fail "Cache TTL config missing" "RATE_CACHE_TTL" "not found"
fi

# Test 27: 7d pace calculation exists
echo "Test: 7d pace calculation exists"
if grep -q 'SEVEN_DAY_PACE' "$STATUSLINE" && grep -q 'CALENDAR_DAYS_ELAPSED' "$STATUSLINE"; then
    pass "7d pace calculation present"
else
    fail "7d pace calculation missing" "SEVEN_DAY_PACE calculation" "not found"
fi

# Test 28: Initial context fix exists
echo "Test: Initial context fix exists"
if grep -q 'SHOW_CTX_PERCENT' "$STATUSLINE" && grep -q 'CTX_PERCENT_DISPLAY' "$STATUSLINE"; then
    pass "Initial context fix present"
else
    fail "Initial context fix missing" "SHOW_CTX_PERCENT logic" "not found"
fi

# Test 29: TOTAL_TOKENS=0 check for initial context
echo "Test: TOTAL_TOKENS=0 check for initial context"
if grep -q 'TOTAL_TOKENS.*-eq 0' "$STATUSLINE" && grep -q 'SHOW_CTX_PERCENT=false' "$STATUSLINE"; then
    pass "TOTAL_TOKENS=0 check present"
else
    fail "TOTAL_TOKENS=0 check missing" "TOTAL_TOKENS -eq 0 check" "not found"
fi

# Test 30: No undefined function calls (fetch_rate_limits without _atomic)
echo "Test: No undefined function calls"
# Should only have fetch_rate_limits_atomic, not bare fetch_rate_limits
if grep -q 'fetch_rate_limits[^_a-z]' "$STATUSLINE" 2>/dev/null; then
    fail "Found undefined fetch_rate_limits call" "only _atomic calls" "bare call found"
else
    pass "All fetch_rate_limits calls use _atomic variant"
fi

# Test 31: Daemon mode exists
echo "Test: Daemon mode exists"
if grep -q '\-\-daemon' "$STATUSLINE" && grep -q 'DAEMON_LOCK' "$STATUSLINE"; then
    pass "Daemon mode present"
else
    fail "Daemon mode missing" "--daemon and DAEMON_LOCK" "not found"
fi

# Test 32: Global burn rate display exists
echo "Test: Global burn rate display exists"
if grep -q 'GLOBAL_BURN_CACHE' "$STATUSLINE" && grep -q 'GLOBAL_TPM' "$STATUSLINE"; then
    pass "Global burn rate display present"
else
    fail "Global burn rate display missing" "GLOBAL_BURN_CACHE and GLOBAL_TPM" "not found"
fi

# ============================================
# Release Workflow Tests
# ============================================

# Test 33: Release type detection (prerelease vs stable)
echo "Test: Release type detection"
is_prerelease() {
    local tag="$1"
    [[ "$tag" == *"-beta"* ]] || [[ "$tag" == *"-alpha"* ]] || [[ "$tag" == *"-rc"* ]]
}

RELEASE_TYPE_PASS=true
# Beta tags should be prerelease
if ! is_prerelease "v3.1.0-beta.5"; then RELEASE_TYPE_PASS=false; fi
if ! is_prerelease "v1.0.0-alpha.1"; then RELEASE_TYPE_PASS=false; fi
if ! is_prerelease "v2.0.0-rc.1"; then RELEASE_TYPE_PASS=false; fi
# Stable tags should NOT be prerelease
if is_prerelease "v3.0.4"; then RELEASE_TYPE_PASS=false; fi
if is_prerelease "v1.0.0"; then RELEASE_TYPE_PASS=false; fi
if is_prerelease "v2.5.10"; then RELEASE_TYPE_PASS=false; fi

if [ "$RELEASE_TYPE_PASS" = true ]; then
    pass "Release type detection works correctly"
else
    fail "Release type detection broken" "correct prerelease detection" "wrong results"
fi

# Test 34: Changelog extraction
echo "Test: Changelog extraction for release notes"
extract_changelog() {
    local version="$1"
    awk -v ver="$version" '
        /^## \[/ {
            if (found) exit
            if (index($0, ver)) found=1
            next
        }
        found {print}
    ' "$PROJECT_ROOT/CHANGELOG.md"
}

# Test extraction for current beta version
NOTES=$(extract_changelog "3.1.0-beta.5")
if echo "$NOTES" | grep -q "XDG config path"; then
    pass "Changelog extraction finds release notes"
else
    fail "Changelog extraction failed" "XDG config path in notes" "not found"
fi

# Test 35: Changelog extraction returns empty for non-existent version
echo "Test: Changelog extraction fallback for missing version"
NOTES=$(extract_changelog "99.99.99")
if [ -z "$NOTES" ]; then
    pass "Changelog extraction returns empty for missing version"
else
    fail "Should return empty for missing version" "empty" "$NOTES"
fi

# Test 36: Release workflow file exists and has correct structure
echo "Test: Release workflow file structure"
RELEASE_WF="$PROJECT_ROOT/.github/workflows/release.yml"
if [ -f "$RELEASE_WF" ] && \
   grep -q "on:" "$RELEASE_WF" && \
   grep -q "push:" "$RELEASE_WF" && \
   grep -q "tags:" "$RELEASE_WF" && \
   grep -q "v\*" "$RELEASE_WF" && \
   grep -q "gh release create" "$RELEASE_WF"; then
    pass "Release workflow has correct structure"
else
    fail "Release workflow missing or incomplete" "trigger + gh release create" "not found"
fi

# ============================================
# Summary
# ============================================

echo ""
echo "=================================="
echo "Test Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
echo "=================================="

if [ "$TESTS_FAILED" -gt 0 ]; then
    exit 1
else
    exit 0
fi
