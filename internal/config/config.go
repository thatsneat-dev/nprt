// Package config handles configuration parsing and environment settings
// for the nixpkgs PR tracker.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var defaultChannels = []Channel{
	{Name: "master", Branch: "master"},
	{Name: "staging-next", Branch: "staging-next"},
	{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable"},
	{Name: "nixos-unstable-small", Branch: "nixos-unstable-small"},
	{Name: "nixos-unstable", Branch: "nixos-unstable"},
}

// GetDefaultChannels returns a copy of the default channels.
func GetDefaultChannels() []Channel {
	out := make([]Channel, len(defaultChannels))
	copy(out, defaultChannels)
	return out
}

// AvailableChannelNames returns the default channel names as a comma-separated string.
func AvailableChannelNames() string {
	names := make([]string, len(defaultChannels))
	for i, ch := range defaultChannels {
		names[i] = ch.Name
	}
	return strings.Join(names, ", ")
}

// Channel represents a nixpkgs branch that serves as a release channel.
type Channel struct {
	Name   string
	Branch string
}

// prURLRegex matches GitHub PR URLs for the NixOS/nixpkgs repository.
var prURLRegex = regexp.MustCompile(`^https://github\.com/NixOS/nixpkgs/pull/(\d+)/?$`)

// issueURLRegex matches GitHub issue URLs for the NixOS/nixpkgs repository.
var issueURLRegex = regexp.MustCompile(`^https://github\.com/NixOS/nixpkgs/issues/(\d+)/?$`)

// ParsePRInput parses a PR number or GitHub PR/issue URL and returns the number.
// Issue URLs are accepted so that the user gets helpful error messages with related PRs.
func ParsePRInput(input string) (int, error) {
	input = strings.TrimSpace(input)

	if num, err := strconv.Atoi(input); err == nil {
		if num <= 0 {
			return 0, fmt.Errorf("PR number must be a positive integer")
		}
		return num, nil
	}

	matches := prURLRegex.FindStringSubmatch(input)
	if matches != nil {
		num, _ := strconv.Atoi(matches[1])
		return num, nil
	}

	matches = issueURLRegex.FindStringSubmatch(input)
	if matches != nil {
		num, _ := strconv.Atoi(matches[1])
		return num, nil
	}

	return 0, fmt.Errorf("invalid PR input: must be a number or https://github.com/NixOS/nixpkgs/pull/{number}")
}

// ParseChannels parses a comma-separated list of channel names and returns
// matching channels. Returns an error if any names are unknown.
// Returns all defaults if input is empty.
func ParseChannels(input string) ([]Channel, error) {
	if input == "" {
		return GetDefaultChannels(), nil
	}

	var requested []string
	for _, part := range strings.Split(input, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			requested = append(requested, name)
		}
	}

	if len(requested) == 0 {
		return GetDefaultChannels(), nil
	}

	valid := make(map[string]bool)
	for _, ch := range defaultChannels {
		valid[ch.Name] = true
	}

	var unknown []string
	for _, name := range requested {
		if !valid[name] {
			unknown = append(unknown, name)
		}
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown channels: %s; available: %s", strings.Join(unknown, ", "), AvailableChannelNames())
	}

	seen := make(map[string]bool)
	var channels []Channel
	for _, ch := range defaultChannels {
		for _, name := range requested {
			if ch.Name == name && !seen[name] {
				channels = append(channels, ch)
				seen[name] = true
			}
		}
	}

	return channels, nil
}

// GetGitHubToken returns the GITHUB_TOKEN environment variable.
func GetGitHubToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

// IsTerminal returns true if stdout is connected to a terminal.
func IsTerminal() bool {
	return isTerminalFile(os.Stdout)
}

func isTerminalFile(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ShouldUseHyperlinks determines if OSC 8 hyperlinks should be used based on
// the hyperlink mode setting and environment. Hyperlinks are independent of
// color: NO_COLOR does not disable hyperlinks, but NO_HYPERLINKS does.
func ShouldUseHyperlinks(hyperlinkMode string) (bool, error) {
	switch hyperlinkMode {
	case "always":
		return true, nil
	case "never":
		return false, nil
	case "auto", "":
		if os.Getenv("NO_HYPERLINKS") != "" {
			return false, nil
		}
		return IsTerminal(), nil
	default:
		return false, fmt.Errorf("invalid hyperlink mode %q: must be auto, always, or never", hyperlinkMode)
	}
}

// ShouldUseHyperlinksForFile determines if OSC 8 hyperlinks should be used
// for a specific file descriptor. Hyperlinks are independent of color support.
func ShouldUseHyperlinksForFile(hyperlinkMode string, f *os.File) bool {
	switch hyperlinkMode {
	case "always":
		return true
	case "never":
		return false
	default:
		if os.Getenv("NO_HYPERLINKS") != "" {
			return false
		}
		return isTerminalFile(f)
	}
}

// ShouldUseColorForFile determines if ANSI color codes should be used
// for a specific file descriptor. This is useful when stderr needs
// independent color detection from stdout.
func ShouldUseColorForFile(colorMode string, f *os.File) bool {
	switch colorMode {
	case "always":
		return true
	case "never":
		return false
	default:
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		return isTerminalFile(f)
	}
}

// ShouldUseColor determines if ANSI color codes should be used based on
// the color mode setting and environment.
func ShouldUseColor(colorMode string) (bool, error) {
	switch colorMode {
	case "always":
		return true, nil
	case "never":
		return false, nil
	case "auto", "":
		if os.Getenv("NO_COLOR") != "" {
			return false, nil
		}
		return IsTerminal(), nil
	default:
		return false, fmt.Errorf("invalid color mode %q: must be auto, always, or never", colorMode)
	}
}
