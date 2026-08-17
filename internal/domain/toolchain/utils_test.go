package toolchain

import (
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Verbatim output from the toolchains on the operator's machine, so the parsers
// are tested against what they will actually be handed.
const (
	realGoVersion  = "go version go1.26.5-X:nodwarf5 linux/amd64"
	realGCCVersion = "gcc (GCC) 16.1.1 20260515 (Red Hat 16.1.1-2)\nCopyright (C) 2026 Free Software Foundation, Inc."
)

func TestParseGoVersion(t *testing.T) {
	cases := map[string]string{
		realGoVersion:                          "1.26.5",
		"go version go1.25.8 linux/amd64":      "1.25.8",
		"go version go1.22 darwin/arm64":       "1.22",
		"go version devel go1.27-abc":          "1.27",
		"":                                     "",
		"something entirely unexpected":        "",
		"go version go1.26.5+custom linux/386": "1.26.5",
	}
	for output, want := range cases {
		if got := ParseGoVersion(output); got != want {
			t.Fatalf("ParseGoVersion(%q) = %q, want %q", output, got, want)
		}
	}
}

func TestParseGoDirective(t *testing.T) {
	cases := map[string]string{
		"module example.com/x\n\ngo 1.25.8\n\nrequire foo v1.0.0\n": "1.25.8",
		"go 1.26":                           "1.26",
		"  go 1.25.8  ":                     "1.25.8",
		"module example.com/x\n":            "",
		"":                                  "",
		"module gopher\nrequire golang x\n": "",
	}
	for contents, want := range cases {
		if got := ParseGoDirective(contents); got != want {
			t.Fatalf("ParseGoDirective(%q) = %q, want %q", contents, got, want)
		}
	}
}

// "go " must match the directive, not a module path that happens to start with
// the letters.
func TestParseGoDirectiveIgnoresLookalikes(t *testing.T) {
	if got := ParseGoDirective("module google.golang.org/x\ntoolchain go1.26.5\n"); got != "" {
		t.Fatalf("ParseGoDirective() = %q, want empty", got)
	}
}

func TestParseCompilerVersion(t *testing.T) {
	cases := map[string]string{
		realGCCVersion:                  "16.1.1",
		"g++ (GCC) 16.1.1 20260515":     "16.1.1",
		"clang version 19.1.0":          "19.1.0",
		"gcc (Debian 12.2.0-14) 12.2.0": "12.2.0",
		"":                              "",
		"no version here":               "",
	}
	for output, want := range cases {
		if got := ParseCompilerVersion(output); got != want {
			t.Fatalf("ParseCompilerVersion(%q) = %q, want %q", output, got, want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{"1.26.5", "1.25.8", 1},
		{"1.25.8", "1.26.5", -1},
		{"1.25.8", "1.25.8", 0},
		{"1.26", "1.26.0", 0},
		{"1.26.0", "1.26", 0},
		{"2.0", "1.99.99", 1},
		{"1.9", "1.10", -1},
		{"", "", 0},
	}
	for _, tc := range cases {
		if got := CompareVersions(tc.left, tc.right); got != tc.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

// The real situation on this machine: go.mod asks for 1.25.8, the host has
// 1.26.5. Newer satisfies older.
func TestSummarySatisfiedWhenHostIsNewer(t *testing.T) {
	summary := Summary{GoRequired: "1.25.8", GoAvailable: "1.26.5"}
	if !summary.GoSatisfied() {
		t.Fatal("a newer host toolchain should satisfy an older requirement")
	}
}

func TestSummaryNotSatisfiedWhenHostIsOlder(t *testing.T) {
	summary := Summary{GoRequired: "1.27.0", GoAvailable: "1.26.5"}
	if summary.GoSatisfied() {
		t.Fatal("an older host toolchain should not satisfy a newer requirement")
	}
}

// An unknown version on either side is reported, not alarmed about.
func TestSummarySatisfiedWhenUnknown(t *testing.T) {
	for _, summary := range []Summary{
		{GoRequired: "", GoAvailable: "1.26.5"},
		{GoRequired: "1.25.8", GoAvailable: ""},
		{},
	} {
		if !summary.GoSatisfied() {
			t.Fatalf("%+v should count as satisfied — basalt cannot substantiate a warning", summary)
		}
	}
}

func TestSummaryCarriesLanguages(t *testing.T) {
	summary := Summary{Languages: []target.Language{target.LanguageC, target.LanguageGo}}
	if len(summary.Languages) != 2 || summary.Languages[0] != target.LanguageC {
		t.Fatalf("Languages = %v, want C then Go", summary.Languages)
	}
}

func BenchmarkParseGoVersion(b *testing.B) {
	for b.Loop() {
		ParseGoVersion(realGoVersion)
	}
}

func BenchmarkParseGoDirective(b *testing.B) {
	contents := "module example.com/x\n\ngo 1.25.8\n\nrequire foo v1.0.0\n"
	for b.Loop() {
		ParseGoDirective(contents)
	}
}

func BenchmarkParseCompilerVersion(b *testing.B) {
	for b.Loop() {
		ParseCompilerVersion(realGCCVersion)
	}
}

func BenchmarkCompareVersions(b *testing.B) {
	for b.Loop() {
		CompareVersions("1.26.5", "1.25.8")
	}
}
