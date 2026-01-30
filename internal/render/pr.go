package render

import (
	"fmt"
	"strings"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/core"
)

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

	if status.Title != "" {
		text = fmt.Sprintf("%s (%s)", text, status.Title)
	}

	displayText := r.formatHeadline(icon, stateColor, text, url)
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
		icon = r.pickIcon(nfIconPRDraft, fallbackIcon)
	case core.PRStateOpen:
		color = colorGreen
		icon = r.pickIcon(nfIconPROpen, fallbackIcon)
	case core.PRStateMerged:
		color = colorPurple
		icon = r.pickIcon(nfIconPRMerged, fallbackIcon)
	case core.PRStateClosed:
		color = colorRed
		icon = r.pickIcon(nfIconPRClosed, fallbackIcon)
	default:
		color = colorYellow
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
