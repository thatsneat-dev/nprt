package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/thatsneat-dev/nprt/internal/core"
)

func TestPropagationPath(t *testing.T) {
	tests := []struct {
		name string
		base string
		want []string
	}{
		{name: "empty defaults to master", base: "", want: []string{"master"}},
		{name: "master direct", base: "master", want: []string{"master"}},
		{name: "staging mass-rebuild", base: "staging", want: []string{"staging", "staging-next", "master"}},
		{
			name: "release staging is stable mass-rebuild",
			base: "staging-25.05",
			want: []string{"staging-25.05", "staging-next-25.05", "release-25.05"},
		},
		{
			name: "staging-nixos back-merges to master (regression: must not match generic staging-*)",
			base: "staging-nixos",
			want: []string{"staging-nixos", "master"},
		},
		{
			name: "staging-next forwards to master",
			base: "staging-next",
			want: []string{"staging-next", "master"},
		},
		{
			name: "release staging-next forwards to release-XX",
			base: "staging-next-25.05",
			want: []string{"staging-next-25.05", "release-25.05"},
		},
		{
			name: "release branch terminates at itself",
			base: "release-25.05",
			want: []string{"release-25.05"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := propagationPath(tc.base)
			if len(got) != len(tc.want) {
				t.Fatalf("propagationPath(%q) length = %d, want %d (%v)", tc.base, len(got), len(tc.want), got)
			}
			for i, step := range got {
				if step.Branch != tc.want[i] {
					t.Errorf("propagationPath(%q)[%d] = %q, want %q", tc.base, i, step.Branch, tc.want[i])
				}
			}
		})
	}
}

func TestChannelParent(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		// Unstable lineage
		{name: "nixpkgs-unstable", want: "master"},
		{name: "nixos-unstable-small", want: "master"},
		{name: "staging-next", want: "master"},
		{name: "nixos-unstable", want: "nixos-unstable-small"},
		// Release lineage
		{name: "nixpkgs-25.05", want: "release-25.05"},
		{name: "staging-next-25.05", want: "release-25.05"},
		{name: "nixos-25.05-small", want: "release-25.05"},
		{name: "nixos-25.05", want: "nixos-25.05-small"},
		// No special parent
		{name: "master", want: ""},
		{name: "release-25.05", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := channelParent(tc.name)
			if got != tc.want {
				t.Errorf("channelParent(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestCommitCell(t *testing.T) {
	r := NewRenderer(&bytes.Buffer{}, false, false)

	t.Run("hidden produces 12 spaces", func(t *testing.T) {
		got := r.commitCell("abc123def456789", false)
		if got != strings.Repeat(" ", 12) {
			t.Errorf("hidden cell = %q, want 12 spaces", got)
		}
	})

	t.Run("empty commit produces 12 spaces even when shown", func(t *testing.T) {
		got := r.commitCell("", true)
		if got != strings.Repeat(" ", 12) {
			t.Errorf("empty cell = %q, want 12 spaces", got)
		}
	})

	t.Run("shown SHA is 12 visible chars without hyperlinks", func(t *testing.T) {
		got := r.commitCell("abc123def456789", true)
		if got != "abc123def456" {
			t.Errorf("plain cell = %q, want %q", got, "abc123def456")
		}
	})

	t.Run("hyperlink wraps only the SHA, padding stays outside", func(t *testing.T) {
		hr := NewRenderer(&bytes.Buffer{}, false, true)
		// Use a commit shorter than 12 chars to verify trailing pad is OUTSIDE the escape.
		got := hr.commitCell("deadbeef", true)
		if !strings.Contains(got, "\033]8;;https://github.com/NixOS/nixpkgs/commit/deadbeef\033\\deadbeef\033]8;;\033\\") {
			t.Errorf("missing wrapped SHA in %q", got)
		}
		if !strings.HasSuffix(got, "    ") {
			t.Errorf("expected trailing padding outside hyperlink, got %q", got)
		}
	})
}

func TestRenderChannelErrors_SharedSingularPluralization(t *testing.T) {
	tests := []struct {
		name     string
		channels []core.ChannelResult
		want     string
	}{
		{
			name: "single channel reports singular",
			channels: []core.ChannelResult{
				{Name: "master", Branch: "master", Status: core.StatusUnknown, Error: "boom"},
			},
			want: "(1 channel affected)",
		},
		{
			name: "multiple channels with same error report plural",
			channels: []core.ChannelResult{
				{Name: "master", Branch: "master", Status: core.StatusUnknown, Error: "boom"},
				{Name: "nixpkgs-unstable", Branch: "nixpkgs-unstable", Status: core.StatusUnknown, Error: "boom"},
			},
			want: "(2 channels affected)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			r := NewRenderer(&buf, false, false)
			r.renderChannelErrors(&core.PRStatus{Channels: tc.channels})
			if !strings.Contains(buf.String(), tc.want) {
				t.Errorf("expected %q in output, got:\n%s", tc.want, buf.String())
			}
		})
	}
}
