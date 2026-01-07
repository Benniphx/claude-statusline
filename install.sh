#!/bin/bash
# Claude Statusline Installer

set -e

REPO_URL="https://raw.githubusercontent.com/Benniphx/claude-statusline/main"
INSTALL_DIR="$HOME/.claude"

echo "Installing Claude Statusline..."

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "❌ jq is required. Install with: brew install jq"
    exit 1
fi

# Create directory if needed
mkdir -p "$INSTALL_DIR"

# Download statusline script
echo "Downloading statusline.sh..."
curl -fsSL "$REPO_URL/statusline.sh" -o "$INSTALL_DIR/statusline.sh"
chmod +x "$INSTALL_DIR/statusline.sh"

# Check if settings.json exists
SETTINGS_FILE="$INSTALL_DIR/settings.json"
if [ -f "$SETTINGS_FILE" ]; then
    # Check if statusLine already configured
    if grep -q '"statusLine"' "$SETTINGS_FILE"; then
        echo "✅ statusLine already configured in settings.json"
    else
        echo "⚠️  Please add to $SETTINGS_FILE:"
        echo '  "statusLine": {'
        echo '    "type": "command",'
        echo '    "command": "~/.claude/statusline.sh"'
        echo '  }'
    fi
else
    echo "Creating settings.json..."
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
EOF
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "Restart Claude Code to see the status line."
