# Release Notes: v0.1.0

**Summary:** Initial release

## Overview

First release of `nprt` (NixPkgs PR Tracker), a CLI tool to track which nixpkgs
channels contain a given pull request.

## Features

- **PR Tracking**: Check if a PR has been merged into various nixpkgs channels
- **Flexible Input**: Accept PR numbers or full GitHub PR URLs
- **Channel Checking**: Query master, staging-next, nixpkgs-unstable,
  nixos-unstable-small, and nixos-unstable by default
- **Custom Channels**: Override default channels with `--channels` flag
- **Colored Output**: ANSI-colored status indicators (green ✓, red ✗)
- **JSON Output**: Machine-readable output with `--json` flag
- **Parallel Requests**: Check all channels concurrently for fast results

## CLI Options

- `--channels` - Comma-separated list of channels to check
- `--color` - Color mode: auto, always, never
- `--json` - Output results as JSON
- `--verbose` - Show detailed progress information
- `--version` - Print version and exit
- `-h, --help` - Show help message

## Environment Variables

- `GITHUB_TOKEN` - GitHub personal access token for higher API rate limits
