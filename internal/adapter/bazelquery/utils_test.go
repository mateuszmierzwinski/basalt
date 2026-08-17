package bazelquery

import (
	"errors"
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Verbatim `bazel query 'kind(rule, //src/...:*)' --output=label_kind` output
// from the real monorepo, so the parser is tested against the actual format.
const realQueryOutput = `go_library rule //src/go/tools/probe/internal/domain/tui:tui
go_test rule //src/go/tools/probe/internal/domain/tui:tui_test
go_binary rule //src/go/tools/probe/cmd/probe:probe
cc_library rule //src/c/mixer:engine
cc_test rule //src/c/mixer:engine_test
genrule rule //src/go/tools/probe:probe_rpm
filegroup rule //src/go/lib/ollama:docs
alias rule //src/go/lib/ollama:client
pkg_rpm rule //src/c/sonic/packaging:rpm
`

func TestParseQueryOutput(t *testing.T) {
	got, err := ParseQueryOutput(realQueryOutput)
	if err != nil {
		t.Fatalf("ParseQueryOutput() error = %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("ParseQueryOutput() returned %d targets, want 9", len(got))
	}
	if got[0].Kind != target.KindGoLibrary || got[0].Name != "tui" {
		t.Fatalf("first target = %+v", got[0])
	}
	if got[2].Display != "[probe:probe]" {
		t.Fatalf("display = %q, want [probe:probe]", got[2].Display)
	}
}

func TestParseQueryOutputSkipsNoiseLines(t *testing.T) {
	noisy := "Loading: 0 packages loaded\n" + realQueryOutput + "\nINFO: Elapsed time: 0.9s\n"
	got, err := ParseQueryOutput(noisy)
	if err != nil {
		t.Fatalf("ParseQueryOutput() error = %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("ParseQueryOutput() returned %d targets, want 9", len(got))
	}
}

func TestParseQueryOutputHandlesBlankLines(t *testing.T) {
	got, err := ParseQueryOutput("\n\ngo_binary rule //src/a:b\n\n\n")
	if err != nil {
		t.Fatalf("ParseQueryOutput() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ParseQueryOutput() returned %d targets, want 1", len(got))
	}
}

func TestParseQueryOutputRejectsEmptyResult(t *testing.T) {
	for _, input := range []string{"", "   \n\n", "INFO: nothing here\nLoading: 0 packages"} {
		if _, err := ParseQueryOutput(input); !errors.Is(err, ErrNoTargets) {
			t.Fatalf("ParseQueryOutput(%q) error = %v, want ErrNoTargets", input, err)
		}
	}
}

func TestLastLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"INFO: loading\nERROR: no such package\n", "ERROR: no such package"},
		{"single", "single"},
		{"trailing\n\n\n", "trailing"},
		{"", "no output"},
		{"   \n  \n", "no output"},
	}
	for _, tc := range cases {
		if got := lastLine(tc.in); got != tc.want {
			t.Fatalf("lastLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func BenchmarkParseQueryOutput(b *testing.B) {
	for b.Loop() {
		ParseQueryOutput(realQueryOutput)
	}
}

// The real monorepo returns ~1300 targets; this sizes the parser against that.
func BenchmarkParseQueryOutputAtScale(b *testing.B) {
	large := strings.Repeat(realQueryOutput, 145)
	b.ResetTimer()
	for b.Loop() {
		ParseQueryOutput(large)
	}
}

func BenchmarkLastLine(b *testing.B) {
	stderr := strings.Repeat("INFO: progress\n", 50) + "ERROR: boom\n"
	for b.Loop() {
		lastLine(stderr)
	}
}
