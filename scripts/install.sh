#!/bin/bash
# Claude Statusline Plugin - Install Script
# Runs on SessionStart to ensure statusline is configured

set -e

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(dirname "$(dirname "$0")")}"
INSTALL_DIR="$HOME/.claude"
SETTINGS_FILE="$INSTALL_DIR/settings.json"
STATUSLINE_SRC="$PLUGIN_ROOT/scripts/statusline.sh"
STATUSLINE_DST="$INSTALL_DIR/statusline.sh"

# Get source version
SOURCE_VERSION=$(grep -o 'VERSION="[^"]*"' "$STATUSLINE_SRC" 2>/dev/null | cut -d'"' -f2 || echo "unknown")

# Check if already installed and up-to-date
if [ -f "$STATUSLINE_DST" ]; then
    INSTALLED_VERSION=$(grep -o 'VERSION="[^"]*"' "$STATUSLINE_DST" 2>/dev/null | cut -d'"' -f2 || echo "unknown")

    if [ "$INSTALLED_VERSION" = "$SOURCE_VERSION" ]; then
        # Already installed and same version - skip silently
        exit 0
    fi
fi

# Check dependencies
check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        echo "statusline: Missing dependency '$1'. Install with: $2" >&2
        return 1
    fi
}

check_dependency jq "brew install jq (macOS) or apt install jq (Linux)" || exit 0
check_dependency curl "brew install curl (macOS) or apt install curl (Linux)" || exit 0

# Create install directory
mkdir -p "$INSTALL_DIR"

# Copy statusline script
cp "$STATUSLINE_SRC" "$STATUSLINE_DST"
chmod +x "$STATUSLINE_DST"

# Update settings.json
if [ -f "$SETTINGS_FILE" ]; then
    # Check if statusLine already configured
    if jq -e '.statusLine' "$SETTINGS_FILE" > /dev/null 2>&1; then
        # Already configured - check if it points to our script
        CURRENT_CMD=$(jq -r '.statusLine.command // empty' "$SETTINGS_FILE")
        if [ "$CURRENT_CMD" != "~/.claude/statusline.sh" ]; then
            # Different statusLine configured - don't overwrite
            exit 0
        fi
    else
        # Add statusLine config
        jq '. + {"statusLine": {"type": "command", "command": "~/.claude/statusline.sh"}}' "$SETTINGS_FILE" > "$SETTINGS_FILE.tmp"
        mv "$SETTINGS_FILE.tmp" "$SETTINGS_FILE"
    fi
else
    # Create new settings.json
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
EOF
fi

echo "statusline: Installed v$SOURCE_VERSION" >&2
