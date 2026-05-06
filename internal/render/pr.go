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

	return r.writeErr
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
	r.println("NETGRAPH")
	r.printf("  %s PR merge %s", graphMergeNode, shortCommit(status.MergeCommit))
	if status.BaseBranch != "" {
		r.printf(" (base: %s)", sanitize(status.BaseBranch))
	}
	r.println()
	r.println("  legend: ● present  ○ pending  ? unknown")
	r.println()

	path := propagationPath(status.BaseBranch)
	lanes := graphLanes(path)
	r.renderGraphHeader(lanes)

	results := channelResultsByBranch(status.Channels)
	rendered := make(map[string]bool)

	r.renderGraphRow(lanes, map[string]string{
		path[0].Branch: graphMergeNode + " " + shortCommit(status.MergeCommit),
	}, "PR merge")

	for i := 1; i < len(path); i++ {
		previous := path[i-1]
		current := path[i]
		cells := map[string]string{
			previous.Branch: graphMergeEdge,
			current.Branch:  r.formatGraphStepNode(current, results[current.Branch]),
		}
		if _, ok := results[current.Branch]; ok {
			rendered[current.Branch] = true
		}
		r.renderGraphRow(lanes, cells, current.Edge)
	}

	if _, ok := results[path[0].Branch]; ok {
		rendered[path[0].Branch] = true
	}

	remaining := remainingChannelResults(status.Channels, rendered)
	if len(remaining) == 0 {
		return
	}

	source := path[len(path)-1].Branch
	for i, result := range remaining {
		edge := graphMergeEdge
		if i == len(remaining)-1 {
			edge = graphLastMergeEdge
		}
		r.renderGraphRow(lanes, map[string]string{
			source:           edge,
			graphChannelLane: r.formatGraphChannelNode(result),
		}, result.Name)
	}
}

func graphLanes(path []propagationStep) []string {
	lanes := make([]string, 0, len(path)+1)
	seen := make(map[string]bool, len(path)+1)
	for _, step := range path {
		if !seen[step.Branch] {
			lanes = append(lanes, step.Branch)
			seen[step.Branch] = true
		}
	}
	lanes = append(lanes, graphChannelLane)
	return lanes
}

func (r *Renderer) renderGraphHeader(lanes []string) {
	cells := make(map[string]string, len(lanes))
	for _, lane := range lanes {
		cells[lane] = lane
	}
	r.renderGraphRow(lanes, cells, "")
	r.renderGraphDivider(lanes)
}

func (r *Renderer) renderGraphDivider(lanes []string) {
	r.printf("  ")
	for i := range lanes {
		if i > 0 {
			r.printf("  ")
		}
		r.printf("%s", strings.Repeat("─", graphCellWidth))
	}
	r.println()
}

func (r *Renderer) renderGraphRow(lanes []string, cells map[string]string, note string) {
	r.printf("  ")
	for i, lane := range lanes {
		if i > 0 {
			r.printf("  ")
		}
		cell := cells[lane]
		if cell == "" {
			cell = graphLaneLine
		}
		r.printf("%s", padGraphCell(cell))
	}
	if note != "" {
		r.printf("  %s", sanitize(note))
	}
	r.println()
}

func (r *Renderer) formatGraphStepNode(step propagationStep, result core.ChannelResult) string {
	if result.Name == "" {
		return graphPresentNode + " " + step.Branch
	}
	return r.formatGraphChannelNode(result)
}

func (r *Renderer) formatGraphChannelNode(ch core.ChannelResult) string {
	parts := []string{r.graphNodeForStatus(ch.Status)}
	if ch.HeadCommit != "" {
		parts = append(parts, shortCommit(ch.HeadCommit))
	} else {
		parts = append(parts, "?")
	}
	parts = append(parts, r.formatChannelStatus(ch.Status))
	return strings.Join(parts, " ")
}

func (r *Renderer) graphNodeForStatus(status core.ChannelStatus) string {
	node := graphUnknownNode
	color := colorYellow
	switch status {
	case core.StatusPresent:
		node = graphPresentNode
		color = colorGreen
	case core.StatusNotPresent:
		node = graphPendingNode
		color = colorRed
	}
	if r.useColor {
		return color + node + colorReset
	}
	return node
}

func padGraphCell(cell string) string {
	visible := visibleWidth(cell)
	if visible >= graphCellWidth {
		return cell
	}
	return cell + strings.Repeat(" ", graphCellWidth-visible)
}

func visibleWidth(s string) int {
	width := 0
	inEscape := false
	for i := 0; i < len(s); i++ {
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		width++
	}
	return width
}

type propagationStep struct {
	Branch string
	Edge   string
	Note   string
}

const (
	graphCellWidth     = 22
	graphChannelLane   = "channels"
	graphLaneLine      = "│"
	graphMergeEdge     = "├──────────────▶"
	graphLastMergeEdge = "└──────────────▶"
	graphMergeNode     = "◆"
	graphPendingNode   = "○"
	graphPresentNode   = "●"
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
