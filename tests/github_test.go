package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/github"
)

func TestGetPullRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/pulls/476497") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"number": 476497,
			"state": "closed",
			"merged": true,
			"merge_commit_sha": "abc123def456",
			"base": {"ref": "master"}
		}`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	pr, err := client.GetPullRequest(context.Background(), 476497)
	if err != nil {
		t.Fatalf("GetPullRequest returned error: %v", err)
	}

	if pr.Number != 476497 {
		t.Errorf("PR number = %d, want 476497", pr.Number)
	}
	if !pr.Merged {
		t.Error("PR should be merged")
	}
	if pr.MergeCommitSHA != "abc123def456" {
		t.Errorf("MergeCommitSHA = %q, want %q", pr.MergeCommitSHA, "abc123def456")
	}
}

func TestGetPullRequest_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Both /pulls and /issues return 404 - number doesn't exist
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	_, err := client.GetPullRequest(context.Background(), 999999)
	if err == nil {
		t.Fatal("GetPullRequest should have returned error")
	}

	var notFoundErr *github.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("error should be NotFoundError, got: %T (%v)", err, err)
	}
	if notFoundErr.Number != 999999 {
		t.Errorf("NotFoundError.Number = %d, want 999999", notFoundErr.Number)
	}
}

func TestGetPullRequest_IsIssueNotPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pulls/") {
			// /pulls returns 404 because it's not a PR
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
			return
		}
		if strings.Contains(r.URL.Path, "/issues/") {
			// /issues returns 200 with an issue (no pull_request field)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 12345,
				"title": "Some bug report",
				"html_url": "https://github.com/NixOS/nixpkgs/issues/12345"
			}`))
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	_, err := client.GetPullRequest(context.Background(), 12345)
	if err == nil {
		t.Fatal("GetPullRequest should have returned error")
	}

	var notPRErr *github.NotPullRequestError
	if !errors.As(err, &notPRErr) {
		t.Errorf("error should be NotPullRequestError, got: %T (%v)", err, err)
	}
	if notPRErr.Number != 12345 {
		t.Errorf("NotPullRequestError.Number = %d, want 12345", notPRErr.Number)
	}
	if notPRErr.Title != "Some bug report" {
		t.Errorf("NotPullRequestError.Title = %q, want %q", notPRErr.Title, "Some bug report")
	}
	if !strings.Contains(err.Error(), "issue, not a pull request") {
		t.Errorf("error message should mention 'issue, not a pull request': %v", err)
	}
}

func TestGetPullRequest_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "rate limit exceeded"}`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	_, err := client.GetPullRequest(context.Background(), 123)
	if err == nil {
		t.Fatal("GetPullRequest should have returned error")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error should mention 'rate limit': %v", err)
	}
}

func TestCompareCommitWithBranch_CommitInBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ahead", "ahead_by": 100, "behind_by": 0}`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	result, err := client.CompareCommitWithBranch(context.Background(), "abc123", "master")
	if err != nil {
		t.Fatalf("CompareCommitWithBranch returned error: %v", err)
	}
	if result.BehindBy != 0 {
		t.Errorf("BehindBy = %d, want 0 (commit is in branch)", result.BehindBy)
	}
}

func TestCompareCommitWithBranch_CommitNotInBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "behind", "ahead_by": 0, "behind_by": 5}`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	result, err := client.CompareCommitWithBranch(context.Background(), "abc123", "nixos-unstable")
	if err != nil {
		t.Fatalf("CompareCommitWithBranch returned error: %v", err)
	}
	if result.BehindBy == 0 {
		t.Error("BehindBy should be > 0 (commit not in branch)")
	}
}

func TestGetPullRequest_IssueEndpointSaysPR(t *testing.T) {
	// Edge case: /pulls returns 404, but /issues returns 200 with pull_request field
	// This indicates an unexpected API state (auth issue, API anomaly, etc.)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pulls/") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
			return
		}
		if strings.Contains(r.URL.Path, "/issues/") {
			// Issue endpoint says it's a PR (has pull_request field)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 99999,
				"title": "Some PR",
				"html_url": "https://github.com/NixOS/nixpkgs/pull/99999",
				"pull_request": {}
			}`))
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	_, err := client.GetPullRequest(context.Background(), 99999)
	if err == nil {
		t.Fatal("GetPullRequest should have returned error")
	}
	if !strings.Contains(err.Error(), "could not be fetched") {
		t.Errorf("error should mention 'could not be fetched': %v", err)
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error should mention GITHUB_TOKEN: %v", err)
	}
}

func TestAPIError_ParsesJSONMessage(t *testing.T) {
	// Test that API errors extract the "message" field from JSON responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Validation Failed", "documentation_url": "https://docs.github.com"}`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	_, err := client.GetPullRequest(context.Background(), 123)
	if err == nil {
		t.Fatal("GetPullRequest should have returned error")
	}

	var apiErr *github.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got: %T", err)
	}
	if apiErr.Message != "Validation Failed" {
		t.Errorf("APIError.Message = %q, want %q", apiErr.Message, "Validation Failed")
	}
}
