// Package github provides a client for interacting with the GitHub API
// to fetch pull request and commit information from NixOS/nixpkgs.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// RelatedPR represents a pull request that is cross-referenced from an issue.
type RelatedPR struct {
	Number int
	Title  string
	URL    string
	State  string // "open", "closed", or "merged" (derived from merged_at presence)
}

// timelineEvent represents a single event from the issue timeline API.
// We only parse fields needed for cross-reference detection.
type timelineEvent struct {
	Event  string                `json:"event"`
	Source *crossReferenceSource `json:"source,omitempty"`
}

// crossReferenceSource contains the source of a cross-reference event.
type crossReferenceSource struct {
	Issue *crossReferenceIssue `json:"issue,omitempty"`
}

// crossReferenceIssue represents the issue/PR that created the cross-reference.
type crossReferenceIssue struct {
	Number      int                       `json:"number"`
	Title       string                    `json:"title"`
	State       string                    `json:"state"`
	HTMLURL     string                    `json:"html_url"`
	PullRequest *crossReferencePR         `json:"pull_request,omitempty"`
	Repository  *crossReferenceRepository `json:"repository,omitempty"`
}

// crossReferencePR contains PR-specific fields from the cross-reference.
// GitHub's timeline API returns state="closed" for merged PRs, so we check
// merged_at to determine if a PR was actually merged vs just closed.
type crossReferencePR struct {
	MergedAt *string `json:"merged_at,omitempty"`
}

// crossReferenceRepository identifies which repo the cross-reference came from.
type crossReferenceRepository struct {
	FullName string `json:"full_name"`
}

// DefaultTimelinePages is the default number of timeline pages to fetch.
const DefaultTimelinePages = 3

// GetRelatedPRs fetches cross-referenced PRs from an issue's timeline.
// Only returns PRs from the NixOS/nixpkgs repository.
// maxPages controls how many pages of timeline events to fetch (100 events per page).
// Returns nil (not error) if the timeline cannot be fetched.
func (c *Client) GetRelatedPRs(ctx context.Context, issueNumber int, maxPages int) []RelatedPR {
	if maxPages <= 0 {
		maxPages = DefaultTimelinePages
	}

	var related []RelatedPR
	seen := make(map[int]bool)
	pagesFetched := 0
	totalEvents := 0

	c.log.Debug("fetching issue timeline",
		zap.Int("issue", issueNumber),
		zap.Int("max_pages", maxPages))

	for page := 1; page <= maxPages; page++ {
		path := fmt.Sprintf("/repos/NixOS/nixpkgs/issues/%d/timeline?per_page=100&page=%d", issueNumber, page)

		body, err := c.doRequestWithAccept(ctx, http.MethodGet, path, "application/vnd.github+json")
		if err != nil {
			c.log.Debug("failed to fetch issue timeline",
				zap.Int("issue", issueNumber),
				zap.Int("page", page),
				zap.Error(err))
			break
		}

		var events []timelineEvent
		if err := json.Unmarshal(body, &events); err != nil {
			c.log.Debug("failed to parse timeline response",
				zap.Int("issue", issueNumber),
				zap.Int("page", page),
				zap.Error(err))
			break
		}

		if len(events) == 0 {
			c.log.Debug("timeline page empty, stopping pagination",
				zap.Int("issue", issueNumber),
				zap.Int("page", page))
			break
		}

		pagesFetched++
		totalEvents += len(events)

		for _, event := range events {
			if event.Event != "cross-referenced" {
				continue
			}
			if event.Source == nil || event.Source.Issue == nil {
				continue
			}

			issue := event.Source.Issue
			if issue.PullRequest == nil {
				continue
			}

			if issue.Repository != nil && issue.Repository.FullName != "NixOS/nixpkgs" {
				continue
			}

			if seen[issue.Number] {
				continue
			}
			seen[issue.Number] = true

			state := issue.State
			if issue.PullRequest.MergedAt != nil && *issue.PullRequest.MergedAt != "" {
				state = StateMerged
			}

			related = append(related, RelatedPR{
				Number: issue.Number,
				Title:  issue.Title,
				URL:    issue.HTMLURL,
				State:  state,
			})
		}
	}

	c.log.Debug("timeline fetch complete",
		zap.Int("issue", issueNumber),
		zap.Int("pages_fetched", pagesFetched),
		zap.Int("total_events", totalEvents),
		zap.Int("related_prs_found", len(related)))

	return related
}

// doRequestWithAccept performs an HTTP request with a custom Accept header.
func (c *Client) doRequestWithAccept(ctx context.Context, method, path, accept string) ([]byte, error) {
	url := c.BaseURL + path

	c.log.Debug("request", zap.String("method", method), zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "nprt/1.0")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error talking to GitHub: %w", err)
	}
	defer resp.Body.Close()

	c.log.Debug("response", zap.Int("status_code", resp.StatusCode), zap.String("status", resp.Status))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    "GitHub API rate limit exceeded. Try again later or set GITHUB_TOKEN.",
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "GitHub authentication failed. Check your GITHUB_TOKEN.",
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    extractAPIMessage(body),
		}
	}

	return body, nil
}
