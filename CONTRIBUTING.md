# Contributing

## Release Checklist

When releasing a new version, **all these files must be updated**:

| File | Location | Format |
|------|----------|--------|
| `scripts/statusline.sh` | Line 2 + Line 5 | `v2.0.5` / `VERSION="2.0.5"` |
| `.claude-plugin/plugin.json` | `version` field | `"2.0.5"` |
| `.claude-plugin/marketplace.json` | `plugins[0].version` | `"2.0.5"` |
| `README.md` | Badge in line 5 | `version-2.0.5-blue` |
| `CHANGELOG.md` | New section at top | `## [2.0.5] - YYYY-MM-DD` |

The CI test (`tests/test_statusline.sh`) will **fail** if any version is out of sync.

## Quick Release Steps

```bash
# 1. Update all version references (see table above)

# 2. Run tests locally
./tests/test_statusline.sh

# 3. Commit
git add -A
git commit -m "feat: v2.0.X - short description"

# 4. Tag and push
git tag -a v2.0.X -m "v2.0.X"
git push origin main --tags

# 5. Create GitHub release (--latest ensures it's not a draft!)
gh release create v2.0.X --title "v2.0.X" --latest --notes "## Changes
- Feature 1
- Fix 2"

# 6. Verify release is published (not Draft!)
gh release list --limit 1
# Should show: v2.0.X  Latest  v2.0.X  <date>
# NOT:         v2.0.X  Draft   v2.0.X  <date>
```

**Important:** If release shows as `Draft`, publish it with:
```bash
gh release edit v2.0.X --draft=false
```

## Development

### Running Tests

```bash
./tests/test_statusline.sh
```

### Testing Locally

```bash
# Test with mock data
echo '{"context_window":{"context_window_size":200000},"model":{"display_name":"Test"}}' | ./scripts/statusline.sh
```
