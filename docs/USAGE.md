---
title: NPRT
section: 1
header: User Manual
footer: nprt
date: 2025
---

# NAME

nprt - track which nixpkgs channels contain a given pull request

# SYNOPSIS

**nprt** \[*options*\] \<*PR number* | *PR URL*\>

# DESCRIPTION

**nprt** checks which nixpkgs release channels contain a given pull request
by comparing the PR's merge commit against each channel branch via the
GitHub API.

# USAGE

```bash
# Check by PR number
nprt 475593

# Check by PR URL
nprt https://github.com/NixOS/nixpkgs/pull/475593

# Check specific channels only
nprt --channels=master,nixos-unstable 475593

# JSON output for scripting
nprt --json 475593

# Show a propagation graph with branch head commits
nprt --netgraph 475593

# Force colors (useful for piping)
nprt --color=always 475593

# Force hyperlinks (independent of color)
nprt --hyperlinks=always 475593

# Verbose output for debugging
nprt --verbose 475593
```

# EXAMPLE OUTPUT

```
● PR #475593 (golang: 1.23.5 -> 1.23.6)
by: someone

CHANNEL               STATUS
────────────────────────────
master                  ✓
staging-next            ✓
nixpkgs-unstable        ✓
nixos-unstable-small    ✓
nixos-unstable          ✗
```

If any channel checks fail (for example, GitHub returns a rate-limit 403),
those rows show `?` and the underlying error is summarized below the table:

```
!  GitHub API rate limit exceeded. Try again later or set GITHUB_TOKEN. (3 channels affected)
```

With `--netgraph`, nprt keeps the table output and appends a compact
git-log-style graph showing the propagation path the PR's merge commit
follows and a fan-out of the remaining checked channels:

```
  ◆  abc123def456  base staging
  │
  ●  stgnxhead123  staging-next
  │
  ●  masterhead12  master
  ├─●  smallhead123  nixos-unstable-small
  │ ╰─○                nixos-unstable
  ╰─●  pkgshead1234  nixpkgs-unstable
```

Glyph legend: `◆` is the PR's merge commit, `●` (green) means the PR is
present on that branch, `○` (red) means it is not yet there, and `?`
(yellow) means the check could not be completed. The 12-character SHA
column shows the branch HEAD commit for **present** channels only — for
pending or unknown channels the cell is left blank because the branch
HEAD has no relationship to the PR. All SHAs are clickable in terminals
that support OSC 8 hyperlinks.

The PR state icon reflects the current PR state; the author line is shown
when author data is available. In terminals with Nerd Fonts installed,
state-specific icons are displayed (`\uf419` for merged, `\uf407` for open,
etc.). Set `NO_NERD_FONTS=1` to use a simple dot (●) instead.

Hyperlinks (OSC 8) are controlled independently from colors. Use `--hyperlinks`
to override auto-detection, or set `NO_HYPERLINKS=1` to disable. Note that
`NO_COLOR` does **not** disable hyperlinks.

# OPTIONS

| Option       | Description                                             |
| ------------ | ------------------------------------------------------- |
| `--channels` | Comma-separated list of channels to check               |
| `--color`    | Color mode: `auto`, `always`, `never` (default: `auto`) |
| `--hyperlinks` | Hyperlink mode: `auto`, `always`, `never` (default: `auto`) |
| `--json`     | Output results as JSON                                  |
| `--netgraph` | Append an ASCII propagation graph with branch commit IDs |
| `--verbose`  | Show detailed progress and debug information            |
| `-v, --version` | Print version and exit                               |
| `--timeline-pages` | Max pages of timeline to fetch for related PRs (default: 3) |
| `-h, --help` | Show help message                                       |

# ENVIRONMENT

| Variable          | Description                                                                   |
| ----------------- | ----------------------------------------------------------------------------- |
| `GITHUB_TOKEN`    | GitHub personal access token for higher API rate limits                       |
| `NO_COLOR`        | Disable colors when set (respects [NO_COLOR](https://no-color.org/) standard) |
| `NO_HYPERLINKS`   | Disable OSC 8 hyperlinks when set                                            |
| `NO_NERD_FONTS`   | Disable Nerd Font icons and use fallback dots                                 |

# ISSUE HANDLING

If you provide an issue number instead of a PR number, nprt will detect this
and display a warning with the issue details and any related pull requests
found in the issue's timeline:

```
WARNING: input is an issue, not a pull request
 Issue #12345 (Example issue title)

Related pull requests:

   #67890  Fix for issue 12345
   #67891  Another related fix
```

The related PRs are discovered via GitHub's timeline API. Use `--timeline-pages`
to control how many pages of timeline events to fetch (default: 3).

# CHANNELS

By default, the following channels are checked:

- `master` - Main development branch
- `staging-next` - Staging integration branch
- `nixpkgs-unstable` - Unstable channel for non-NixOS users
- `nixos-unstable-small` - Fast-moving unstable channel with fewer packages
- `nixos-unstable` - Main unstable channel for NixOS

Most PRs merge directly to `master`; those can advance to `nixpkgs-unstable`,
`nixos-unstable-small`, and `nixos-unstable` after the corresponding Hydra
jobsets succeed. Mass-rebuild PRs typically merge to `staging`, then are batched
through `staging-next`, and finally reach `master` via a manual PR before the
unstable channels can update. Stable fixes use `release-YY.MM` branches, or
`staging-YY.MM` / `staging-next-YY.MM` for stable mass rebuilds, before the
stable channel branches move.

# EXIT CODES

| Code | Meaning                                      |
| ---- | -------------------------------------------- |
| 0    | Success (including unmerged PRs)             |
| 1    | General error (PR not found, network issues) |
| 2    | CLI usage error (bad arguments)              |
| 3    | GitHub rate limit or auth failure            |
