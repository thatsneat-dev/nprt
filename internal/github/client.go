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
	"os"
	"time"
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
	Verbose    bool
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

// NewClient creates a new GitHub API client with the given token.
func NewClient(token string, verbose bool) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		Verbose: verbose,
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	url := c.BaseURL + path

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "  → %s %s\n", method, url)
	}

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

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "  ← %d %s\n", resp.StatusCode, resp.Status)
	}

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
			Message:    string(body),
		}
	}

	return body, nil
}

// GetPullRequest fetches a pull request by number from NixOS/nixpkgs.
func (c *Client) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/NixOS/nixpkgs/pulls/%d", number)

	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("PR #%d not found in NixOS/nixpkgs", number)
		}
		return nil, err
	}

	var pr PullRequest
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	return &pr, nil
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
