# Release Notes: v0.2.0

**Summary:** Post-initial cleanup

## Overview

Quality-of-life improvements, Nix flake support, and project cleanup.

## Features

### Nix Flake Support

- Added `flake.nix` for reproducible builds and easy installation via Nix
- Includes development shell with Go toolchain and gofumpt

### CLI Improvements

- **Flexible Flag Ordering**: Flags can now appear before or after the PR
  argument (e.g., `nprt 123 --json` and `nprt --json 123` both work)
- **PR Title Display**: Shows the PR title in parentheses after the PR number
- **Author Attribution**: Displays "by: author" line for merged PRs
- **Clickable Hyperlinks**: PR number links to GitHub in supported terminals
  (OSC 8 escape sequences)

### Visual Refinements

- Improved table formatting with centered status symbols
- Author line displayed in subdued gray color
- Use blue instead of purple for merged state (better accessibility)

## Bug Fixes

- Fixed various edge cases from v0.1.0
- Removed binary from git tracking
- Added proper `.gitignore`

## Project Updates

- Added MIT license
- Improved Makefile with additional targets
