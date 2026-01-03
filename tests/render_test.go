package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/core"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/render"
)

func TestRenderTable_NoColor(t *testing.T) {
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
	if !strings.Contains(output, "\uf407") && !strings.Contains(output, "●") {
		t.Error("Output should contain PR state icon")
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
	if !strings.Contains(output, "\033[32m") {
		t.Error("Output should contain green ANSI color code")
	}
	if !strings.Contains(output, "\033[35m") {
		t.Error("Output should contain purple ANSI color code for merged PR status line")
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
