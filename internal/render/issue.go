package render

import (
	"fmt"

	"github.com/thatsneat-dev/nprt/internal/github"
)

// IssueWarning contains information for formatting an issue-not-PR warning.
type IssueWarning struct {
	Number int
	Title  string
	// State is "open", "closed", or "draft"
	State      string
	URL        string
	RelatedPRs []github.RelatedPR
}

// RenderIssueWarning outputs a warning that the input was an issue, not a PR.
// It renders the issue with appropriate icons/colors and lists related PRs in a table.
func (r *Renderer) RenderIssueWarning(info IssueWarning) error {
	r.writeErr = nil
	r.renderWarningLine()
	r.renderIssueLine(info)
	r.println()

	if len(info.RelatedPRs) > 0 {
		r.renderRelatedPRsTable(info.RelatedPRs)
	}

	return r.writeErr
}

func (r *Renderer) renderWarningLine() {
	msg := "WARNING: input is an issue, not a pull request"
	if r.useColor {
		r.println(colorRed + msg + colorReset)
	} else {
		r.println(msg)
	}
}

func (r *Renderer) renderIssueLine(info IssueWarning) {
	icon, stateColor := r.getIssueStateIconAndColor(info.State)
	text := fmt.Sprintf("Issue #%d", info.Number)

	if info.Title != "" {
		text = fmt.Sprintf("%s (%s)", text, sanitize(info.Title))
	}

	displayText := r.formatHeadline(icon, stateColor, text, info.URL)
	r.println(displayText)
}

// getIssueStateIconAndColor returns the icon and color for a given issue state.
func (r *Renderer) getIssueStateIconAndColor(state string) (icon, color string) {
	switch state {
	case "open":
		color = colorGreen
		icon = r.pickIcon(nfIconIssueOpen, fallbackIcon)
	case "closed":
		color = colorPurple
		icon = r.pickIcon(nfIconIssueClosed, fallbackIcon)
	case "draft":
		color = colorGray
		icon = r.pickIcon(nfIconIssueDraft, fallbackIcon)
	default:
		color = colorYellow
		icon = fallbackIcon
	}
	return icon, color
}

func (r *Renderer) renderRelatedPRsTable(prs []github.RelatedPR) {
	r.println("Related pull requests:")
	r.println()

	maxNumLen := 2
	for _, pr := range prs {
		numStr := fmt.Sprintf("#%d", pr.Number)
		if len(numStr) > maxNumLen {
			maxNumLen = len(numStr)
		}
	}

	for _, pr := range prs {
		icon, stateColor := r.getPRStateFromString(pr.State)
		numStr := fmt.Sprintf("#%d", pr.Number)

		var content string
		if r.useColor {
			iconDisplay := stateColor + icon + colorReset
			numDisplay := fmt.Sprintf("%s%-*s%s", colorBold, maxNumLen, numStr, colorReset)
			content = fmt.Sprintf("%s  %s  %s", iconDisplay, numDisplay, sanitize(pr.Title))
		} else {
			numDisplay := fmt.Sprintf("%-*s", maxNumLen, numStr)
			content = fmt.Sprintf("%s  %s  %s", icon, numDisplay, sanitize(pr.Title))
		}

		if r.useHyperlinks && pr.URL != "" {
			content = wrapHyperlink(content, pr.URL)
		}

		r.printf("  %s\n", content)
	}
}

// getPRStateFromString maps a string state to icon and color.
func (r *Renderer) getPRStateFromString(state string) (icon, color string) {
	switch state {
	case "open":
		color = colorGreen
		icon = r.pickIcon(nfIconPROpen, fallbackIcon)
	case "closed":
		color = colorRed
		icon = r.pickIcon(nfIconPRClosed, fallbackIcon)
	case "merged":
		color = colorPurple
		icon = r.pickIcon(nfIconPRMerged, fallbackIcon)
	case "draft":
		color = colorGray
		icon = r.pickIcon(nfIconPRDraft, fallbackIcon)
	default:
		color = colorYellow
		icon = fallbackIcon
	}
	return icon, color
}
