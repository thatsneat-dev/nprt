# Release Notes: v0.3.0

**Summary:** Shell palette colors and Nerd Fonts

## Overview

Improved terminal theming compatibility and added Nerd Font icon support.

## Features

### Theme-Compatible Colors

- Switched from hardcoded RGB colors to 256-color palette indices (0-15)
- Colors now adapt to terminal color schemes (Solarized, Dracula, etc.)
- Palette mapping:
  - Green (10): success/present indicators
  - Red (9): error/not-present indicators
  - Purple (13): merged PR state
  - Gray (8): subdued text (author, draft state)
  - Yellow (11): unknown/warning state

### Nerd Font Icons

- PR state now displayed with Nerd Font Octicons:
  - `\uf4dd` (nf-oct-git_pull_request_draft) Draft PR
  - `\uf407` (nf-oct-git_pull_request) Open PR
  - `\uf419` (nf-oct-git_merge) Merged PR
  - `\uf4dc` (nf-oct-git_pull_request_closed) Closed PR
- Set `NO_NERD_FONTS=1` to use fallback dot icon (●)

## Environment Variables

- `NO_NERD_FONTS` - Disable Nerd Font icons and use simple dot instead
