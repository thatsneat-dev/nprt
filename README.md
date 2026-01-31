<div align="center">
    <img src=".github/assets/nprt.png" alt="nprt Logo">
    <br/>
    <h1>nprt</h1>
</div>
<div align="center">
  <p>
    <a href="https://github.com/thatsneat-dev/nprt/releases/latest">
      <img alt="Latest release" src="https://img.shields.io/github/v/release/thatsneat-dev/nprt?style=for-the-badge&color=C9CBFF&logoColor=D9E0EE&labelColor=302D41" />
    </a>
    <a href="https://github.com/thatsneat-dev/nprt/pulse">
      <img alt="Last commit" src="https://img.shields.io/github/last-commit/thatsneat-dev/nprt?style=for-the-badge&color=8bd5ca&logoColor=D9E0EE&labelColor=302D41"/>
    </a>
    <a href="https://github.com/thatsneat-dev/nprt/blob/main/LICENSE">
      <img alt="License" src="https://img.shields.io/github/license/thatsneat-dev/nprt?style=for-the-badge&color=ee999f&logoColor=D9E0EE&labelColor=302D41" />
    </a>
    <a href="https://github.com/thatsneat-dev/nprt/stargazers">
      <img alt="Stars" src="https://img.shields.io/github/stars/thatsneat-dev/nprt?style=for-the-badge&color=c69ff5&logoColor=D9E0EE&labelColor=302D41" />
    </a>
    <a href="https://github.com/NotAShelf/nvf/issues">
      <img alt="Issues" src="https://img.shields.io/github/issues/thatsneat-dev/nprt?style=for-the-badge&color=F5E0DC&logoColor=D9E0EE&labelColor=302D41" />
    </a>
    <a href="https://github.com/thatsneat-dev/nprt">
      <img alt="Repo Size" src="https://img.shields.io/github/repo-size/thatsneat-dev/nprt?color=%23DDB6F2&label=SIZE&style=for-the-badge&logoColor=D9E0EE&labelColor=302D41" />
    </a>
  </p>
</div>

`nprt` (**N**ixpkgs **PR** **T**racker) is a CLI tool to track which
[nixpkgs](https://github.com/NixOS/nixpkgs) channels contain a given pull
request.

![View cast here: https://asciinema.org/a/5yBQZUxMM8DUYILC](./docs/assets/nprt.svg)

## Features

- Check if a PR has been merged into various nixpkgs channels
- Support for PR numbers or full GitHub URLs
- Colored terminal output with Nerd Font icons
- Clickable hyperlinks to PRs (in supported terminals)
- JSON output for scripting
- Parallel channel checking for fast results

## Installation

### Using the Nix Flake

This repository includes a flake for installation:

```bash
# Run directly
nix run github:thatsneat-dev/nprt

# Install to profile
nix profile install github:thatsneat-dev/nprt

# Or add to your flake inputs
{
  inputs.nprt.url = "github:thatsneat-dev/nprt";
}
```

### Using Go

Requires Go 1.25 or later:

```bash
go install github.com/thatsneat-dev/nprt@latest
```

## Usage

See the full [usage documentation](docs/USAGE.md) for options, examples, and
exit codes.

```bash
# Check by PR number
nprt 475593

# Check by PR URL
nprt https://github.com/NixOS/nixpkgs/pull/475593

# JSON output for scripting
nprt --json 475593
```
