// Package github provides a client for interacting with the GitHub API
// to fetch pull request and commit information from NixOS/nixpkgs.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultBaseURL = "https://api.github.com"
	DefaultTimeout = 30 * time.Second
)

// Client is a GitHub API client configured for NixOS/nixpkgs.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	log        *zap.Logger
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
	Number int
	Title  string
	URL    string
}

func (e *NotPullRequestError) Error() string {
	if e.Title != "" {
		return fmt.Sprintf("#%d is an issue, not a pull request: %q (%s)", e.Number, e.Title, e.URL)
	}
	return fmt.Sprintf("#%d is an issue, not a pull request", e.Number)
}

// Issue represents a GitHub issue with minimal fields for type detection.
type Issue struct {
	Number      int     `json:"number"`
	Title       string  `json:"title"`
	HTMLURL     string  `json:"html_url"`
	PullRequest *struct{} `json:"pull_request"`
}

// NewClient creates a new GitHub API client with the given token and logger.
func NewClient(token string, log *zap.Logger) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		log: log.Named("github"),
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	url := c.BaseURL + path

	c.log.Debug("request", zap.String("method", method), zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
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
			return &NotPullRequestError{
				Number: number,
				Title:  issue.Title,
				URL:    issue.HTMLURL,
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
	path := fmt.Sprintf("/repos/NixOS/nixpkgs/compare/%s...%s", commit, branch)

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
