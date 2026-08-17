package toolchain

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Glyphs the description uses. They are passed in rather than imported from
// layout so this package stays free of presentation dependencies; the caller
// supplies whatever vocabulary it draws with.
type Glyphs struct {
	Go string
	C  string
}

// Segment is one language's entry in the toolchain header — its own framed cell
// rather than a run of text, so two toolchains never read as one sentence.
//
// Satisfied is per-segment on purpose: when the host Go is behind, that is news
// about Go, and reddening the C frame alongside it would be a false alarm.
type Segment struct {
	Text      string
	Satisfied bool
}

const unknownVersion = "?"

// Segments describes each language the package is written in, in the order they
// were given — which Languages already sorts alphabetically, so a mixed package
// always reads C before Go.
//
// It returns nothing when there is nothing to say, which is the signal to drop
// the header and give the rows back to the target list.
func Segments(summary Summary, glyphs Glyphs) []Segment {
	built := make([]Segment, 0, len(summary.Languages))
	for _, language := range summary.Languages {
		if segment, ok := segmentFor(summary, language, glyphs); ok {
			built = append(built, segment)
		}
	}
	return built
}

func segmentFor(summary Summary, language target.Language, glyphs Glyphs) (Segment, bool) {
	switch language {
	case target.LanguageGo:
		return Segment{Text: describeGo(summary, glyphs), Satisfied: summary.GoSatisfied()}, true
	case target.LanguageC:
		return Segment{Text: glyphs.C + " " + describeNative(summary), Satisfied: true}, true
	default:
		return Segment{}, false
	}
}

func describeGo(summary Summary, glyphs Glyphs) string {
	return glyphs.Go + " go " + versionOr(summary.GoRequired) + " → " + versionOr(summary.GoAvailable)
}

// describeNative collapses gcc and g++ into one segment when they agree, which
// they almost always do — they ship and version together.
func describeNative(summary Summary) string {
	if summary.GCC != "" && summary.GCC == summary.GXX {
		return "gcc/g++ " + summary.GCC
	}
	return "gcc " + versionOr(summary.GCC) + "  g++ " + versionOr(summary.GXX)
}

// versionOr reports an absent version as a question mark rather than as blank,
// so "not installed" is visibly different from "not applicable".
func versionOr(version string) string {
	if version == "" {
		return unknownVersion
	}
	return version
}
