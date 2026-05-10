package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/thatsneat-dev/nprt/internal/core"
	"github.com/thatsneat-dev/nprt/internal/github"
	"github.com/thatsneat-dev/nprt/internal/render"
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

func TestRenderNetgraph_StagingPathWithHeadCommits(t *testing.T) {
	status := &core.PRStatus{
		Number:      476497,
		State:       core.PRStateMerged,
		BaseBranch:  "staging",
		MergeCommit: "abc123def456789012",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent, HeadCommit: "masterhead123456789"},
			{Name: "staging-next", Branch: "staging-next", Status: core.StatusPresent, HeadCommit: "staginghead123456"},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusNotPresent, HeadCommit: "unstablehead123456"},
		},
	}

	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	err := renderer.RenderNetgraph(status)
	if err != nil {
		t.Fatalf("RenderNetgraph returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"◆  abc123def456",
		"base staging",
		"●  staginghead1  staging-next",
		"●  masterhead12  master",
		"╰─○                nixos-unstable",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("RenderNetgraph output missing %q:\n%s", want, output)
		}
	}

	for _, unwanted := range []string{
		"NETGRAPH",
		"legend:",
		"channels",
		"──────────────▶",
		"◀╌",          // provenance arrows removed
		"unstablehead", // pending channel HEAD must not be shown
	} {
		if strings.Contains(output, unwanted) {
			t.Errorf("RenderNetgraph output should not contain %q:\n%s", unwanted, output)
		}
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
		RelatedPRs: []github.RelatedPR{
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
		RelatedPRs: []github.RelatedPR{
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

func TestRenderTable_SurfacesSharedChannelError(t *testing.T) {
	rateLimit := "GitHub API rate limit exceeded. Try again later or set GITHUB_TOKEN."
	status := &core.PRStatus{
		Number: 476497,
		State:  core.PRStateMerged,
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusUnknown, Error: rateLimit},
			{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable", Status: core.StatusUnknown, Error: rateLimit},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusUnknown, Error: rateLimit},
		},
	}
	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	if err := renderer.RenderTable(status); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, rateLimit) {
		t.Errorf("expected output to surface the shared error message, got:\n%s", out)
	}
	if !strings.Contains(out, "3 channels affected") {
		t.Errorf("expected output to include count of affected channels, got:\n%s", out)
	}
}

func TestRenderTable_SurfacesPerChannelErrors(t *testing.T) {
	status := &core.PRStatus{
		Number: 1,
		State:  core.PRStateMerged,
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusUnknown, Error: "context deadline exceeded"},
			{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable", Status: core.StatusUnknown, Error: "404 Not Found"},
		},
	}
	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	if err := renderer.RenderTable(status); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"channel checks reported errors:",
		"nixos-unstable: context deadline exceeded",
		"nixpkgs-unstable: 404 Not Found",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderTable_NoErrorsNoWarning(t *testing.T) {
	status := &core.PRStatus{
		Number: 1,
		State:  core.PRStateMerged,
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent},
		},
	}
	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	if err := renderer.RenderTable(status); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	if strings.Contains(buf.String(), "channel checks reported errors") {
		t.Errorf("did not expect error footer when no channel errors are present")
	}
}

func TestRenderNetgraph_NestedLineage_NixosUnstableUnderSmall(t *testing.T) {
	status := &core.PRStatus{
		Number:      1,
		State:       core.PRStateMerged,
		BaseBranch:  "master",
		MergeCommit: "deadbeefcafe",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent, HeadCommit: "deadbeefcafe"},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusPresent, HeadCommit: "fullhead12345"},
			{Name: "nixos-unstable-small", Branch: "nixos-unstable-small", Status: core.StatusPresent, HeadCommit: "smallhead1234"},
			{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable", Status: core.StatusPresent, HeadCommit: "pkgshead12345"},
			{Name: "staging-next", Branch: "staging-next", Status: core.StatusPresent, HeadCommit: "stghead12345"},
		},
	}
	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	if err := renderer.RenderNetgraph(status); err != nil {
		t.Fatalf("RenderNetgraph: %v", err)
	}
	out := buf.String()

	// nixos-unstable must appear as a nested child of nixos-unstable-small,
	// not as a direct child of master.
	for _, want := range []string{
		"●  smallhead123  nixos-unstable-small",
		"╰─●  fullhead1234  nixos-unstable",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q (lineage), got:\n%s", want, out)
		}
	}

	// nixos-unstable must NOT appear with a top-level divergence prefix.
	for _, unwanted := range []string{
		"│ ●  fullhead1234  nixos-unstable\n",
		"  ●  fullhead1234  nixos-unstable\n",
	} {
		if strings.Contains(out, unwanted) {
			t.Errorf("nixos-unstable should not be rendered as top-level fan-out: %q\n%s", unwanted, out)
		}
	}
}

func TestRenderNetgraph_NestedLineage_OrphanFallsBackToTopLevel(t *testing.T) {
	// nixos-unstable is checked but its parent (nixos-unstable-small) is not.
	// In that case, render nixos-unstable as a top-level fan-out off master
	// rather than dangling under a missing parent.
	status := &core.PRStatus{
		Number:      1,
		State:       core.PRStateMerged,
		BaseBranch:  "master",
		MergeCommit: "deadbeefcafe",
		Channels: []core.ChannelResult{
			{Name: "master", Branch: "master", Status: core.StatusPresent, HeadCommit: "deadbeefcafe"},
			{Name: "nixos-unstable", Branch: "nixos-unstable", Status: core.StatusPresent, HeadCommit: "fullhead12345"},
			{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable", Status: core.StatusPresent, HeadCommit: "pkgshead12345"},
		},
	}
	var buf bytes.Buffer
	renderer := render.NewRenderer(&buf, false, false)
	if err := renderer.RenderNetgraph(status); err != nil {
		t.Fatalf("RenderNetgraph: %v", err)
	}
	out := buf.String()
	// When the parent (nixos-unstable-small) isn't in the checked set,
	// nixos-unstable should appear as a top-level fan-out off master:
	// "  ├─●" or "  ╰─●" with no leading master-lane continuation char.
	if !strings.Contains(out, "\n  ├─●  fullhead1234  nixos-unstable") &&
		!strings.Contains(out, "\n  ╰─●  fullhead1234  nixos-unstable") {
		t.Errorf("nixos-unstable should fan out off master when parent is unchecked, got:\n%s", out)
	}
	// And it must NOT appear with a master-lane prefix indicating nesting.
	for _, nested := range []string{
		"\n  │ ╰─●  fullhead1234  nixos-unstable",
		"\n  │ ├─●  fullhead1234  nixos-unstable",
		"\n    ╰─●  fullhead1234  nixos-unstable",
		"\n    ├─●  fullhead1234  nixos-unstable",
	} {
		if strings.Contains(out, nested) {
			t.Errorf("nixos-unstable should not be rendered as nested when its parent is unchecked: %q\n%s", nested, out)
		}
	}
}
