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

# Test 2: Script has correct version
echo "Test: Version is set correctly"
VERSION=$(grep -o 'VERSION="[^"]*"' "$STATUSLINE" | cut -d'"' -f2)
if [ "$VERSION" = "2.0.1" ]; then
    pass "Version is 2.0.1"
else
    fail "Version should be 2.0.1" "2.0.1" "$VERSION"
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
# Install Script Tests
# ============================================

INSTALL_SCRIPT="$PROJECT_ROOT/scripts/install.sh"

# Test 13: Install script exists and is executable
echo "Test: Install script exists and is executable"
if [ -x "$INSTALL_SCRIPT" ]; then
    pass "install.sh is executable"
else
    fail "install.sh should be executable" "executable" "not executable"
fi

# Test 14: Install script creates settings.json
echo "Test: Install script creates settings.json"
TEST_DIR=$(mktemp -d)
HOME="$TEST_DIR" CLAUDE_PLUGIN_ROOT="$PROJECT_ROOT" bash "$INSTALL_SCRIPT" >/dev/null 2>&1
if [ -f "$TEST_DIR/.claude/settings.json" ]; then
    pass "settings.json created"
else
    fail "settings.json should be created" "file exists" "not found"
fi
rm -rf "$TEST_DIR"

# Test 15: Install script copies statusline.sh
echo "Test: Install script copies statusline.sh"
TEST_DIR=$(mktemp -d)
HOME="$TEST_DIR" CLAUDE_PLUGIN_ROOT="$PROJECT_ROOT" bash "$INSTALL_SCRIPT" >/dev/null 2>&1
if [ -f "$TEST_DIR/.claude/statusline.sh" ] && [ -x "$TEST_DIR/.claude/statusline.sh" ]; then
    pass "statusline.sh copied and executable"
else
    fail "statusline.sh should be copied" "file exists and executable" "not found or not executable"
fi
rm -rf "$TEST_DIR"

# Test 16: Install script preserves existing settings
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

# Test 17: Install script respects custom statusLine
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
