# Release Notes: v0.4.0

**Summary:** Error handling and structured logging

## Overview

Improved error messages for common failure cases, structured logging with zap,
and a code quality review pass.

## Features

### Improved Error Handling

- **Issue vs PR Detection**: When a number is an Issue (not a PR), the error
  message now clearly states this with the issue title and URL
- **Not Found Detection**: When a number doesn't exist at all, provides a
  specific "no PR or issue #N exists" message
- **Colored Errors**: All error messages display in red when color is enabled
- **Cleaner API Errors**: GitHub API error responses now extract the `message`
  field from JSON instead of dumping raw response bodies

### Structured Logging

- Replaced ad-hoc verbose output with zap structured logging
- In verbose mode (`--verbose`), logs include:
  - Timestamps (HH:MM:SS format)
  - Log level (DEBUG)
  - Source file and line number
  - Function name
  - Structured key-value fields
- In normal mode, logging is completely silent (no performance overhead)

### Visual Fixes

- Fixed Nerd Font icon for merged PRs: now uses `\uf419` (nf-oct-git_merge)
  instead of sharing the same icon as open PRs

## Bug Fixes

- Fixed edge case where `/pulls` returns 404 but `/issues` indicates it's a PR
  (now provides actionable error about checking GITHUB_TOKEN permissions)
- Simplified error handling flow in main.go

## Documentation

- Updated DESIGN.md to match implementation (compare API uses `behind_by == 0`)
- Fixed DESIGN.md reference to `--no-color` (now correctly documents
  `--color=never` and `NO_COLOR` env var)
- Updated README example output to show current format with merged icon

## Testing

- Added tests for `FormatError` helper
- Added test for "issue endpoint says PR" edge case
- Added test for GitHub API JSON message parsing
