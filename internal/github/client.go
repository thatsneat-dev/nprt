// Package github provides a client for interacting with the GitHub API
// to fetch pull request and commit information from NixOS/nixpkgs.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultBaseURL = "https://api.github.com"
	DefaultTimeout = 30 * time.Second
)

const (
	StateOpen   = "open"
	StateClosed = "closed"
	StateMerged = "merged"
	StateDraft  = "draft"
)

// Client is a GitHub API client configured for NixOS/nixpkgs.
type Client struct {
	BaseURL       string
	Token         string
	UserAgent     string
	HTTPClient    *http.Client
	TimelinePages int
	log           *zap.Logger
}

// PullRequest represents a GitHub pull request with relevant fields.
type PullRequest struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	State          string `json:"state"`
	Draft          bool   `json:"draft"`
	Merged         bool   `json:"merged"`
	MergeCommitSHA string `json:"merge_commit_sha"`
	User           struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// CompareResult represents the result of comparing two commits or branches.
type CompareResult struct {
	Status   string `json:"status"`
	AheadBy  int    `json:"ahead_by"`
	BehindBy int    `json:"behind_by"`
}

// APIError represents an error response from the GitHub API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("GitHub API error (status %d): %s", e.StatusCode, e.Message)
}

// NotFoundError indicates that no issue or pull request exists with the given number.
// Callers can detect this with: var nf *NotFoundError; errors.As(err, &nf)
type NotFoundError struct {
	Number int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no PR or issue #%d exists in NixOS/nixpkgs", e.Number)
}

// NotPullRequestError indicates that the number exists but is an Issue, not a PR.
// Callers can detect this with: var npr *NotPullRequestError; errors.As(err, &npr)
type NotPullRequestError struct {
	Number     int
	Title      string
	State      string
	URL        string
	RelatedPRs []RelatedPR
}

func (e *NotPullRequestError) Error() string {
	var base string
	if e.Title != "" {
		base = fmt.Sprintf("#%d is an issue, not a pull request: %q (%s)", e.Number, e.Title, e.URL)
	} else {
		base = fmt.Sprintf("#%d is an issue, not a pull request", e.Number)
	}

	if len(e.RelatedPRs) == 0 {
		return base
	}

	base += "\n\nRelated pull requests:"
	for _, pr := range e.RelatedPRs {
		base += fmt.Sprintf("\n  • #%d (%s): %s", pr.Number, pr.State, pr.Title)
	}
	return base
}

// Issue represents a GitHub issue with minimal fields for type detection.
type Issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	HTMLURL     string    `json:"html_url"`
	PullRequest *struct{} `json:"pull_request"`
}

// NewClient creates a new GitHub API client with the given token, user agent, and logger.
func NewClient(token string, userAgent string, log *zap.Logger) *Client {
	return &Client{
		BaseURL:       DefaultBaseURL,
		Token:         token,
		UserAgent:     userAgent,
		TimelinePages: DefaultTimelinePages,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		log: log.Named("github"),
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	return c.doRequestWithAccept(ctx, method, path, "application/vnd.github.v3+json")
}

// extractAPIMessage attempts to parse a GitHub API error response and extract
// the "message" field. Falls back to the raw body if parsing fails.
func extractAPIMessage(body []byte) string {
	var parsed struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Message != "" {
		return parsed.Message
	}
	return string(body)
}

// GetPullRequest fetches a pull request by number from NixOS/nixpkgs.
// If the number is an Issue (not a PR), returns NotPullRequestError.
// If the number doesn't exist at all, returns NotFoundError.
func (c *Client) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/NixOS/nixpkgs/pulls/%d", number)

	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, c.disambiguateNotFound(ctx, number)
		}
		return nil, err
	}

	var pr PullRequest
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	return &pr, nil
}

// GetIssue fetches an issue by number from NixOS/nixpkgs.
func (c *Client) GetIssue(ctx context.Context, number int) (*Issue, error) {
	path := fmt.Sprintf("/repos/NixOS/nixpkgs/issues/%d", number)

	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue response: %w", err)
	}

	return &issue, nil
}

// disambiguateNotFound checks if a 404 from /pulls is due to the number being
// an Issue (not a PR) or not existing at all.
func (c *Client) disambiguateNotFound(ctx context.Context, number int) error {
	issue, err := c.GetIssue(ctx, number)
	if err == nil {
		if issue.PullRequest == nil {
			relatedPRs := c.GetRelatedPRs(ctx, number, c.TimelinePages)
			return &NotPullRequestError{
				Number:     number,
				Title:      issue.Title,
				State:      issue.State,
				URL:        issue.HTMLURL,
				RelatedPRs: relatedPRs,
			}
		}
		// The issue endpoint says it's a PR, but /pulls returned 404.
		// This is unexpected; possibly an auth/scope issue or API anomaly.
		return fmt.Errorf("PR #%d exists but could not be fetched; check your GITHUB_TOKEN permissions", number)
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return &NotFoundError{Number: number}
	}

	return err
}

// CompareCommitWithBranch checks if a commit is present in a branch.
func (c *Client) CompareCommitWithBranch(ctx context.Context, commit, branch string) (*CompareResult, error) {
	path := fmt.Sprintf("/repos/NixOS/nixpkgs/compare/%s...%s", url.PathEscape(commit), url.PathEscape(branch))

	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result CompareResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse compare response: %w", err)
	}

	return &result, nil
}
