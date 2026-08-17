package target

import (
	"errors"
	"strings"
	"testing"
)

func TestParseLabelKind(t *testing.T) {
	got, err := ParseLabelKind("go_binary rule //src/go/tools/basalt/cmd/basalt:basalt")
	if err != nil {
		t.Fatalf("ParseLabelKind() error = %v", err)
	}
	want := Target{
		Kind:    KindGoBinary,
		Package: "//src/go/tools/basalt/cmd/basalt",
		Name:    "basalt",
		Label:   "//src/go/tools/basalt/cmd/basalt:basalt",
		Display: "[basalt:basalt]",
	}
	if got != want {
		t.Fatalf("ParseLabelKind() = %+v, want %+v", got, want)
	}
}

func TestParseLabelKindToleratesExtraSpacing(t *testing.T) {
	got, err := ParseLabelKind("  go_library   rule   //src/go/lib/ollama:ollama  ")
	if err != nil {
		t.Fatalf("ParseLabelKind() error = %v", err)
	}
	if got.Name != "ollama" || got.Kind != KindGoLibrary {
		t.Fatalf("ParseLabelKind() = %+v", got)
	}
}

func TestParseLabelKindRejectsMalformedLines(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"empty", ""},
		{"blank", "   "},
		{"two fields", "go_binary //a:b"},
		{"four fields", "go_binary rule //a:b extra"},
		{"missing rule keyword", "go_binary source //a:b"},
		{"source file entry", "source file //a:b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseLabelKind(tc.line); !errors.Is(err, ErrMalformedQueryLine) {
				t.Fatalf("ParseLabelKind(%q) error = %v, want ErrMalformedQueryLine", tc.line, err)
			}
		})
	}
}

func TestSplitLabelRejectsMalformedLabels(t *testing.T) {
	cases := []struct {
		name  string
		label string
	}{
		{"relative", "src/go:thing"},
		{"single slash", "/src/go:thing"},
		{"no colon", "//src/go/tools/basalt"},
		{"empty name", "//src/go:"},
		{"two colons", "//src/go:a:b"},
		{"root package only", "//:thing"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := SplitLabel(tc.label); !errors.Is(err, ErrMalformedLabel) {
				t.Fatalf("SplitLabel(%q) error = %v, want ErrMalformedLabel", tc.label, err)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	cases := []struct {
		pkg, name, want string
	}{
		{"//src/go/apps/falcon", "linux", "[falcon:linux]"},
		{"//src/go/tools/basalt/cmd/basalt", "basalt", "[basalt:basalt]"},
		{"//src", "thing", "[src:thing]"},
	}
	for _, tc := range cases {
		if got := DisplayName(tc.pkg, tc.name); got != tc.want {
			t.Fatalf("DisplayName(%q, %q) = %q, want %q", tc.pkg, tc.name, got, tc.want)
		}
	}
}

func TestDirectoryResolvesUnderRoot(t *testing.T) {
	tgt, err := New(KindGoBinary, "//src/go/tools/basalt/cmd/basalt:basalt")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	got, err := Directory("/repo", tgt)
	if err != nil {
		t.Fatalf("Directory() error = %v", err)
	}
	if got != "/repo/src/go/tools/basalt/cmd/basalt" {
		t.Fatalf("Directory() = %q", got)
	}
}

// A poisoned cache is the realistic source of these: the cache is JSON in the
// repository root and anyone can edit it.
func TestDirectoryRejectsEscapingLabels(t *testing.T) {
	cases := []struct {
		name string
		pkg  string
	}{
		{"parent traversal", "//../../etc"},
		{"embedded traversal", "//src/go/../../../etc"},
		{"bare parent", "//.."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tgt := Target{Package: tc.pkg, Name: "passwd", Label: tc.pkg + ":passwd"}
			if _, err := Directory("/repo", tgt); !errors.Is(err, ErrLabelEscapesRoot) {
				t.Fatalf("Directory(%q) error = %v, want ErrLabelEscapesRoot", tc.pkg, err)
			}
		})
	}
}

func TestDirectoryRejectsSiblingPrefixOfRoot(t *testing.T) {
	tgt := Target{Package: "//../repo-evil", Name: "x", Label: "//../repo-evil:x"}
	if _, err := Directory("/repo", tgt); !errors.Is(err, ErrLabelEscapesRoot) {
		t.Fatalf("Directory() error = %v, want ErrLabelEscapesRoot", err)
	}
}

func BenchmarkParseLabelKind(b *testing.B) {
	for b.Loop() {
		ParseLabelKind("go_library rule //src/go/tools/probe/internal/domain/tui:tui")
	}
}

func BenchmarkSplitLabel(b *testing.B) {
	for b.Loop() {
		SplitLabel("//src/go/tools/probe/internal/domain/tui:tui")
	}
}

func BenchmarkDisplayName(b *testing.B) {
	for b.Loop() {
		DisplayName("//src/go/apps/falcon", "linux")
	}
}

func BenchmarkDirectory(b *testing.B) {
	tgt := Target{Package: "//src/go/tools/basalt/cmd/basalt"}
	for b.Loop() {
		Directory("/repo", tgt)
	}
}

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		New(KindGoLibrary, "//src/go/tools/probe/internal/domain/tui:tui")
	}
}

func BenchmarkParseLabelKindOverRealQueryOutput(b *testing.B) {
	lines := strings.Split(strings.TrimSpace(sampleQueryOutput), "\n")
	b.ResetTimer()
	for b.Loop() {
		for _, line := range lines {
			ParseLabelKind(line)
		}
	}
}

// Verbatim `bazel query --output=label_kind` output, kept as a fixture so the
// parser is exercised against the real format rather than an idealized one.
const sampleQueryOutput = `
go_library rule //src/go/tools/probe/internal/domain/tui:tui
go_test rule //src/go/tools/probe/internal/domain/tui:tui_test
go_binary rule //src/go/tools/probe/cmd/probe:probe
cc_library rule //src/c/mixer:engine
genrule rule //src/go/tools/probe:probe_rpm
alias rule //src/go/lib/ollama:ollama_alias
`
