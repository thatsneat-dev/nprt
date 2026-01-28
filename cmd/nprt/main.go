package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/config"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/core"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/github"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/logging"
	"github.com/taylrfnt/nixpkgs-pr-tracker/internal/render"
)

var version = "dev"

const usage = `Usage: nprt [options] <PR number | PR URL>

Track which nixpkgs channels contain a given pull request.

Arguments:
  PR number    A pull request number (e.g., 476497)
  PR URL       A full GitHub PR URL (e.g., https://github.com/NixOS/nixpkgs/pull/476497)

Options:
  --channels   Comma-separated list of channels to check (default: master,staging-next,nixpkgs-unstable,nixos-unstable-small,nixos-unstable)
  --color      Color output mode: auto, always, never (default: auto)
  --json       Output results as JSON
  --verbose    Show detailed progress and debug information
  --version    Print version and exit
  -h, --help   Show this help message

Environment:
  GITHUB_TOKEN  GitHub personal access token for higher rate limits
`

func main() {
	os.Exit(run())
}

func run() int {
	var (
		channelsFlag string
		colorMode    string
		jsonOutput   bool
		verbose      bool
		showVersion  bool
	)

	flag.StringVar(&channelsFlag, "channels", "", "Comma-separated list of channels to check")
	flag.StringVar(&colorMode, "color", "auto", "Color output: auto, always, never")
	flag.BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	flag.BoolVar(&verbose, "verbose", false, "Show detailed progress and debug information")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	if err := flag.CommandLine.Parse(reorderArgs(os.Args[1:])); err != nil {
		return 2
	}

	if showVersion {
		fmt.Printf("nprt version %s\n", version)
		return 0
	}

	// Compute color settings early so all errors can be styled
	useColor := config.ShouldUseColor(colorMode)
	useHyperlinks := config.IsTerminal()

	args := flag.Args()

	if unknown := hasUnknownFlags(args); unknown != "" {
		fmt.Fprintln(os.Stderr, render.FormatError("unknown flag "+unknown, useColor))
		return 2
	}

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	prNumber, err := config.ParsePRInput(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, render.FormatError(err.Error(), useColor))
		return 2
	}

	channels, err := config.ParseChannels(channelsFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, render.FormatError(err.Error(), useColor))
		return 2
	}

	token := config.GetGitHubToken()

	// Create logger based on verbose flag
	log := logging.New(verbose)
	defer func() { _ = log.Sync() }()

	log.Debug("fetching PR", zap.Int("pr", prNumber))

	client := github.NewClient(token, log)
	checker := core.NewChecker(client, log)

	// Set up context with signal handling for clean cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	status, err := checker.CheckPR(ctx, prNumber, channels)
	if err != nil {
		// 403 errors (rate limit, auth failure) get a distinct exit code
		var apiErr *github.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
			fmt.Fprintln(os.Stderr, render.FormatError(apiErr.Message, useColor))
			return 3
		}
		fmt.Fprintln(os.Stderr, render.FormatError(err.Error(), useColor))
		return 1
	}

	renderer := render.NewRenderer(os.Stdout, useColor, useHyperlinks)

	if jsonOutput {
		if err := renderer.RenderJSON(status); err != nil {
			fmt.Fprintln(os.Stderr, render.FormatError("rendering output: "+err.Error(), useColor))
			return 1
		}
	} else {
		if err := renderer.RenderTable(status); err != nil {
			fmt.Fprintln(os.Stderr, render.FormatError("rendering output: "+err.Error(), useColor))
			return 1
		}
	}

	return 0
}

func hasUnknownFlags(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "-") && a != "-" && a != "--" {
			return a
		}
	}
	return ""
}

// boolFlag matches the interface used by the standard flag package for boolean flags.
type boolFlag interface {
	flag.Value
	IsBoolFlag() bool
}

func isBoolFlag(f *flag.Flag) bool {
	if bf, ok := f.Value.(boolFlag); ok {
		return bf.IsBoolFlag()
	}
	return false
}

// parseFlagName extracts the flag name from a token like "-json" or "--channels=unstable".
func parseFlagName(arg string) (name string, hasValue bool) {
	s := strings.TrimLeft(arg, "-")
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i], true
	}
	return s, false
}

// reorderArgs moves all recognized flags (and their values) before positional
// arguments while preserving their relative order. Tokens after a standalone
// "--" are treated as positionals. Unknown flags are left in place and later
// detected as errors after flag parsing.
func reorderArgs(args []string) []string {
	fs := flag.CommandLine

	var flags []string
	var positionals []string

	i := 0
	for i < len(args) {
		a := args[i]

		if a == "--" {
			positionals = append(positionals, args[i:]...)
			break
		}

		if strings.HasPrefix(a, "-") && a != "-" {
			name, hasValue := parseFlagName(a)
			if f := fs.Lookup(name); f != nil {
				if hasValue || isBoolFlag(f) {
					flags = append(flags, a)
					i++
				} else if i+1 < len(args) {
					flags = append(flags, a, args[i+1])
					i += 2
				} else {
					flags = append(flags, a)
					i++
				}
				continue
			}
		}

		positionals = append(positionals, a)
		i++
	}

	return append(flags, positionals...)
}
