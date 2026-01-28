#!/bin/bash
# Setup git hooks for development
# Run once after cloning: ./scripts/setup-hooks.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "Installing git hooks..."

# Pre-push hook: Run tests before pushing
cat > "$HOOKS_DIR/pre-push" << 'EOF'
#!/bin/bash
# Pre-push hook: Run tests before pushing

echo "Running tests before push..."
./tests/test_statusline.sh

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Tests failed! Push aborted."
    echo "Fix the issues and try again."
    exit 1
fi

echo "✅ All tests passed. Pushing..."
EOF

chmod +x "$HOOKS_DIR/pre-push"

echo "✅ Git hooks installed:"
echo "   - pre-push: runs tests before every push"
