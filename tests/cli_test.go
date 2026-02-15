package tests

import (
	"flag"
	"reflect"
	"testing"

	"github.com/thatsneat-dev/nprt/internal/cli"
)

func testFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("channels", "", "")
	fs.String("color", "auto", "")
	fs.Bool("json", false, "")
	fs.Int("timeline-pages", 3, "")
	fs.Bool("verbose", false, "")
	fs.Bool("version", false, "")
	return fs
}

func TestReorderArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flags before positional",
			args: []string{"--json", "476497"},
			want: []string{"--json", "476497"},
		},
		{
			name: "positional before flag",
			args: []string{"476497", "--json"},
			want: []string{"--json", "476497"},
		},
		{
			name: "flag with value after positional",
			args: []string{"476497", "--channels", "master"},
			want: []string{"--channels", "master", "476497"},
		},
		{
			name: "flag with equals syntax",
			args: []string{"476497", "--channels=master"},
			want: []string{"--channels=master", "476497"},
		},
		{
			name: "double dash stops reordering",
			args: []string{"--", "--json"},
			want: []string{"--", "--json"},
		},
		{
			name: "unknown flags stay in positionals",
			args: []string{"476497", "--unknown"},
			want: []string{"476497", "--unknown"},
		},
		{
			name: "mixed known and unknown",
			args: []string{"476497", "--json", "--unknown"},
			want: []string{"--json", "476497", "--unknown"},
		},
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := testFlagSet()
			got := cli.ReorderArgs(fs, tt.args)
			if got == nil {
				got = []string{}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReorderArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestHasUnknownFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no flags",
			args: []string{"476497"},
			want: "",
		},
		{
			name: "unknown flag",
			args: []string{"--unknown"},
			want: "--unknown",
		},
		{
			name: "bare dash",
			args: []string{"-"},
			want: "",
		},
		{
			name: "double dash",
			args: []string{"--"},
			want: "",
		},
		{
			name: "negative number not a flag",
			args: []string{"-1"},
			want: "",
		},
		{
			name: "multiple with one unknown",
			args: []string{"476497", "--bad"},
			want: "--bad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cli.HasUnknownFlags(tt.args)
			if got != tt.want {
				t.Errorf("HasUnknownFlags(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseFlagName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		arg      string
		wantName string
		wantVal  bool
	}{
		{
			name:     "single dash",
			arg:      "-json",
			wantName: "json",
			wantVal:  false,
		},
		{
			name:     "double dash",
			arg:      "--channels",
			wantName: "channels",
			wantVal:  false,
		},
		{
			name:     "with value",
			arg:      "--channels=master",
			wantName: "channels",
			wantVal:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotName, gotVal := cli.ParseFlagName(tt.arg)
			if gotName != tt.wantName || gotVal != tt.wantVal {
				t.Errorf("ParseFlagName(%q) = (%q, %v), want (%q, %v)",
					tt.arg, gotName, gotVal, tt.wantName, tt.wantVal)
			}
		})
	}
}
