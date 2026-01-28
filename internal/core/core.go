// Package core provides the business logic for checking PR propagation
// across nixpkgs release channels.
package core

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"go.uber.org/zap"

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
	client *github.Client
	log    *zap.Logger
}

// NewChecker creates a new Checker with the given GitHub client and logger.
func NewChecker(client *github.Client, log *zap.Logger) *Checker {
	return &Checker{client: client, log: log.Named("core")}
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
		c.log.Debug("PR not merged, skipping channel checks", zap.Int("pr", prNumber))
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

	c.log.Debug("checking channels",
		zap.Int("count", len(channels)),
		zap.String("commit", pr.MergeCommitSHA[:12]),
	)

	// Check all channels in parallel for faster results
	results := make([]ChannelResult, len(channels))
	var wg sync.WaitGroup
	wg.Add(len(channels))

	for i, ch := range channels {
		i, ch := i, ch // capture loop variables
		go func() {
			defer wg.Done()
			c.log.Debug("checking channel", zap.String("channel", ch.Name), zap.String("branch", ch.Branch))
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
		c.log.Debug("channel check failed", zap.String("channel", ch.Name), zap.Error(err))
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

// SortChannelResults sorts channels so present channels come first (preserving
// their original order), followed by non-present channels sorted alphabetically.
// Sorts in place and returns the same slice.
func SortChannelResults(results []ChannelResult) []ChannelResult {
	sort.SliceStable(results, func(i, j int) bool {
		isPresentI := results[i].Status == StatusPresent
		isPresentJ := results[j].Status == StatusPresent

		if isPresentI != isPresentJ {
			return isPresentI
		}
		if !isPresentI {
			return results[i].Name < results[j].Name
		}
		return false
	})
	return results
}
