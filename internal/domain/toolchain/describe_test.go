package toolchain

import (
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

var testGlyphs = Glyphs{Go: "G", C: "C"}

// only returns the single segment a one-language summary must produce.
func only(t *testing.T, summary Summary) Segment {
	t.Helper()
	got := Segments(summary, testGlyphs)
	if len(got) != 1 {
		t.Fatalf("Segments() = %+v, want exactly one", got)
	}
	return got[0]
}

func TestSegmentForGo(t *testing.T) {
	got := only(t, Summary{
		Languages:   []target.Language{target.LanguageGo},
		GoRequired:  "1.25.8",
		GoAvailable: "1.26.5",
	})
	if got.Text != "G go 1.25.8 → 1.26.5" {
		t.Fatalf("text = %q", got.Text)
	}
	if !got.Satisfied {
		t.Fatal("a host newer than required was reported as unsatisfied")
	}
}

func TestGoSegmentIsUnsatisfiedWhenTheHostIsBehind(t *testing.T) {
	got := only(t, Summary{
		Languages:   []target.Language{target.LanguageGo},
		GoRequired:  "1.27.0",
		GoAvailable: "1.26.5",
	})
	if got.Satisfied {
		t.Fatal("a host behind the requirement was reported as satisfied")
	}
}

// gcc and g++ ship together, so agreeing versions collapse to one description.
func TestSegmentForCCollapsesMatchingCompilers(t *testing.T) {
	got := only(t, Summary{
		Languages: []target.Language{target.LanguageC},
		GCC:       "16.1.1",
		GXX:       "16.1.1",
	})
	if got.Text != "C gcc/g++ 16.1.1" {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestSegmentForCSeparatesDifferingCompilers(t *testing.T) {
	got := only(t, Summary{
		Languages: []target.Language{target.LanguageC},
		GCC:       "16.1.1",
		GXX:       "15.0.0",
	})
	if !strings.Contains(got.Text, "gcc 16.1.1") || !strings.Contains(got.Text, "g++ 15.0.0") {
		t.Fatalf("text = %q, want both compilers named", got.Text)
	}
}

// Nothing declares a required gcc, so there is no requirement for the host to
// fall short of and the C cell never raises an alarm.
func TestCSegmentIsAlwaysSatisfied(t *testing.T) {
	got := only(t, Summary{Languages: []target.Language{target.LanguageC}})
	if !got.Satisfied {
		t.Fatal("the C cell warned although nothing declares a required gcc")
	}
}

// A mixed package reads C before Go, because Languages sorts alphabetically and
// Segments preserves that order.
func TestSegmentsAreAlphabeticalForAMixedPackage(t *testing.T) {
	got := Segments(Summary{
		Languages:   []target.Language{target.LanguageC, target.LanguageGo},
		GoRequired:  "1.25.8",
		GoAvailable: "1.26.5",
		GCC:         "16.1.1",
		GXX:         "16.1.1",
	}, testGlyphs)

	if len(got) != 2 {
		t.Fatalf("Segments() = %+v, want one cell per language", got)
	}
	if !strings.Contains(got[0].Text, "gcc") || !strings.Contains(got[1].Text, "go 1.25.8") {
		t.Fatalf("Segments() = %+v, want C first then Go", got)
	}
}

// One cell warning must not drag the other with it: the alarm names which
// compiler is behind.
func TestOnlyTheUnsatisfiedSegmentWarns(t *testing.T) {
	got := Segments(Summary{
		Languages:   []target.Language{target.LanguageC, target.LanguageGo},
		GoRequired:  "1.99.0",
		GoAvailable: "1.26.5",
		GCC:         "16.1.1",
		GXX:         "16.1.1",
	}, testGlyphs)

	if !got[0].Satisfied {
		t.Fatal("the C cell warned because Go was behind")
	}
	if got[1].Satisfied {
		t.Fatal("the Go cell did not warn although the host is behind")
	}
}

// A missing compiler reads as unknown rather than blank, so "not installed" is
// visibly different from "not applicable".
func TestSegmentsMarkAbsentToolchains(t *testing.T) {
	got := only(t, Summary{Languages: []target.Language{target.LanguageC}})
	if !strings.Contains(got.Text, unknownVersion) {
		t.Fatalf("text = %q, want an unknown marker", got.Text)
	}
}

func TestSegmentForGoWithNoRequirement(t *testing.T) {
	got := only(t, Summary{
		Languages:   []target.Language{target.LanguageGo},
		GoAvailable: "1.26.5",
	})
	if !strings.Contains(got.Text, "? → 1.26.5") {
		t.Fatalf("text = %q", got.Text)
	}
}

// Nothing to say means no header at all.
func TestSegmentsEmptyWhenNoLanguages(t *testing.T) {
	if got := Segments(Summary{}, testGlyphs); len(got) != 0 {
		t.Fatalf("Segments() = %+v, want none", got)
	}
	if got := Segments(Summary{Languages: []target.Language{}}, testGlyphs); len(got) != 0 {
		t.Fatalf("Segments() = %+v, want none", got)
	}
}

func TestSegmentsSkipUnknownLanguages(t *testing.T) {
	if got := Segments(Summary{Languages: []target.Language{"Fortran"}}, testGlyphs); len(got) != 0 {
		t.Fatalf("Segments() = %+v, want a language with no toolchain to be skipped", got)
	}
}

func BenchmarkSegments(b *testing.B) {
	summary := Summary{
		Languages:   []target.Language{target.LanguageC, target.LanguageGo},
		GoRequired:  "1.25.8",
		GoAvailable: "1.26.5",
		GCC:         "16.1.1",
		GXX:         "16.1.1",
	}
	for b.Loop() {
		Segments(summary, testGlyphs)
	}
}
