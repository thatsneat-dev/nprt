# Release Notes: v1.0.0

**Summary:** Independent hyperlink control, improved error handling tests

## Overview

This release decouples hyperlink support from color support, giving users
granular control over OSC 8 hyperlinks via a new `--hyperlinks` flag and
`NO_HYPERLINKS` environment variable. It also adds test coverage for
`ShouldUseColor` error handling.

## Features

### Independent Hyperlink Control

- **`--hyperlinks` flag**: New CLI flag with `auto`, `always`, `never` modes
  (default: `auto`), matching the existing `--color` flag pattern
- **`NO_HYPERLINKS` env var**: Set to disable OSC 8 hyperlinks independently
  of color support — `NO_COLOR` no longer affects hyperlinks
- **`ShouldUseHyperlinks(mode)`**: New config function for stdout hyperlink
  detection, respecting `NO_HYPERLINKS` and terminal detection
- **`ShouldUseHyperlinksForFile(mode, file)`**: New config function for
  per-file-descriptor hyperlink detection (used for stderr rendering)

### Bug Fix: Hyperlink/Color Coupling

- Previously, stderr hyperlinks in the issue warning renderer were controlled
  by `ShouldUseColorForFile("auto", os.Stderr)`, meaning `NO_COLOR` would
  incorrectly disable hyperlinks. This is now fixed to use
  `ShouldUseHyperlinksForFile` with the `--hyperlinks` flag value.
- Stdout hyperlink detection previously used `IsTerminal()` directly with no
  flag control. It now uses `ShouldUseHyperlinks(hyperlinkMode)`.

## Testing

- **`ShouldUseColor` error handling**: Added `TestShouldUseColor_InvalidMode`
  and `TestShouldUseColor_ValidModes` to verify error returns for invalid
  color modes and correct behavior for `always`/`never`
- **`ShouldUseHyperlinks` tests**: Added `TestShouldUseHyperlinks_InvalidMode`,
  `TestShouldUseHyperlinks_ValidModes`, `TestShouldUseHyperlinks_NoHyperlinksEnv`,
  and `TestShouldUseHyperlinks_IgnoresNoColor`
- **`ShouldUseHyperlinksForFile` tests**: Added coverage for `always`, `never`,
  and `NO_HYPERLINKS` env var behavior

## Documentation

- Updated `docs/USAGE.md` (manpage source) with `--hyperlinks` flag,
  `NO_HYPERLINKS` env var, and usage example
- Added note that hyperlinks are independent of color control
