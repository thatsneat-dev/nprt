# Context

This project is intended to be a CLI application written in Golang (>=v1.23)
that can accept a PR number or link from the
[nixpkgs repository](https://github.com/NixOS/nixpkgs).

PRs in the nixpkgs repository are merged into branches. Certain branches are
used as "channels" - the particularly important ones are:

- master (where most PRs are merged by default)
- staging-next (where large/impactful PRs are merged into master from - they
  originate from staging)
- nixpkgs-unstable
- nixos-unstable-small
- nixos-unstable

# User Experience

## Invoking the command

The command should able to be invoked via the following syntax:

```bash
nprt [ PR number | PR link ]
```

PR numbers follow the standard GitHub practice of integers, unsigned, and should
not include the # prefix. PR links must be for the NixOS/nixpkgs repository, and
would follow the format below:

https://github.com/NixOS/nixpkgs/pull/476497

### CLI Flags

| Flag | Type | Description |
|------|------|-------------|
| `--channels` | comma-separated list | Override the default channel list |
| `--color` | `auto`, `always`, `never` | Control ANSI color output (default: `auto`) |
| `--json` | boolean | Output JSON instead of table |
| `-h, --help` | boolean | Print usage information |
| `--version` | boolean | Print version and exit |

**Environment Variables:**

- `GITHUB_TOKEN`: If set, used for authenticated GitHub API requests (higher
  rate limits)

## Output

The output should be a neatly formatted table containing the channel names in
the first column, with a simple status icon in the second, like the example
below:

```
CHANNEL               STATUS
----------------------------
staging-next          ✓
master                ✓
nixos-unstable-small  ✗
nixpkgs-unstable      ✗
nixos-unstable        ✗
```

- The check marks should be colored green using ANSI colors, and the ✗ symbol
  colored red with ANSI colors.
- All successful channels (those which the PR has been promoted to) should be
  listed first
- Any failed channels (those which the PR has not been promoted to) should be
  listed last, in alphabetical order
- When `--no-color` is set or stdout is not a TTY, output plain text without
  ANSI codes

### JSON Output

When `--json` is specified, output follows this schema:

```json
{
  "pr": 476497,
  "state": "merged",
  "merge_commit": "abc123def456",
  "channels": [
    {"name": "staging-next", "branch": "staging-next", "status": "present"},
    {"name": "master", "branch": "master", "status": "not_present"}
  ]
}
```

# Algorithm

## How It Works

1. **Parse input**
   - Accept a single argument: either a PR number (`\d+`) or a full PR URL
     matching `https://github.com/NixOS/nixpkgs/pull/{number}`.
   - Extract the PR number; reject anything else with a clear error.

2. **Fetch PR metadata (GitHub REST API)**
   - Call `GET /repos/NixOS/nixpkgs/pulls/{number}`:
     - Use fields: `state`, `merged`, `merge_commit_sha`, `base.ref`.
   - Cases:
     - `404`: "PR not found."
     - `state != "closed"` or `merged == false`: "PR not merged yet."
     - `merged == true`: Use `merge_commit_sha` as the canonical commit.

3. **Check "is commit in branch?" via compare API**
   - For each channel branch, call:
     `GET /repos/NixOS/nixpkgs/compare/{commit}...{branch}`
   - Interpret:
     - If `status` is `"behind"` or `"identical"`: commit is in the branch →
       channel **contains** the PR.
     - Otherwise: channel does **not** contain the PR.

4. **Status mapping & display**
   - For each channel, record: `present`, `not_present`, or `unknown`.
   - Sort: all `present` channels first, then `not_present`/`unknown`
     alphabetically.
   - Map to icons: `present` → green ✓, `not_present` → red ✗, `unknown` →
     yellow ?

# Error Handling

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (including unmerged PRs, which show all channels as not_present) |
| 1 | General error (PR not found, no merge commit, network/API issues except 403) |
| 2 | CLI usage error (bad arguments) |
| 3 | GitHub rate limit or auth failure (HTTP 403) |

## Error Messages

**Input validation:**
- Invalid argument: `Usage: nprt [PR number | https://github.com/NixOS/nixpkgs/pull/{number}]`

**PR lifecycle:**
- PR not found: `PR #NNN not found in NixOS/nixpkgs.`
- PR not merged: `PR #NNN is not merged; it cannot be in any channel yet.`

**Network/API:**
- Network error: `Network error talking to GitHub: <reason>`
- Rate limit: `GitHub API rate limit exceeded. Try again later or set GITHUB_TOKEN.`
- Auth error: `GitHub authentication failed. Check your GITHUB_TOKEN.`

**Partial failures:**
- If some channels fail to check, mark them as `unknown` and print a warning.

# Code Structure and Standards

## Architecture

This code repository follows clean
[SOLID architecture](https://en.wikipedia.org/wiki/SOLID).

### Package Layout

```
cmd/
  nprt/
    main.go           # CLI entry point, flag parsing, dependency injection

internal/
  config/
    config.go         # Configuration struct, parsing, defaults
  github/
    client.go         # GitHub REST API client
  core/
    core.go           # Domain logic (PR status, channel checking)
  render/
    render.go         # Table and JSON output rendering

tests/
  config_test.go      # Tests for config package
  github_test.go      # Tests for github package
  core_test.go        # Tests for core package
  render_test.go      # Tests for render package
```

## Authoring Code

- All code should be properly linted and formatted. The required formatter for
  this project is `gofumpt`. No code that fails a formatting call should be
  accepted or committed.
- Any code that is an "adapter" or "handler" (not responsible for the handling
  of user input/invocation or printing to the console) should be written in its
  own `.go` file within the `internal` directory.
- All handlers and adapters MUST be accompanied by unit tests, which should have
  the same name as the parent adapter/handler file, with a `_test` suffix, and
  hosted in the `tests` directory.
- The main entry point should be housed in `cmd/nprt/main.go`.

## Automation & Builds

A `Makefile` should be used to provide helper commands to:

- perform a repo-wide formatting check for all `.go` files (`make format-check`)
- run all tests (`make test`)
- perform an artifact build (`make build`)

# Testing Strategy

## Unit Tests

**config package:**
- Parsing PR numbers and URLs
- Parsing `--channels` flag (valid/invalid input)
- Color/no-color and output mode selection

**github package:**
- Use `httptest.Server` to simulate GitHub responses:
  - Successful PR fetch with `merge_commit_sha`
  - PR not merged
  - PR not found (404)
  - Compare results: commit present/not present
  - Rate limit (403)

**core package:**
- Sorting channels (present first, then alphabetical)
- Mapping compare results to ChannelStatus
- Behavior when some channels return `unknown`

**render package:**
- Table rendering (with and without color)
- JSON rendering (validate schema)

## Edge Cases

- PR merged with squash (no merge_commit_sha)
- Missing channel branch (404 on compare)
- Network/API errors during compare for some channels only
