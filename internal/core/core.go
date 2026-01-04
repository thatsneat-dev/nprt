// Package core provides the business logic for checking PR propagation
// across nixpkgs release channels.
package core

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/config"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/github"
)

// ChannelStatus indicates whether a PR's merge commit is present in a channel.
type ChannelStatus string

const (
	StatusPresent    ChannelStatus = "present"
	StatusNotPresent ChannelStatus = "not_present"
	StatusUnknown    ChannelStatus = "unknown"
)

// PRState represents the current state of a pull request.
type PRState string

const (
	PRStateDraft  PRState = "draft"
	PRStateOpen   PRState = "open"
	PRStateMerged PRState = "merged"
	PRStateClosed PRState = "closed"
)

// ChannelResult holds the propagation status for a single channel.
type ChannelResult struct {
	Name   string        `json:"name"`
	Branch string        `json:"branch"`
	Status ChannelStatus `json:"status"`
	Error  string        `json:"error,omitempty"`
}

// PRStatus contains the full status of a PR including all channel results.
type PRStatus struct {
	Number      int             `json:"pr"`
	Title       string          `json:"title,omitempty"`
	Author      string          `json:"author,omitempty"`
	State       PRState         `json:"state"`
	MergeCommit string          `json:"merge_commit,omitempty"`
	Channels    []ChannelResult `json:"channels"`
}

// Checker queries GitHub to determine PR status and channel propagation.
type Checker struct {
	client  *github.Client
	verbose bool
}

// NewChecker creates a new Checker with the given GitHub client.
func NewChecker(client *github.Client, verbose bool) *Checker {
	return &Checker{client: client, verbose: verbose}
}

// CheckPR fetches a PR and checks its propagation across the given channels.
func (c *Checker) CheckPR(ctx context.Context, prNumber int, channels []config.Channel) (*PRStatus, error) {
	pr, err := c.client.GetPullRequest(ctx, prNumber)
	if err != nil {
		return nil, err
	}

	status := &PRStatus{
		Number:      pr.Number,
		Title:       pr.Title,
		Author:      pr.User.Login,
		State:       determinePRState(pr),
		MergeCommit: pr.MergeCommitSHA,
	}

	if !pr.Merged {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "PR is not merged, skipping channel checks\n")
		}
		results := make([]ChannelResult, len(channels))
		for i, ch := range channels {
			results[i] = ChannelResult{
				Name:   ch.Name,
				Branch: ch.Branch,
				Status: StatusNotPresent,
			}
		}
		status.Channels = SortChannelResults(results)
		return status, nil
	}

	if pr.MergeCommitSHA == "" {
		return nil, fmt.Errorf("PR #%d has no merge commit SHA", prNumber)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "Checking %d channels for commit %s...\n", len(channels), pr.MergeCommitSHA[:12])
	}

	// Check all channels in parallel for faster results
	results := make([]ChannelResult, len(channels))
	var wg sync.WaitGroup
	wg.Add(len(channels))

	for i, ch := range channels {
		i, ch := i, ch // capture loop variables
		go func() {
			defer wg.Done()
			if c.verbose {
				fmt.Fprintf(os.Stderr, "Checking channel %s...\n", ch.Name)
			}
			results[i] = c.checkChannel(ctx, pr.MergeCommitSHA, ch)
		}()
	}

	wg.Wait()

	status.Channels = SortChannelResults(results)
	return status, nil
}

func determinePRState(pr *github.PullRequest) PRState {
	if pr.Merged {
		return PRStateMerged
	}
	if pr.Draft {
		return PRStateDraft
	}
	if pr.State == "open" {
		return PRStateOpen
	}
	return PRStateClosed
}

// checkChannel determines if a commit is present in the given channel branch.
func (c *Checker) checkChannel(ctx context.Context, commit string, ch config.Channel) ChannelResult {
	result := ChannelResult{
		Name:   ch.Name,
		Branch: ch.Branch,
		Status: StatusUnknown,
	}

	compare, err := c.client.CompareCommitWithBranch(ctx, commit, ch.Branch)
	if err != nil {
		result.Error = err.Error()
		if c.verbose {
			fmt.Fprintf(os.Stderr, "  Error checking %s: %v\n", ch.Name, err)
		}
		return result
	}

	// GitHub compare: BASE=commit, HEAD=branch
	// If BehindBy == 0, branch contains all commits from BASE (the PR's merge commit)
	if compare.BehindBy == 0 {
		result.Status = StatusPresent
	} else {
		result.Status = StatusNotPresent
	}

	return result
}

// SortChannelResults sorts channels with present first, then others alphabetically.
// Sorts in place and returns the same slice.
func SortChannelResults(results []ChannelResult) []ChannelResult {
	sort.SliceStable(results, func(i, j int) bool {
		// Present channels come first
		if results[i].Status == StatusPresent && results[j].Status != StatusPresent {
			return true
		}
		if results[i].Status != StatusPresent && results[j].Status == StatusPresent {
			return false
		}
		// Non-present channels sorted alphabetically
		if results[i].Status != StatusPresent && results[j].Status != StatusPresent {
			return results[i].Name < results[j].Name
		}
		return false
	})
	return results
}
