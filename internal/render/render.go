// Package render handles formatting and output of PR status results
// in table and JSON formats with optional ANSI colors.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/core"
)

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"

	// 256-color palette indices (0-15) for theme compatibility
	colorGray   = "\033[38;5;8m"  // palette 8: subdued/secondary (draft, author)
	colorGreen  = "\033[38;5;10m" // palette 10: success/active (open, present)
	colorBlue   = "\033[38;5;12m" // palette 12: info/completed (merged)
	colorRed    = "\033[38;5;9m"  // palette 9: error/negative (closed, not present)
	colorYellow = "\033[38;5;11m" // palette 11: warning/unknown

	iconPresent    = "✓"
	iconNotPresent = "✗"
	iconUnknown    = "?"

	// Nerd Font icons
	nfIconDraft  = "\uf4dd"
	nfIconOpen   = "\uf407"
	nfIconMerged = "\uf407"
	nfIconClosed = "\uf4dc"

	// Fallback dot icon
	fallbackIcon = "●"
)

// Renderer outputs PR status in various formats.
type Renderer struct {
	useColor      bool
	useHyperlinks bool
	useNerdFonts  bool
	writer        io.Writer
}

// NewRenderer creates a new Renderer with the given output settings.
// Nerd Font icons are enabled by default; set NO_NERD_FONTS=1 to disable.
func NewRenderer(writer io.Writer, useColor bool, useHyperlinks bool) *Renderer {
	return &Renderer{
		useColor:      useColor,
		useHyperlinks: useHyperlinks,
		useNerdFonts:  os.Getenv("NO_NERD_FONTS") == "",
		writer:        writer,
	}
}

// RenderTable outputs the PR status as a formatted ASCII table.
func (r *Renderer) RenderTable(status *core.PRStatus) error {
	r.renderPRStatusLine(status)
	r.renderAuthorLine(status)
	fmt.Fprintln(r.writer)

	maxNameLen := len("CHANNEL")
	for _, ch := range status.Channels {
		if len(ch.Name) > maxNameLen {
			maxNameLen = len(ch.Name)
		}
	}

	headerFmt := fmt.Sprintf("%%-%ds  STATUS\n", maxNameLen)
	fmt.Fprintf(r.writer, headerFmt, "CHANNEL")

	dividerLen := maxNameLen + 2 + 6
	fmt.Fprintln(r.writer, strings.Repeat("-", dividerLen))

	rowFmt := fmt.Sprintf("%%-%ds  %%s\n", maxNameLen)
	for _, ch := range status.Channels {
		icon := r.formatChannelStatus(ch.Status)
		fmt.Fprintf(r.writer, rowFmt, ch.Name, fmt.Sprintf("  %s  ", icon))
	}

	return nil
}

func (r *Renderer) renderPRStatusLine(status *core.PRStatus) {
	icon, stateColor := r.getPRStateIconAndColor(status.State)
	text := fmt.Sprintf("PR #%d", status.Number)
	url := fmt.Sprintf("https://github.com/NixOS/nixpkgs/pull/%d", status.Number)

	// Add title in parentheses if available
	if status.Title != "" {
		text = fmt.Sprintf("%s (%s)", text, status.Title)
	}

	var displayText string
	if r.useColor {
		// Color the entire line (icon + text) with the state color
		displayText = fmt.Sprintf("%s%s %s%s%s", stateColor, icon, colorBold, text, colorReset)
	} else {
		displayText = fmt.Sprintf("%s %s", icon, text)
	}

	if r.useHyperlinks {
		// Wrap the entire display in a hyperlink
		displayText = fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, displayText)
	}

	fmt.Fprintln(r.writer, displayText)
}

func (r *Renderer) renderAuthorLine(status *core.PRStatus) {
	if status.Author != "" {
		if r.useColor {
			fmt.Fprintf(r.writer, "%sby: %s%s\n", colorGray, status.Author, colorReset)
		} else {
			fmt.Fprintf(r.writer, "by: %s\n", status.Author)
		}
	}
}

// getPRStateIconAndColor returns the icon and color for a given PR state.
func (r *Renderer) getPRStateIconAndColor(state core.PRState) (icon, color string) {
	switch state {
	case core.PRStateDraft:
		color = colorGray
		if r.useNerdFonts {
			icon = nfIconDraft
		} else {
			icon = fallbackIcon
		}
	case core.PRStateOpen:
		color = colorGreen
		if r.useNerdFonts {
			icon = nfIconOpen
		} else {
			icon = fallbackIcon
		}
	case core.PRStateMerged:
		color = colorBlue
		if r.useNerdFonts {
			icon = nfIconMerged
		} else {
			icon = fallbackIcon
		}
	case core.PRStateClosed:
		color = colorRed
		if r.useNerdFonts {
			icon = nfIconClosed
		} else {
			icon = fallbackIcon
		}
	default:
		color = colorGreen
		icon = fallbackIcon
	}
	return icon, color
}

func (r *Renderer) formatChannelStatus(status core.ChannelStatus) string {
	switch status {
	case core.StatusPresent:
		if r.useColor {
			return colorGreen + iconPresent + colorReset
		}
		return iconPresent
	case core.StatusNotPresent:
		if r.useColor {
			return colorRed + iconNotPresent + colorReset
		}
		return iconNotPresent
	default:
		if r.useColor {
			return colorYellow + iconUnknown + colorReset
		}
		return iconUnknown
	}
}

// RenderJSON outputs the PR status as pretty-printed JSON.
func (r *Renderer) RenderJSON(status *core.PRStatus) error {
	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}
