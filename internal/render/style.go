package render

import "fmt"

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"

	// 256-color palette indices (0-15) for theme compatibility
	colorGray   = "\033[38;5;8m"  // palette 8: subdued/secondary (draft, author)
	colorGreen  = "\033[38;5;10m" // palette 10: success/active (open, present)
	colorBlue   = "\033[38;5;12m" // palette 12: info/completed
	colorRed    = "\033[38;5;9m"  // palette 9: error/negative (closed, not present)
	colorYellow = "\033[38;5;11m" // palette 11: warning/unknown
	colorPurple = "\033[38;5;13m" // palette 13: merged PRs, closed issues

	iconPresent    = "✓"
	iconNotPresent = "✗"
	iconUnknown    = "?"

	// Nerd Font icons for PRs (from nf-oct-* Octicons set)
	nfIconPRDraft  = "\uf4dd" // nf-oct-git_pull_request_draft
	nfIconPROpen   = "\uf407" // nf-oct-git_pull_request
	nfIconPRMerged = "\uf419" // nf-oct-git_merge
	nfIconPRClosed = "\uf4dc" // nf-oct-git_pull_request_closed

	// Nerd Font icons for issues (from nf-oct-* Octicons set)
	nfIconIssueOpen   = "\uf41b" // nf-oct-issue_opened
	nfIconIssueClosed = "\uf41d" // nf-oct-issue_closed
	nfIconIssueDraft  = "\uf4e7" // nf-oct-issue_draft

	// Fallback dot icon
	fallbackIcon = "●"
)

// pickIcon returns nerd font icon if enabled, otherwise the fallback.
func (r *Renderer) pickIcon(nerd, fallback string) string {
	if r.useNerdFonts {
		return nerd
	}
	return fallback
}

// wrapHyperlink wraps text in an OSC 8 hyperlink escape sequence.
func wrapHyperlink(text, url string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// formatHeadline formats "colored icon + bold text" pattern.
func (r *Renderer) formatHeadline(icon, stateColor, text, url string) string {
	var displayText string
	if r.useColor {
		displayText = fmt.Sprintf("%s%s %s%s%s", stateColor, icon, colorBold, text, colorReset)
	} else {
		displayText = fmt.Sprintf("%s %s", icon, text)
	}

	if r.useHyperlinks && url != "" {
		displayText = wrapHyperlink(displayText, url)
	}

	return displayText
}

// FormatError formats an error message with red color if color is enabled.
func FormatError(msg string, useColor bool) string {
	if useColor {
		return colorRed + "Error: " + msg + colorReset
	}
	return "Error: " + msg
}
