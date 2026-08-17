package grouping

import (
	"sort"
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

const (
	// kindSeparator divides a target's name from its rule kind.
	kindSeparator = " · "
	// noSpan marks a row with nothing highlighted.
	noSpan = -1
)

// Build arranges matches into package headers followed by their targets.
//
// The matches are sorted by label first. `bazel query` already returns them in
// that order and filtering preserves it, so this is normally a no-op — but
// sorting here means the tree is correct by construction rather than by relying
// on an upstream ordering that nothing enforces.
func Build(matches []filter.Match, statuses map[string]runlog.Status) []Row {
	ordered := sortedByLabel(matches)
	rows := make([]Row, 0, len(ordered)*2)
	currentPackage := ""
	for index, match := range ordered {
		if match.Target.Package != currentPackage {
			currentPackage = match.Target.Package
			rows = append(rows, groupRow(match, languagesFrom(ordered, index)))
		}
		rows = append(rows, targetRow(match, statuses[match.Target.Label]))
	}
	return rows
}

// languagesFrom collects the languages of one package's run of targets.
//
// The matches are already sorted by label, so a package occupies a contiguous
// run and this only has to walk forward until the package changes — rather than
// indexing the whole set up front on every keystroke.
func languagesFrom(ordered []filter.Match, start int) []target.Language {
	pkg := ordered[start].Target.Package
	kinds := make([]target.Kind, 0, 8)
	for i := start; i < len(ordered) && ordered[i].Target.Package == pkg; i++ {
		kinds = append(kinds, ordered[i].Target.Kind)
	}
	return target.Languages(kinds)
}

func sortedByLabel(matches []filter.Match) []filter.Match {
	ordered := make([]filter.Match, len(matches))
	copy(ordered, matches)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Target.Label < ordered[j].Target.Label
	})
	return ordered
}

// groupRow builds the header for a package, highlighting the part of the path
// the query matched.
func groupRow(match filter.Match, languages []target.Language) Row {
	start, end := PackageSpan(match)
	return Row{
		Kind:       KindGroup,
		Text:       match.Target.Package,
		MatchStart: start,
		MatchEnd:   end,
		Languages:  languages,
	}
}

// targetRow builds a target line: its name and its rule kind, tagged with how
// it last ran. The indentation that sets it in from its heading is carried by
// the renderer's marker, not by this text.
func targetRow(match filter.Match, status runlog.Status) Row {
	start, end := NameSpan(match)
	return Row{
		Kind:        KindTarget,
		Text:        match.Target.Name + kindSeparator + string(match.Target.Kind),
		MatchStart:  start,
		MatchEnd:    end,
		Status:      status,
		IsPackaging: target.IsPackaging(match.Target),
		Target:      match.Target,
	}
}

// PackageSpan maps a match's span on the full label onto the package path
// alone, returning no span when the match fell in the target name instead.
func PackageSpan(match filter.Match) (int, int) {
	packageLength := len(match.Target.Package)
	if match.Start < 0 || match.End > packageLength {
		return noSpan, noSpan
	}
	return match.Start, match.End
}

// NameSpan maps a match's span on the full label onto the target name alone,
// returning no span when the match fell in the package path instead.
//
// A query that straddles the colon — "falcon:lin" — matches neither half
// cleanly, so nothing is highlighted rather than highlighting the wrong run.
func NameSpan(match filter.Match) (int, int) {
	nameOffset := len(match.Target.Package) + len(":")
	if match.Start < nameOffset {
		return noSpan, noSpan
	}
	return match.Start - nameOffset, match.End - nameOffset
}

// Packages counts the distinct packages in a row list, for the status line.
func Packages(rows []Row) int {
	count := 0
	for _, row := range rows {
		if row.Kind == KindGroup {
			count++
		}
	}
	return count
}

// TargetCount counts the selectable rows.
func TargetCount(rows []Row) int {
	count := 0
	for _, row := range rows {
		if row.IsSelectable() {
			count++
		}
	}
	return count
}

// CommonPrefix returns the longest shared package prefix, which is useful for
// reporting what a filtered tree has narrowed down to.
func CommonPrefix(rows []Row) string {
	prefix := ""
	first := true
	for _, row := range rows {
		if row.Kind != KindGroup {
			continue
		}
		if first {
			prefix, first = row.Text, false
			continue
		}
		prefix = sharedPrefix(prefix, row.Text)
	}
	return prefix
}

func sharedPrefix(a, b string) string {
	limit := min(len(a), len(b))
	shared := 0
	for shared < limit && a[shared] == b[shared] {
		shared++
	}
	return strings.TrimSuffix(a[:shared], "/")
}
