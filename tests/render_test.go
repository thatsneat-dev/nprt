package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/core"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/render"
)

func TestRenderTable_NoColor_WithNerdFonts(t *testing.T) {
	// Ensure Nerd Fonts are enabled for this test
	t.Setenv("NO_NERD_FONTS", "")

	status := &core.PRStatus{
		Number:      476497,
		State:       core.PRStateMerged,
		MergeCommit: "abc123",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusNotPresent},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	err := renderer.RenderTable(status)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "PR #476497") {
		t.Error("Output should contain PR status line")
	}
	// Merged PR should show merged icon (\uf419), not open icon
	if !strings.Contains(output, "\uf419") {
		t.Error("Merged PR should contain merged icon (\\uf419)")
	}
	if !strings.Contains(output, "CHANNEL") {
		t.Error("Output should contain 'CHANNEL' header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("Output should contain 'STATUS' header")
	}
	if !strings.Contains(output, "master") {
		t.Error("Output should contain 'master'")
	}
	if !strings.Contains(output, "nixos-unstable") {
		t.Error("Output should contain 'nixos-unstable'")
	}
	if !strings.Contains(output, "✓") {
		t.Error("Output should contain check mark")
	}
	if !strings.Contains(output, "✗") {
		t.Error("Output should contain X mark")
	}
}

func TestRenderTable_NoColor_NoNerdFonts(t *testing.T) {
	// Disable Nerd Fonts for deterministic fallback icon
	t.Setenv("NO_NERD_FONTS", "1")

	status := &core.PRStatus{
		Number:      476497,
		State:       core.PRStateMerged,
		MergeCommit: "abc123",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	err := renderer.RenderTable(status)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "●") {
		t.Error("Output should contain fallback dot icon when NO_NERD_FONTS=1")
	}
	// Should NOT contain any Nerd Font icons
	if strings.Contains(output, "\uf419") || strings.Contains(output, "\uf407") {
		t.Error("Output should not contain Nerd Font icons when NO_NERD_FONTS=1")
	}
}

func TestRenderTable_WithColor(t *testing.T) {
	status := &core.PRStatus{
		Number:      476497,
		State:       core.PRStateMerged,
		MergeCommit: "abc123",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, true, false)
	err := renderer.RenderTable(status)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\033[38;5;10m") {
		t.Error("Output should contain green palette color code")
	}
	if !strings.Contains(output, "\033[38;5;13m") {
		t.Error("Output should contain purple palette color code for merged PR status line")
	}
}

func TestRenderTable_WithHyperlinks(t *testing.T) {
	status := &core.PRStatus{
		Number:      476497,
		State:       core.PRStateMerged,
		MergeCommit: "abc123",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, true)
	err := renderer.RenderTable(status)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\033]8;;https://github.com/NixOS/nixpkgs/pull/476497\033\\") {
		t.Error("Output should contain OSC 8 hyperlink start sequence")
	}
	if !strings.Contains(output, "\033]8;;\033\\") {
		t.Error("Output should contain OSC 8 hyperlink end sequence")
	}
}

func TestRenderJSON(t *testing.T) {
	status := &core.PRStatus{
		Number:      476497,
		State:       core.PRStateMerged,
		MergeCommit: "abc123def456",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusNotPresent},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	err := renderer.RenderJSON(status)
	if err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if result["pr"].(float64) != 476497 {
		t.Errorf("PR number = %v, want 476497", result["pr"])
	}
	if result["state"] != "merged" {
		t.Errorf("state = %v, want 'merged'", result["state"])
	}
	if result["merge_commit"] != "abc123def456" {
		t.Errorf("merge_commit = %v, want 'abc123def456'", result["merge_commit"])
	}

	channels := result["channels"].([]interface{})
	if len(channels) != 2 {
		t.Errorf("channels length = %d, want 2", len(channels))
	}
}

func TestRenderTable_UnknownStatus(t *testing.T) {
	status := &core.PRStatus{
		Number:      123,
		State:       core.PRStateMerged,
		MergeCommit: "abc",
		Channels: []core.ChannelResult{
			{Name: "unknown-branch", Branch: "unknown-branch", Status: core.StatusUnknown},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	err := renderer.RenderTable(status)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "?") {
		t.Error("Output should contain '?' for unknown status")
	}
}

func TestFormatError_WithColor(t *testing.T) {
	msg := "something went wrong"
	result := render.FormatError(msg, true)

	if !strings.Contains(result, "Error:") {
		t.Error("FormatError should contain 'Error:' prefix")
	}
	if !strings.Contains(result, msg) {
		t.Error("FormatError should contain the original message")
	}
	if !strings.Contains(result, "\033[") {
		t.Error("FormatError with color should contain ANSI codes")
	}
}

func TestFormatError_WithoutColor(t *testing.T) {
	msg := "something went wrong"
	result := render.FormatError(msg, false)

	expected := "Error: something went wrong"
	if result != expected {
		t.Errorf("FormatError = %q, want %q", result, expected)
	}
	if strings.Contains(result, "\033[") {
		t.Error("FormatError without color should not contain ANSI codes")
	}
}

func TestRenderIssueWarning_WithHyperlinks(t *testing.T) {
	info := render.IssueWarning{
		Number: 12345,
		Title:  "Test issue",
		State:  "open",
		URL:    "https://github.com/NixOS/nixpkgs/issues/12345",
		RelatedPRs: []render.RelatedPR{
			{
				Number: 67890,
				Title:  "Fix for issue",
				URL:    "https://github.com/NixOS/nixpkgs/pull/67890",
				State:  "open",
			},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, true)
	err := renderer.RenderIssueWarning(info)
	if err != nil {
		t.Fatalf("RenderIssueWarning returned error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "\033]8;;") {
		t.Error("Output should contain OSC 8 hyperlink start sequence")
	}
	if !strings.Contains(result, "\033]8;;\033\\") {
		t.Error("Output should contain OSC 8 hyperlink end sequence")
	}
	if !strings.Contains(result, "https://github.com/NixOS/nixpkgs/issues/12345") {
		t.Error("Output should contain issue URL")
	}
	if !strings.Contains(result, "https://github.com/NixOS/nixpkgs/pull/67890") {
		t.Error("Output should contain PR URL")
	}
	if !strings.Contains(result, "Related pull requests:") {
		t.Error("Output should contain related PRs header")
	}
	if !strings.Contains(result, "WARNING:") {
		t.Error("Output should contain WARNING prefix")
	}
}

func TestRenderIssueWarning_WithoutHyperlinks(t *testing.T) {
	info := render.IssueWarning{
		Number: 12345,
		Title:  "Test issue",
		State:  "open",
		URL:    "https://github.com/NixOS/nixpkgs/issues/12345",
		RelatedPRs: []render.RelatedPR{
			{
				Number: 67890,
				Title:  "Fix for issue",
				URL:    "https://github.com/NixOS/nixpkgs/pull/67890",
				State:  "open",
			},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	err := renderer.RenderIssueWarning(info)
	if err != nil {
		t.Fatalf("RenderIssueWarning returned error: %v", err)
	}

	result := buf.String()
	if strings.Contains(result, "\033]8;;") {
		t.Error("Output should not contain OSC 8 hyperlink sequences when disabled")
	}
	if !strings.Contains(result, "#12345") {
		t.Error("Output should contain issue number")
	}
	if !strings.Contains(result, "#67890") {
		t.Error("Output should contain related PR number")
	}
}

func TestRenderIssueWarning_IssueStates(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"open", "Issue #123"},
		{"closed", "Issue #123"},
	}

	for _, tc := range tests {
		info := render.IssueWarning{
			Number: 123,
			Title:  "Test",
			State:  tc.state,
			URL:    "https://github.com/NixOS/nixpkgs/issues/123",
		}

		var buf bytes.Buffer
		renderer := render.NewRenderer(&buf, false, false)
		_ = renderer.RenderIssueWarning(info)

		result := buf.String()
		if !strings.Contains(result, tc.expected) {
			t.Errorf("state=%s: expected output to contain %q, got: %s", tc.state, tc.expected, result)
		}
	}
}
