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
				"state": "open",
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

func TestGetPullRequest_IsIssueWithRelatedPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pulls/") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
			return
		}
		if strings.Contains(r.URL.Path, "/timeline") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"event": "cross-referenced",
					"source": {
						"issue": {
							"number": 123456,
							"title": "Fix the bug",
							"state": "open",
							"html_url": "https://github.com/NixOS/nixpkgs/pull/123456",
							"pull_request": {},
							"repository": {"full_name": "NixOS/nixpkgs"}
						}
					}
				},
				{
					"event": "cross-referenced",
					"source": {
						"issue": {
							"number": 789012,
							"title": "Another fix",
							"state": "closed",
							"html_url": "https://github.com/NixOS/nixpkgs/pull/789012",
							"pull_request": {"merged_at": "2024-01-01T00:00:00Z"},
							"repository": {"full_name": "NixOS/nixpkgs"}
						}
					}
				},
				{
					"event": "cross-referenced",
					"source": {
						"issue": {
							"number": 999,
							"title": "External PR",
							"state": "open",
							"html_url": "https://github.com/other/repo/pull/999",
							"pull_request": {},
							"repository": {"full_name": "other/repo"}
						}
					}
				},
				{
					"event": "commented",
					"body": "Just a comment"
				}
			]`))
			return
		}
		if strings.Contains(r.URL.Path, "/issues/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 12345,
				"title": "Some bug report",
				"state": "open",
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

	if len(notPRErr.RelatedPRs) != 2 {
		t.Errorf("expected 2 related PRs, got %d", len(notPRErr.RelatedPRs))
	}

	if notPRErr.RelatedPRs[0].Number != 123456 {
		t.Errorf("first related PR number = %d, want 123456", notPRErr.RelatedPRs[0].Number)
	}
	if notPRErr.RelatedPRs[1].Number != 789012 {
		t.Errorf("second related PR number = %d, want 789012", notPRErr.RelatedPRs[1].Number)
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Related pull requests:") {
		t.Errorf("error message should contain 'Related pull requests:': %v", errMsg)
	}
	if !strings.Contains(errMsg, "#123456") {
		t.Errorf("error message should contain PR #123456: %v", errMsg)
	}
}

func TestGetPullRequest_IsIssueTimelineFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pulls/") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
			return
		}
		if strings.Contains(r.URL.Path, "/timeline") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "Internal Server Error"}`))
			return
		}
		if strings.Contains(r.URL.Path, "/issues/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 12345,
				"title": "Some bug report",
				"state": "open",
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

	if len(notPRErr.RelatedPRs) != 0 {
		t.Errorf("expected 0 related PRs when timeline fails, got %d", len(notPRErr.RelatedPRs))
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "Related pull requests:") {
		t.Errorf("error message should not contain 'Related pull requests:' when none found: %v", errMsg)
	}
}

func TestGetRelatedPRs_DeduplicatesPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"event": "cross-referenced",
				"source": {
					"issue": {
						"number": 123456,
						"title": "Fix the bug",
						"state": "open",
						"html_url": "https://github.com/NixOS/nixpkgs/pull/123456",
						"pull_request": {},
						"repository": {"full_name": "NixOS/nixpkgs"}
					}
				}
			},
			{
				"event": "cross-referenced",
				"source": {
					"issue": {
						"number": 123456,
						"title": "Fix the bug",
						"state": "open",
						"html_url": "https://github.com/NixOS/nixpkgs/pull/123456",
						"pull_request": {},
						"repository": {"full_name": "NixOS/nixpkgs"}
					}
				}
			}
		]`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	related := client.GetRelatedPRs(context.Background(), 12345, 3)
	if len(related) != 1 {
		t.Errorf("expected 1 deduplicated PR, got %d", len(related))
	}
}

func TestGetRelatedPRs_MergedAtSetsMergedState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"event": "cross-referenced",
				"source": {
					"issue": {
						"number": 111,
						"title": "Merged PR",
						"state": "closed",
						"html_url": "https://github.com/NixOS/nixpkgs/pull/111",
						"pull_request": {"merged_at": "2024-01-15T12:00:00Z"},
						"repository": {"full_name": "NixOS/nixpkgs"}
					}
				}
			},
			{
				"event": "cross-referenced",
				"source": {
					"issue": {
						"number": 222,
						"title": "Closed PR",
						"state": "closed",
						"html_url": "https://github.com/NixOS/nixpkgs/pull/222",
						"pull_request": {},
						"repository": {"full_name": "NixOS/nixpkgs"}
					}
				}
			}
		]`))
	}))
	defer server.Close()

	client := github.NewClient("", zap.NewNop())
	client.BaseURL = server.URL

	related := client.GetRelatedPRs(context.Background(), 12345, 3)
	if len(related) != 2 {
		t.Fatalf("expected 2 related PRs, got %d", len(related))
	}

	if related[0].Number != 111 {
		t.Errorf("first PR number = %d, want 111", related[0].Number)
	}
	if related[0].State != "merged" {
		t.Errorf("first PR state = %q, want %q (closed with merged_at should be merged)", related[0].State, "merged")
	}

	if related[1].Number != 222 {
		t.Errorf("second PR number = %d, want 222", related[1].Number)
	}
	if related[1].State != "closed" {
		t.Errorf("second PR state = %q, want %q (closed without merged_at should be closed)", related[1].State, "closed")
	}
}
