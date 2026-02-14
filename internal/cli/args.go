// Package cli provides command-line argument parsing utilities.
package cli

import (
	"flag"
	"strconv"
	"strings"
)

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

// ParseFlagName extracts the flag name from a token like "-json" or "--channels=unstable".
func ParseFlagName(arg string) (name string, hasValue bool) {
	s := strings.TrimLeft(arg, "-")
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i], true
	}
	return s, false
}

// ReorderArgs moves all recognized flags (and their values) before positional
// arguments while preserving their relative order. Tokens after a standalone
// "--" are treated as positionals. Unknown flags are left in place and later
// detected as errors after flag parsing.
func ReorderArgs(fs *flag.FlagSet, args []string) []string {
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
			name, hasValue := ParseFlagName(a)
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

// HasUnknownFlags checks remaining args for unrecognized flags.
func HasUnknownFlags(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "-") && a != "-" && a != "--" {
			// Don't treat negative numbers as flags; let them be
			// caught by input validation with a clearer error.
			trimmed := strings.TrimLeft(a, "-")
			if _, err := strconv.Atoi(trimmed); err == nil {
				continue
			}
			return a
		}
	}
	return ""
}
