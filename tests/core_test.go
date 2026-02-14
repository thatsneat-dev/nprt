package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/thatsneat-dev/nprt/internal/config"
	"github.com/thatsneat-dev/nprt/internal/core"
	"github.com/thatsneat-dev/nprt/internal/github"
)

func TestSortChannelResults_PresentFirst(t *testing.T) {
	results := []core.ChannelResult{
		{Name: "alpha", Status: core.StatusNotPresent},
		{Name: "beta", Status: core.StatusPresent},
		{Name: "gamma", Status: core.StatusNotPresent},
		{Name: "delta", Status: core.StatusPresent},
	}

	sorted := core.SortChannelResults(results)

	if sorted[0].Status != core.StatusPresent || sorted[1].Status != core.StatusPresent {
		t.Error("Present channels should come first")
	}

	if sorted[0].Name != "beta" || sorted[1].Name != "delta" {
		t.Errorf("Present channels should maintain relative order: got %s, %s", sorted[0].Name, sorted[1].Name)
	}
}

func TestSortChannelResults_NotPresentAlphabetical(t *testing.T) {
	results := []core.ChannelResult{
		{Name: "zebra", Status: core.StatusNotPresent},
		{Name: "alpha", Status: core.StatusNotPresent},
		{Name: "master", Status: core.StatusPresent},
		{Name: "beta", Status: core.StatusNotPresent},
	}

	sorted := core.SortChannelResults(results)

	if sorted[0].Name != "master" {
		t.Errorf("First should be master (present), got %s", sorted[0].Name)
	}
	if sorted[1].Name != "alpha" {
		t.Errorf("Second should be alpha, got %s", sorted[1].Name)
	}
	if sorted[2].Name != "beta" {
		t.Errorf("Third should be beta, got %s", sorted[2].Name)
	}
	if sorted[3].Name != "zebra" {
		t.Errorf("Fourth should be zebra, got %s", sorted[3].Name)
	}
}

func TestSortChannelResults_UnknownWithNotPresent(t *testing.T) {
	results := []core.ChannelResult{
		{Name: "unknown-channel", Status: core.StatusUnknown},
		{Name: "master", Status: core.StatusPresent},
		{Name: "alpha", Status: core.StatusNotPresent},
	}

	sorted := core.SortChannelResults(results)

	if sorted[0].Name != "master" {
		t.Errorf("First should be master (present), got %s", sorted[0].Name)
	}
	if sorted[1].Name != "alpha" {
		t.Errorf("Second should be alpha (alphabetically first among non-present), got %s", sorted[1].Name)
	}
	if sorted[2].Name != "unknown-channel" {
		t.Errorf("Third should be unknown-channel, got %s", sorted[2].Name)
	}
}

func TestSortChannelResults_Empty(t *testing.T) {
	results := []core.ChannelResult{}
	sorted := core.SortChannelResults(results)
	if len(sorted) != 0 {
		t.Error("Empty input should return empty output")
	}
}

func TestSortChannelResults_AllPresent(t *testing.T) {
	results := []core.ChannelResult{
		{Name: "gamma", Status: core.StatusPresent},
		{Name: "alpha", Status: core.StatusPresent},
		{Name: "beta", Status: core.StatusPresent},
	}

	sorted := core.SortChannelResults(results)

	if sorted[0].Name != "gamma" || sorted[1].Name != "alpha" || sorted[2].Name != "beta" {
		t.Error("All present channels should maintain original order")
	}
}

func TestCheckPR_MergedPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pulls/100"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 100,
				"title": "Test PR",
				"state": "closed",
				"merged": true,
				"merge_commit_sha": "abc123def456789012",
				"user": {"login": "testuser"},
				"base": {"ref": "master"}
			}`))
		case strings.Contains(r.URL.Path, "/compare/"):
			if strings.Contains(r.URL.Path, "master") {
				w.Write([]byte(`{"status": "ahead", "ahead_by": 10, "behind_by": 0}`))
			} else {
				w.Write([]byte(`{"status": "behind", "ahead_by": 0, "behind_by": 5}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := github.NewClient("", "", zap.NewNop())
	client.BaseURL = server.URL
	checker := core.NewChecker(client, zap.NewNop())

	channels := []config.Channel{
		{Name: "master", Branch: "master"},
		{Name: "nixos-unstable", Branch: "nixos-unstable"},
	}

	status, err := checker.CheckPR(context.Background(), 100, channels)
	if err != nil {
		t.Fatalf("CheckPR returned error: %v", err)
	}

	if status.Number != 100 {
		t.Errorf("Number = %d, want 100", status.Number)
	}
	if status.State != core.PRStateMerged {
		t.Errorf("State = %q, want %q", status.State, core.PRStateMerged)
	}
	if status.Author != "testuser" {
		t.Errorf("Author = %q, want %q", status.Author, "testuser")
	}

	// After sorting, present channels come first
	if len(status.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(status.Channels))
	}
	if status.Channels[0].Status != core.StatusPresent {
		t.Errorf("first channel status = %q, want present", status.Channels[0].Status)
	}
}

func TestCheckPR_UnmergedPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"number": 200,
			"title": "Open PR",
			"state": "open",
			"merged": false,
			"draft": false,
			"user": {"login": "dev"},
			"base": {"ref": "master"}
		}`))
	}))
	defer server.Close()

	client := github.NewClient("", "", zap.NewNop())
	client.BaseURL = server.URL
	checker := core.NewChecker(client, zap.NewNop())

	channels := []config.Channel{
		{Name: "master", Branch: "master"},
	}

	status, err := checker.CheckPR(context.Background(), 200, channels)
	if err != nil {
		t.Fatalf("CheckPR returned error: %v", err)
	}

	if status.State != core.PRStateOpen {
		t.Errorf("State = %q, want %q", status.State, core.PRStateOpen)
	}
	for _, ch := range status.Channels {
		if ch.Status != core.StatusNotPresent {
			t.Errorf("channel %s status = %q, want not_present for unmerged PR", ch.Name, ch.Status)
		}
	}
}

func TestCheckPR_ChannelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pulls/300"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 300,
				"title": "Test",
				"state": "closed",
				"merged": true,
				"merge_commit_sha": "deadbeef12345678",
				"user": {"login": "dev"},
				"base": {"ref": "master"}
			}`))
		case strings.Contains(r.URL.Path, "/compare/") && strings.Contains(r.URL.Path, "master"):
			w.Write([]byte(`{"status": "ahead", "ahead_by": 10, "behind_by": 0}`))
		case strings.Contains(r.URL.Path, "/compare/"):
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := github.NewClient("", "", zap.NewNop())
	client.BaseURL = server.URL
	checker := core.NewChecker(client, zap.NewNop())

	channels := []config.Channel{
		{Name: "master", Branch: "master"},
		{Name: "bad-branch", Branch: "bad-branch"},
	}

	status, err := checker.CheckPR(context.Background(), 300, channels)
	if err != nil {
		t.Fatalf("CheckPR returned error: %v", err)
	}

	var found bool
	for _, ch := range status.Channels {
		if ch.Name == "bad-branch" {
			found = true
			if ch.Status != core.StatusUnknown {
				t.Errorf("bad-branch status = %q, want unknown", ch.Status)
			}
			if ch.Error == "" {
				t.Error("bad-branch should have error message")
			}
		}
	}
	if !found {
		t.Error("bad-branch not found in results")
	}
}
