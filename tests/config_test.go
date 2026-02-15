package tests

import (
	"os"
	"testing"

	"github.com/thatsneat-dev/nprt/internal/config"
)

func TestParsePRInput_ValidNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"476497", 476497},
		{"1", 1},
		{"123456789", 123456789},
	}

	for _, tc := range tests {
		result, err := config.ParsePRInput(tc.input)
		if err != nil {
			t.Errorf("ParsePRInput(%q) returned error: %v", tc.input, err)
		}
		if result != tc.expected {
			t.Errorf("ParsePRInput(%q) = %d, want %d", tc.input, result, tc.expected)
		}
	}
}

func TestParsePRInput_ValidURL(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"https://github.com/NixOS/nixpkgs/pull/476497", 476497},
		{"https://github.com/NixOS/nixpkgs/pull/1", 1},
		{"https://github.com/NixOS/nixpkgs/pull/476497/", 476497},
		{"https://github.com/NixOS/nixpkgs/issues/123", 123},
		{"https://github.com/NixOS/nixpkgs/issues/483584/", 483584},
	}

	for _, tc := range tests {
		result, err := config.ParsePRInput(tc.input)
		if err != nil {
			t.Errorf("ParsePRInput(%q) returned error: %v", tc.input, err)
		}
		if result != tc.expected {
			t.Errorf("ParsePRInput(%q) = %d, want %d", tc.input, result, tc.expected)
		}
	}
}

func TestParsePRInput_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"-1",
		"0",
		"https://github.com/other/repo/pull/123",
		"https://github.com/other/repo/issues/123",
		"github.com/NixOS/nixpkgs/pull/123",
	}

	for _, input := range tests {
		_, err := config.ParsePRInput(input)
		if err == nil {
			t.Errorf("ParsePRInput(%q) should have returned error", input)
		}
	}
}

func TestShouldUseColor_InvalidMode(t *testing.T) {
	invalidModes := []string{"invalid", "yes", "true", "on"}
	for _, mode := range invalidModes {
		_, err := config.ShouldUseColor(mode)
		if err == nil {
			t.Errorf("ShouldUseColor(%q) should have returned error", mode)
		}
	}
}

func TestShouldUseColor_ValidModes(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"always", true},
		{"never", false},
	}
	for _, tc := range tests {
		result, err := config.ShouldUseColor(tc.mode)
		if err != nil {
			t.Errorf("ShouldUseColor(%q) returned error: %v", tc.mode, err)
		}
		if result != tc.expected {
			t.Errorf("ShouldUseColor(%q) = %v, want %v", tc.mode, result, tc.expected)
		}
	}
}

func TestShouldUseHyperlinks_InvalidMode(t *testing.T) {
	invalidModes := []string{"invalid", "yes", "true", "on"}
	for _, mode := range invalidModes {
		_, err := config.ShouldUseHyperlinks(mode)
		if err == nil {
			t.Errorf("ShouldUseHyperlinks(%q) should have returned error", mode)
		}
	}
}

func TestShouldUseHyperlinks_ValidModes(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"always", true},
		{"never", false},
	}
	for _, tc := range tests {
		result, err := config.ShouldUseHyperlinks(tc.mode)
		if err != nil {
			t.Errorf("ShouldUseHyperlinks(%q) returned error: %v", tc.mode, err)
		}
		if result != tc.expected {
			t.Errorf("ShouldUseHyperlinks(%q) = %v, want %v", tc.mode, result, tc.expected)
		}
	}
}

func TestShouldUseHyperlinks_NoHyperlinksEnv(t *testing.T) {
	t.Setenv("NO_HYPERLINKS", "1")
	result, err := config.ShouldUseHyperlinks("auto")
	if err != nil {
		t.Fatalf("ShouldUseHyperlinks returned error: %v", err)
	}
	if result {
		t.Error("ShouldUseHyperlinks should return false when NO_HYPERLINKS is set")
	}
}

func TestShouldUseHyperlinksForFile_NoHyperlinksEnv(t *testing.T) {
	t.Setenv("NO_HYPERLINKS", "1")
	result := config.ShouldUseHyperlinksForFile("auto", os.Stdout)
	if result {
		t.Error("ShouldUseHyperlinksForFile should return false when NO_HYPERLINKS is set")
	}
}

func TestShouldUseHyperlinksForFile_AlwaysMode(t *testing.T) {
	result := config.ShouldUseHyperlinksForFile("always", os.Stdout)
	if !result {
		t.Error("ShouldUseHyperlinksForFile(always) should return true")
	}
}

func TestShouldUseHyperlinksForFile_NeverMode(t *testing.T) {
	result := config.ShouldUseHyperlinksForFile("never", os.Stdout)
	if result {
		t.Error("ShouldUseHyperlinksForFile(never) should return false")
	}
}

func TestShouldUseHyperlinks_IgnoresNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result, err := config.ShouldUseHyperlinks("always")
	if err != nil {
		t.Fatalf("ShouldUseHyperlinks returned error: %v", err)
	}
	if !result {
		t.Error("ShouldUseHyperlinks(always) should not be affected by NO_COLOR")
	}
}

func TestParseChannels_Default(t *testing.T) {
	channels, err := config.ParseChannels("")
	if err != nil {
		t.Fatalf("ParseChannels(\"\") returned error: %v", err)
	}
	if len(channels) != len(config.GetDefaultChannels()) {
		t.Errorf("ParseChannels(\"\") returned %d channels, want %d", len(channels), len(config.GetDefaultChannels()))
	}
}

func TestParseChannels_Custom(t *testing.T) {
	channels, err := config.ParseChannels("master,nixos-unstable")
	if err != nil {
		t.Fatalf("ParseChannels returned error: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("ParseChannels returned %d channels, want 2", len(channels))
	}
	if channels[0].Name != "master" {
		t.Errorf("First channel should be master, got %s", channels[0].Name)
	}
	if channels[1].Name != "nixos-unstable" {
		t.Errorf("Second channel should be nixos-unstable, got %s", channels[1].Name)
	}
}

func TestParseChannels_InvalidChannel(t *testing.T) {
	_, err := config.ParseChannels("nonexistent-channel")
	if err == nil {
		t.Error("ParseChannels should return error for invalid channel")
	}
}

func TestParseChannels_Whitespace(t *testing.T) {
	channels, err := config.ParseChannels(" master , nixos-unstable ")
	if err != nil {
		t.Fatalf("ParseChannels returned error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("ParseChannels returned %d channels, want 2", len(channels))
	}
}
