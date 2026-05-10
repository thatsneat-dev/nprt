package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thatsneat-dev/nprt/internal/core"
)

// RenderTable outputs the PR status as a formatted ASCII table.
func (r *Renderer) RenderTable(status *core.PRStatus) error {
	r.writeErr = nil
	r.renderPRStatusLine(status)
	r.renderAuthorLine(status)
	r.println()

	maxNameLen := len("CHANNEL")
	for _, ch := range status.Channels {
		if len(ch.Name) > maxNameLen {
			maxNameLen = len(ch.Name)
		}
	}

	headerFmt := fmt.Sprintf("%%-%ds  STATUS\n", maxNameLen)
	r.printf(headerFmt, "CHANNEL")

	dividerLen := maxNameLen + 2 + 6
	r.println(strings.Repeat("-", dividerLen))

	rowFmt := fmt.Sprintf("%%-%ds  %%s\n", maxNameLen)
	for _, ch := range status.Channels {
		icon := r.formatChannelStatus(ch.Status)
		r.printf(rowFmt, ch.Name, fmt.Sprintf("  %s  ", icon))
	}

	r.renderChannelErrors(status)

	return r.writeErr
}

// renderChannelErrors surfaces any per-channel errors so users can see WHY a
// channel reported unknown status (rate limits, auth failures, network errors)
// rather than silently displaying "?" everywhere.
func (r *Renderer) renderChannelErrors(status *core.PRStatus) {
	type errEntry struct {
		channel string
		message string
	}
	var errs []errEntry
	seen := make(map[string]bool)
	for _, ch := range status.Channels {
		if ch.Error == "" {
			continue
		}
		errs = append(errs, errEntry{channel: ch.Name, message: ch.Error})
		seen[ch.Error] = true
	}
	if len(errs) == 0 {
		return
	}

	r.println()
	// If every error shares the same message, surface it once with the count
	// of affected channels rather than repeating it per row.
	if len(seen) == 1 {
		msg := errs[0].message
		r.printf("%s  %s (%d channel", r.warnIcon(), msg, len(errs))
		if len(errs) != 1 {
			r.printf("s")
		}
		r.println(" affected)")
		return
	}
	r.println(r.warnIcon() + "  channel checks reported errors:")
	for _, e := range errs {
		r.printf("    %s: %s\n", e.channel, e.message)
	}
}

func (r *Renderer) warnIcon() string {
	return r.colorize("!", colorYellow)
}

func (r *Renderer) renderPRStatusLine(status *core.PRStatus) {
	icon, stateColor := r.getPRStateIconAndColor(status.State)
	text := fmt.Sprintf("PR #%d", status.Number)
	url := fmt.Sprintf("https://github.com/NixOS/nixpkgs/pull/%d", status.Number)

	if status.Title != "" {
		text = fmt.Sprintf("%s (%s)", text, sanitize(status.Title))
	}

	displayText := r.formatHeadline(icon, stateColor, text, url)
	r.println(displayText)
}

func (r *Renderer) renderAuthorLine(status *core.PRStatus) {
	if status.Author != "" {
		if r.useColor {
			r.printf("%sby: %s%s\n", colorGray, sanitize(status.Author), colorReset)
		} else {
			r.printf("by: %s\n", sanitize(status.Author))
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

// RenderNetgraph outputs an ASCII graph of the PR's likely path through
// nixpkgs integration branches and selected channel branches.
func (r *Renderer) RenderNetgraph(status *core.PRStatus) error {
	r.writeErr = nil
	r.renderNetgraph(status)
	return r.writeErr
}

func (r *Renderer) renderNetgraph(status *core.PRStatus) {
	r.println()
	r.printf("  %s  %s", r.bold(graphMergeNode), r.commitLink(status.MergeCommit))
	if status.BaseBranch != "" {
		r.printf("  %s %s", r.dim("base"), sanitize(status.BaseBranch))
	}
	r.println()

	path := propagationPath(status.BaseBranch)
	results := channelResultsByBranch(status.Channels)

	// Mark all path-branch channels so they're rendered in the path section
	// rather than the channels fan-out.
	rendered := make(map[string]bool)
	for _, step := range path {
		if _, ok := results[step.Branch]; ok {
			rendered[step.Branch] = true
		}
	}
	remaining := remainingChannelResults(status.Channels, rendered)

	stem := r.dim(graphStem)

	// Path section: vertical chain of branches the commit flows through.
	for i := 1; i < len(path); i++ {
		current := path[i]
		result, hasResult := results[current.Branch]
		r.println("  " + stem)
		r.printf("  %s\n", r.formatPathRow(current, result, hasResult))
	}

	if len(remaining) == 0 {
		return
	}

	// Fan-out section: channels branch off the terminal path branch.
	for i, ch := range remaining {
		connector := graphBranchTee
		if i == len(remaining)-1 {
			connector = graphBranchLastTee
		}
		r.printf("  %s%s\n", r.dim(connector), r.formatChannelRow(ch))
	}
}

func (r *Renderer) formatPathRow(step propagationStep, result core.ChannelResult, hasResult bool) string {
	var glyph, commit string
	if !hasResult {
		glyph = r.colorize(graphPresentNode, colorGreen)
		commit = r.commitCell("", false)
	} else {
		glyph = r.graphNodeForStatus(result.Status)
		commit = r.commitCell(result.HeadCommit, result.Status == core.StatusPresent)
	}
	return fmt.Sprintf("%s  %s  %s", glyph, commit, step.Branch)
}

func (r *Renderer) formatChannelRow(ch core.ChannelResult) string {
	return fmt.Sprintf("%s  %s  %s",
		r.graphNodeForStatus(ch.Status),
		r.commitCell(ch.HeadCommit, ch.Status == core.StatusPresent),
		ch.Name)
}

// commitCell returns a fixed-width 12-char cell. When show is true and a
// commit is present, the short SHA is rendered (and OSC-8 linked when
// hyperlinks are enabled). Otherwise the cell is blank, preserving column
// alignment. Padding is appended after any escape sequences so the visible
// width is always 12.
func (r *Renderer) commitCell(commit string, show bool) string {
	pad := strings.Repeat(" ", 12)
	if !show || commit == "" {
		return pad
	}
	short := shortCommit(commit)
	trail := ""
	if n := 12 - len(short); n > 0 {
		trail = strings.Repeat(" ", n)
	}
	if !r.useHyperlinks {
		return short + trail
	}
	url := fmt.Sprintf("https://github.com/NixOS/nixpkgs/commit/%s", commit)
	return wrapHyperlink(short, url) + trail
}

// commitLink wraps the short SHA in an OSC 8 hyperlink without padding.
func (r *Renderer) commitLink(commit string) string {
	short := shortCommit(commit)
	if commit == "" || !r.useHyperlinks {
		return short
	}
	url := fmt.Sprintf("https://github.com/NixOS/nixpkgs/commit/%s", commit)
	return wrapHyperlink(short, url)
}

func (r *Renderer) graphNodeForStatus(status core.ChannelStatus) string {
	switch status {
	case core.StatusPresent:
		return r.colorize(graphPresentNode, colorGreen)
	case core.StatusNotPresent:
		return r.colorize(graphPendingNode, colorRed)
	default:
		return r.colorize(graphUnknownNode, colorYellow)
	}
}

func (r *Renderer) colorize(s, color string) string {
	if r.useColor {
		return color + s + colorReset
	}
	return s
}

func (r *Renderer) dim(s string) string {
	return r.colorize(s, colorGray)
}

func (r *Renderer) bold(s string) string {
	if r.useColor {
		return colorBold + s + colorReset
	}
	return s
}

type propagationStep struct {
	Branch string
	Edge   string
	Note   string
}

const (
	graphStem          = "│"
	graphBranchTee     = "├─"
	graphBranchLastTee = "╰─"
	graphMergeNode     = "◆"
	graphPresentNode   = "●"
	graphPendingNode   = "○"
	graphUnknownNode   = "?"
)

func propagationPath(baseBranch string) []propagationStep {
	switch {
	case baseBranch == "staging":
		return []propagationStep{
			{Branch: "staging", Edge: "merged to staging", Note: "mass rebuild batch"},
			{Branch: "staging-next", Edge: "manual staging batch", Note: "Hydra staging jobset"},
			{Branch: "master", Edge: "manual PR", Note: "unstable source"},
		}
	case strings.HasPrefix(baseBranch, "staging-") && !strings.HasPrefix(baseBranch, "staging-next-"):
		release := strings.TrimPrefix(baseBranch, "staging-")
		return []propagationStep{
			{Branch: baseBranch, Edge: "merged to release staging", Note: "stable mass rebuild batch"},
			{Branch: "staging-next-" + release, Edge: "manual staging batch", Note: "Hydra staging jobset"},
			{Branch: "release-" + release, Edge: "manual PR", Note: "stable channel source"},
		}
	case baseBranch == "staging-nixos":
		return []propagationStep{
			{Branch: "staging-nixos", Edge: "merged to staging-nixos", Note: "NixOS tests/kernel batch"},
			{Branch: "master", Edge: "manual PR", Note: "unstable source"},
		}
	case strings.HasPrefix(baseBranch, "staging-next"):
		return []propagationStep{
			{Branch: baseBranch, Edge: "merged to staging-next", Note: "staging fixup"},
			{Branch: destinationFromStagingNext(baseBranch), Edge: "manual PR", Note: "channel source"},
		}
	case strings.HasPrefix(baseBranch, "release-"):
		return []propagationStep{
			{Branch: baseBranch, Edge: "merged to release", Note: "stable channel source"},
		}
	default:
		branch := baseBranch
		if branch == "" {
			branch = "master"
		}
		return []propagationStep{
			{Branch: branch, Edge: "merged to target", Note: "channel source"},
		}
	}
}

func destinationFromStagingNext(branch string) string {
	release, ok := strings.CutPrefix(branch, "staging-next-")
	if ok {
		return "release-" + release
	}
	return "master"
}

func channelResultsByBranch(channels []core.ChannelResult) map[string]core.ChannelResult {
	results := make(map[string]core.ChannelResult, len(channels))
	for _, ch := range channels {
		results[ch.Branch] = ch
	}
	return results
}

func remainingChannelResults(channels []core.ChannelResult, rendered map[string]bool) []core.ChannelResult {
	remaining := make([]core.ChannelResult, 0, len(channels))
	for _, ch := range channels {
		if !rendered[ch.Branch] {
			remaining = append(remaining, ch)
		}
	}
	sort.SliceStable(remaining, func(i, j int) bool {
		return remaining[i].Name < remaining[j].Name
	})
	return remaining
}

func shortCommit(commit string) string {
	if commit == "" {
		return "?"
	}
	if len(commit) <= 12 {
		return commit
	}
	return commit[:12]
}
