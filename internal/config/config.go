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

// DefaultChannels is the set of nixpkgs channels checked when no --channels flag is provided.
var DefaultChannels = []Channel{
	{Name: "master", Branch: "master"},
	{Name: "staging-next", Branch: "staging-next"},
	{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable"},
	{Name: "nixos-unstable-small", Branch: "nixos-unstable-small"},
	{Name: "nixos-unstable", Branch: "nixos-unstable"},
}

// AvailableChannelNames returns the default channel names as a comma-separated string.
func AvailableChannelNames() string {
	names := make([]string, len(DefaultChannels))
	for i, ch := range DefaultChannels {
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

// ParsePRInput parses a PR number or GitHub PR URL and returns the PR number.
func ParsePRInput(input string) (int, error) {
	input = strings.TrimSpace(input)

	if num, err := strconv.Atoi(input); err == nil {
		if num <= 0 {
			return 0, fmt.Errorf("PR number must be a positive integer")
		}
		return num, nil
	}

	matches := prURLRegex.FindStringSubmatch(input)
	if matches == nil {
		return 0, fmt.Errorf("invalid PR input: must be a number or https://github.com/NixOS/nixpkgs/pull/{number}")
	}

	// Error ignored: regex guarantees matches[1] contains only digits
	num, _ := strconv.Atoi(matches[1])
	return num, nil
}

// ParseChannels parses a comma-separated list of channel names and returns
// matching channels from DefaultChannels. Returns all defaults if input is empty.
func ParseChannels(input string) ([]Channel, error) {
	if input == "" {
		return DefaultChannels, nil
	}

	requested := make(map[string]bool)
	for _, part := range strings.Split(input, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			requested[name] = true
		}
	}

	if len(requested) == 0 {
		return DefaultChannels, nil
	}

	channels := make([]Channel, 0)
	for _, ch := range DefaultChannels {
		if requested[ch.Name] {
			channels = append(channels, ch)
		}
	}

	if len(channels) == 0 {
		return nil, fmt.Errorf("no matching channels found; available: %s", AvailableChannelNames())
	}

	return channels, nil
}

// GetGitHubToken returns the GITHUB_TOKEN environment variable.
func GetGitHubToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

// IsTerminal returns true if stdout is connected to a terminal.
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ShouldUseColor determines if ANSI color codes should be used based on
// the color mode setting and environment.
func ShouldUseColor(colorMode string) bool {
	switch colorMode {
	case "always":
		return true
	case "never":
		return false
	default:
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		return IsTerminal()
	}
}
