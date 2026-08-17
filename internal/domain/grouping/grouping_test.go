package grouping

import (
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

func match(t *testing.T, kind target.Kind, label string, start, end int) filter.Match {
	t.Helper()
	tgt, err := target.New(kind, label)
	if err != nil {
		t.Fatalf("target.New(%q): %v", label, err)
	}
	return filter.Match{Target: tgt, Start: start, End: end}
}

func unmatched(t *testing.T, kind target.Kind, label string) filter.Match {
	t.Helper()
	return match(t, kind, label, -1, -1)
}

// buildRows groups matches with no run history, which is the common case in
// these tests — the status-specific cases pass a map explicitly.
func buildRows(matches []filter.Match) []Row {
	return Build(matches, nil)
}

func kinds(rows []Row) string {
	var b strings.Builder
	for _, row := range rows {
		if row.Kind == KindGroup {
			b.WriteByte('G')
			continue
		}
		b.WriteByte('T')
	}
	return b.String()
}

func TestBuildGroupsTargetsUnderTheirPackage(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoBinary, "//src/go/apps/falcon:linux"),
		unmatched(t, target.KindGoBinary, "//src/go/apps/falcon:macos"),
		unmatched(t, target.KindGoBinary, "//src/go/tools/basalt/cmd/basalt:basalt"),
	})
	if got := kinds(rows); got != "GTTGT" {
		t.Fatalf("row shape = %q, want GTTGT", got)
	}
	if rows[0].Text != "//src/go/apps/falcon" {
		t.Fatalf("first header = %q", rows[0].Text)
	}
	if rows[3].Text != "//src/go/tools/basalt/cmd/basalt" {
		t.Fatalf("second header = %q", rows[3].Text)
	}
}

// The row text is the target alone — the indentation that sets it in from its
// heading is carried by the renderer's marker, which keeps the match offsets
// measured against the name rather than against decoration.
func TestBuildComposesTargetTextWithoutDecoration(t *testing.T) {
	rows := Build([]filter.Match{unmatched(t, target.KindGoBinary, "//src/go/a:thing")}, nil)
	if rows[1].Text != "thing · go_binary" {
		t.Fatalf("target row = %q", rows[1].Text)
	}
}

func TestBuildEmitsOneHeaderPerPackage(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoLibrary, "//src/go/a:one"),
		unmatched(t, target.KindGoLibrary, "//src/go/a:two"),
		unmatched(t, target.KindGoLibrary, "//src/go/a:three"),
	})
	if got := kinds(rows); got != "GTTT" {
		t.Fatalf("row shape = %q, want GTTT", got)
	}
	if Packages(rows) != 1 {
		t.Fatalf("Packages() = %d, want 1", Packages(rows))
	}
}

// The tree must be correct even if the matches arrive interleaved, which is why
// Build sorts rather than trusting the caller's ordering.
func TestBuildSortsSoPackagesAreNotSplit(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoLibrary, "//src/go/b:one"),
		unmatched(t, target.KindGoLibrary, "//src/go/a:one"),
		unmatched(t, target.KindGoLibrary, "//src/go/b:two"),
		unmatched(t, target.KindGoLibrary, "//src/go/a:two"),
	})
	if got := kinds(rows); got != "GTTGTT" {
		t.Fatalf("row shape = %q, want GTTGTT — a package was split", got)
	}
	if Packages(rows) != 2 {
		t.Fatalf("Packages() = %d, want 2", Packages(rows))
	}
}

func TestBuildOnEmptyInput(t *testing.T) {
	if rows := buildRows(nil); len(rows) != 0 {
		t.Fatalf("Build(nil) = %v", rows)
	}
}

func TestTargetCount(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoLibrary, "//src/go/a:one"),
		unmatched(t, target.KindGoLibrary, "//src/go/b:two"),
	})
	if got := TargetCount(rows); got != 2 {
		t.Fatalf("TargetCount() = %d, want 2", got)
	}
}

func TestRowsCarryTheirTarget(t *testing.T) {
	rows := Build([]filter.Match{unmatched(t, target.KindGoBinary, "//src/go/a:thing")}, nil)
	if rows[0].IsSelectable() {
		t.Fatal("a header must not be selectable")
	}
	if !rows[1].IsSelectable() {
		t.Fatal("a target row must be selectable")
	}
	if rows[1].Target.Label != "//src/go/a:thing" {
		t.Fatalf("target row carries %q", rows[1].Target.Label)
	}
}

// A query matching the path highlights the header; one matching the name
// highlights the target. This is what makes the tree readable while filtering.
func TestMatchInThePathHighlightsTheHeader(t *testing.T) {
	// "apps" sits inside the package half of "//src/go/apps/falcon:linux".
	label := "//src/go/apps/falcon:linux"
	start := strings.Index(label, "apps")
	rows := Build([]filter.Match{match(t, target.KindGoBinary, label, start, start+len("apps"))}, nil)

	header := rows[0]
	if header.MatchStart != start || header.Text[header.MatchStart:header.MatchEnd] != "apps" {
		t.Fatalf("header span [%d:%d] of %q", header.MatchStart, header.MatchEnd, header.Text)
	}
	if rows[1].MatchStart != -1 {
		t.Fatalf("the target row should carry no span, got %d", rows[1].MatchStart)
	}
}

func TestMatchInTheNameHighlightsTheTarget(t *testing.T) {
	label := "//src/go/apps/falcon:linux"
	start := strings.Index(label, "linux")
	rows := Build([]filter.Match{match(t, target.KindGoBinary, label, start, start+len("linux"))}, nil)

	if rows[0].MatchStart != -1 {
		t.Fatalf("the header should carry no span, got %d", rows[0].MatchStart)
	}
	row := rows[1]
	if row.Text[row.MatchStart:row.MatchEnd] != "linux" {
		t.Fatalf("target span [%d:%d] of %q", row.MatchStart, row.MatchEnd, row.Text)
	}
}

// A query straddling the colon belongs to neither half; highlighting either one
// would mark the wrong run, so nothing is highlighted.
func TestMatchAcrossTheColonHighlightsNothing(t *testing.T) {
	label := "//src/go/a:thing"
	start := strings.Index(label, "a:thing")
	rows := Build([]filter.Match{match(t, target.KindGoBinary, label, start, start+len("a:thing"))}, nil)

	if rows[0].MatchStart != -1 || rows[1].MatchStart != -1 {
		t.Fatalf("spans = header %d, target %d; want none", rows[0].MatchStart, rows[1].MatchStart)
	}
}

func TestUnmatchedRowsCarryNoSpan(t *testing.T) {
	rows := Build([]filter.Match{unmatched(t, target.KindGoBinary, "//src/go/a:thing")}, nil)
	for i, row := range rows {
		if row.MatchStart != -1 || row.MatchEnd != -1 {
			t.Fatalf("row %d carries a span [%d:%d]", i, row.MatchStart, row.MatchEnd)
		}
	}
}

// A target carries the outcome of its last run, so the list can be coloured by
// what is currently building and what is not.
func TestRowsCarryTheirRunStatus(t *testing.T) {
	rows := Build([]filter.Match{
		unmatched(t, target.KindGoBinary, "//src/go/a:passed"),
		unmatched(t, target.KindGoBinary, "//src/go/a:failed"),
		unmatched(t, target.KindGoBinary, "//src/go/a:never"),
	}, map[string]runlog.Status{
		"//src/go/a:passed": runlog.StatusPassed,
		"//src/go/a:failed": runlog.StatusFailed,
	})

	statuses := map[string]runlog.Status{}
	for _, row := range rows {
		if row.IsSelectable() {
			statuses[row.Target.Name] = row.Status
		}
	}
	if statuses["passed"] != runlog.StatusPassed {
		t.Fatalf("passed target has status %v", statuses["passed"])
	}
	if statuses["failed"] != runlog.StatusFailed {
		t.Fatalf("failed target has status %v", statuses["failed"])
	}
	if statuses["never"] != runlog.StatusUnknown {
		t.Fatalf("a target that never ran has status %v, want unknown", statuses["never"])
	}
}

// A heading is not a target and has no outcome of its own.
func TestHeadingsCarryNoStatus(t *testing.T) {
	rows := Build([]filter.Match{unmatched(t, target.KindGoBinary, "//src/go/a:thing")},
		map[string]runlog.Status{"//src/go/a:thing": runlog.StatusFailed})

	if rows[0].Status != runlog.StatusUnknown {
		t.Fatalf("heading carries status %v", rows[0].Status)
	}
	if rows[1].Status != runlog.StatusFailed {
		t.Fatalf("target carries status %v, want failed", rows[1].Status)
	}
}

func TestCommonPrefix(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoLibrary, "//src/go/tools/basalt:a"),
		unmatched(t, target.KindGoLibrary, "//src/go/tools/packer:b"),
	})
	if got := CommonPrefix(rows); got != "//src/go/tools" {
		t.Fatalf("CommonPrefix() = %q", got)
	}
}

func TestCommonPrefixOfOnePackage(t *testing.T) {
	rows := Build([]filter.Match{unmatched(t, target.KindGoLibrary, "//src/go/a:one")}, nil)
	if got := CommonPrefix(rows); got != "//src/go/a" {
		t.Fatalf("CommonPrefix() = %q", got)
	}
}

func TestCommonPrefixOfNothing(t *testing.T) {
	if got := CommonPrefix(nil); got != "" {
		t.Fatalf("CommonPrefix(nil) = %q", got)
	}
}

func benchMatches(count int) []filter.Match {
	built := make([]filter.Match, 0, count)
	for i := 0; i < count; i++ {
		tgt, _ := target.New(target.KindGoLibrary, "//src/go/tools/basalt/internal/domain/grouping:grouping")
		built = append(built, filter.Match{Target: tgt, Start: -1, End: -1})
	}
	return built
}

// Sized to the real monorepo: this runs on every keystroke.
func BenchmarkBuildAtScale(b *testing.B) {
	matches := benchMatches(1350)
	b.ResetTimer()
	for b.Loop() {
		Build(matches, nil)
	}
}

func BenchmarkPackageSpan(b *testing.B) {
	tgt, _ := target.New(target.KindGoBinary, "//src/go/apps/falcon:linux")
	m := filter.Match{Target: tgt, Start: 9, End: 13}
	for b.Loop() {
		PackageSpan(m)
	}
}

func BenchmarkNameSpan(b *testing.B) {
	tgt, _ := target.New(target.KindGoBinary, "//src/go/apps/falcon:linux")
	m := filter.Match{Target: tgt, Start: 21, End: 26}
	for b.Loop() {
		NameSpan(m)
	}
}

// A heading carries the languages of everything beneath it, alphabetically, so
// a package holding both Go and C reads C first however its targets are
// ordered.
func TestBuildRecordsAPackagesLanguages(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoTest, "//src/c/audio:bridge_test"),
		unmatched(t, target.KindCcLibrary, "//src/c/audio:engine"),
		unmatched(t, target.KindGoBinary, "//src/go/demo:demo"),
	})

	cases := []struct {
		heading string
		want    []target.Language
	}{
		{"//src/c/audio", []target.Language{target.LanguageC, target.LanguageGo}},
		{"//src/go/demo", []target.Language{target.LanguageGo}},
	}
	for _, want := range cases {
		got := languagesOfHeading(t, rows, want.heading)
		if len(got) != len(want.want) {
			t.Fatalf("%s languages = %v, want %v", want.heading, got, want.want)
		}
		for i := range got {
			if got[i] != want.want[i] {
				t.Fatalf("%s languages = %v, want %v", want.heading, got, want.want)
			}
		}
	}
}

func languagesOfHeading(t *testing.T, rows []Row, heading string) []target.Language {
	t.Helper()
	for _, row := range rows {
		if row.Kind == KindGroup && strings.Contains(row.Text, heading) {
			return row.Languages
		}
	}
	t.Fatalf("no heading for %q", heading)
	return nil
}

func TestBuildFlagsPackagingTargets(t *testing.T) {
	rows := buildRows([]filter.Match{
		unmatched(t, target.KindGoBinary, "//src/go/tools/basalt:basalt"),
		unmatched(t, target.KindGoBinary, "//src/go/tools/basalt:basalt_rpm"),
	})

	for _, row := range rows {
		if row.Kind == KindGroup {
			continue
		}
		want := strings.Contains(row.Text, "basalt_rpm")
		if row.IsPackaging != want {
			t.Errorf("%q IsPackaging = %v, want %v", row.Text, row.IsPackaging, want)
		}
	}
}

func BenchmarkTargetCount(b *testing.B) {
	rows := Build(benchMatches(1350), nil)
	b.ResetTimer()
	for b.Loop() {
		TargetCount(rows)
	}
}

func BenchmarkPackages(b *testing.B) {
	rows := Build(benchMatches(1350), nil)
	b.ResetTimer()
	for b.Loop() {
		Packages(rows)
	}
}

func BenchmarkCommonPrefix(b *testing.B) {
	rows := Build(benchMatches(1350), nil)
	b.ResetTimer()
	for b.Loop() {
		CommonPrefix(rows)
	}
}
