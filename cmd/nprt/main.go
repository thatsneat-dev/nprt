package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/thatsneat-dev/nprt/internal/cli"
	"github.com/thatsneat-dev/nprt/internal/config"
	"github.com/thatsneat-dev/nprt/internal/core"
	"github.com/thatsneat-dev/nprt/internal/github"
	"github.com/thatsneat-dev/nprt/internal/logging"
	"github.com/thatsneat-dev/nprt/internal/render"
)

var version = "dev"

const usage = `Usage: nprt [options] <PR number | PR URL>

Track which nixpkgs channels contain a given pull request.

Arguments:
  PR number    A pull request number (e.g., 476497)
  PR URL       A full GitHub PR URL (e.g., https://github.com/NixOS/nixpkgs/pull/476497)

Options:
  --channels         Comma-separated list of channels to check (default: master,staging-next,nixpkgs-unstable,nixos-unstable-small,nixos-unstable)
  --color            Color output mode: auto, always, never (default: auto)
  --json             Output results as JSON
  --timeline-pages   Number of timeline pages to fetch for related PRs (default: 3)
  --verbose          Show detailed progress and debug information
  --version          Print version and exit
  -h, --help         Show this help message

Environment:
  GITHUB_TOKEN  GitHub personal access token for higher rate limits
`

func main() {
	os.Exit(run())
}

func run() int {
	var (
		channelsFlag  string
		colorMode     string
		jsonOutput    bool
		timelinePages int
		verbose       bool
		showVersion   bool
	)

	flag.StringVar(&channelsFlag, "channels", "", "Comma-separated list of channels to check")
	flag.StringVar(&colorMode, "color", "auto", "Color output: auto, always, never")
	flag.BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	flag.IntVar(&timelinePages, "timeline-pages", github.DefaultTimelinePages, "Number of timeline pages to fetch for related PRs")
	flag.BoolVar(&verbose, "verbose", false, "Show detailed progress and debug information")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	if err := flag.CommandLine.Parse(cli.ReorderArgs(flag.CommandLine, os.Args[1:])); err != nil {
		return 2
	}

	if showVersion {
		fmt.Printf("nprt version %s\n", version)
		return 0
	}

	if timelinePages < 1 || timelinePages > 10 {
		fmt.Fprintln(os.Stderr, "Error: --timeline-pages must be between 1 and 10")
		return 2
	}

	// Compute color settings early so all errors can be styled
	useColor, err := config.ShouldUseColor(colorMode)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	stderrColor := config.ShouldUseColorForFile(colorMode, os.Stderr)
	useHyperlinks := config.IsTerminal()

	args := flag.Args()

	if unknown := cli.HasUnknownFlags(args); unknown != "" {
		fmt.Fprintln(os.Stderr, render.FormatError("unknown flag "+unknown, stderrColor))
		return 2
	}

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	prNumber, err := config.ParsePRInput(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, render.FormatError(err.Error(), stderrColor))
		return 2
	}

	channels, err := config.ParseChannels(channelsFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, render.FormatError(err.Error(), stderrColor))
		return 2
	}

	token := config.GetGitHubToken()

	// Create logger based on verbose flag
	log := logging.New(verbose)
	defer func() { _ = log.Sync() }()

	log.Debug("fetching PR", zap.Int("pr", prNumber))

	client := github.NewClient(token, "nprt/"+version, log)
	client.TimelinePages = timelinePages
	checker := core.NewChecker(client, log)

	// Set up context with signal handling for clean cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	status, err := checker.CheckPR(ctx, prNumber, channels)
	if err != nil {
		// 403 errors (rate limit, auth failure) get a distinct exit code
		var apiErr *github.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 403 || apiErr.StatusCode == 429) {
			fmt.Fprintln(os.Stderr, render.FormatError(apiErr.Message, stderrColor))
			return 3
		}

		// NotPullRequestError gets special rendering with icons/colors/hyperlinks
		var notPRErr *github.NotPullRequestError
		if errors.As(err, &notPRErr) {
			info := render.IssueWarning{
				Number: notPRErr.Number,
				Title:  notPRErr.Title,
				State:  notPRErr.State,
				URL:    notPRErr.URL,
			}
			info.RelatedPRs = notPRErr.RelatedPRs
			stderrHyperlinks := config.ShouldUseColorForFile("auto", os.Stderr)
			errRenderer := render.NewRenderer(os.Stderr, stderrColor, stderrHyperlinks)
			_ = errRenderer.RenderIssueWarning(info)
			return 1
		}

		fmt.Fprintln(os.Stderr, render.FormatError(err.Error(), stderrColor))
		return 1
	}

	renderer := render.NewRenderer(os.Stdout, useColor, useHyperlinks)

	if jsonOutput {
		if err := renderer.RenderJSON(status); err != nil {
			fmt.Fprintln(os.Stderr, render.FormatError("rendering output: "+err.Error(), stderrColor))
			return 1
		}
	} else {
		if err := renderer.RenderTable(status); err != nil {
			fmt.Fprintln(os.Stderr, render.FormatError("rendering output: "+err.Error(), stderrColor))
			return 1
		}
	}

	return 0
}
