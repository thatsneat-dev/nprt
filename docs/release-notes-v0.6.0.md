# Release Notes: v0.6.0

**Summary:** Manpage, code cleanup, and packaging improvements

## Overview

This release introduces a manpage, refactors internal structure for
maintainability, and modernizes the Nix flake for correctness and
cross-platform reliability.

## Features

### Manpage

- **`nprt.1` manpage**: Generated from `docs/USAGE.md` via Pandoc, installed
  automatically with the Nix package

### CLI Refactor

- **Extracted CLI logic**: Argument parsing (`reorderArgs`, `hasUnknownFlags`)
  moved to `internal/cli/args.go`
- **State constants**: `StateOpen`, `StateClosed`, `StateMerged`, `StateDraft`
  replace magic strings in the `github` package
- **Deduplicated `doRequest`**: Now delegates to `doRequestWithAccept`
- **Removed duplicate `RelatedPR`**: `render` package uses `github.RelatedPR`
  directly
- **Unexported `DefaultChannels`**: Renamed to `defaultChannels` with a
  `GetDefaultChannels()` accessor returning a copy

## Packaging

### Nix Flake

- Fixed manpage installation using `installShellFiles`/`installManPage`
- Fixed `VERSION` trailing newline via `lib.fileContents`
- Added `meta.mainProgram` and `meta.platforms`
- Switched devShell to `mkShell` with `packages` (added `alejandra`, `statix`,
  `deadnix`)
- Updated nixpkgs input

### README

- Added badges and logo
- Added demo GIF
