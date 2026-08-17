package layout

import (
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/text"
)

func plain(line Line) string {
	var b strings.Builder
	for _, span := range line.Spans {
		b.WriteString(span.Text)
	}
	return b.String()
}

func sampleView() View {
	return View{
		Rows: []Row{
			{IsGroup: true, Text: "//src/go/apps/falcon", MatchStart: 14, MatchEnd: 20},
			{Text: "  linux · go_binary", MatchStart: -1, MatchEnd: -1},
			{Text: "  macos · go_binary", MatchStart: -1, MatchEnd: -1},
			{IsGroup: true, Text: "//src/go/tools/basalt/cmd/basalt", MatchStart: -1, MatchEnd: -1},
			{Text: "  basalt · go_binary", MatchStart: -1, MatchEnd: -1},
		},
		MatchCount:   3,
		TotalTargets: 1304,
		Actions:      []string{"Bazel > Run", "Bazel > Build", "Nano > README.md"},
		Query:        "falcon",
		ModeName:     "literal",
	}
}

// The invariant the whole renderer rests on: every line is exactly the
// terminal width, so a redraw fully overwrites the previous frame.
func TestRenderEveryLineIsExactlyTerminalWidth(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 120, 200, 41, 79, 137} {
		for _, height := range []int{8, 12, 24, 50} {
			frame := NewRenderer().Render(sampleView(), width, height)
			for i, line := range frame.Lines {
				if got := text.Width(plain(line)); got != width {
					t.Fatalf("at %dx%d line %d has width %d: %q", width, height, i, got, plain(line))
				}
			}
		}
	}
}

func TestRenderProducesExactlyTerminalHeight(t *testing.T) {
	for _, height := range []int{8, 10, 24, 50} {
		frame := NewRenderer().Render(sampleView(), 80, height)
		if len(frame.Lines) != height {
			t.Fatalf("at height %d got %d lines", height, len(frame.Lines))
		}
	}
}

func TestRenderShowsTargetCounts(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)
	if !strings.Contains(plain(frame.Lines[0]), "BAZEL TARGETS 3/1304") {
		t.Fatalf("title = %q", plain(frame.Lines[0]))
	}
}

func TestRenderShowsQueryAndMode(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)
	search := plain(frame.Lines[len(frame.Lines)-2])
	if !strings.Contains(search, "falcon") || !strings.Contains(search, "literal") {
		t.Fatalf("search line = %q", search)
	}
}

func TestRenderShowsActions(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)
	joined := ""
	for _, line := range frame.Lines {
		joined += plain(line)
	}
	for _, want := range []string{"Bazel > Run", "Nano > README.md", "TARGET ACTIONS"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("frame does not contain %q", want)
		}
	}
}

func TestRenderHighlightsTheMatch(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)
	found := false
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleMatch && strings.Contains(span.Text, "falcon") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no StyleMatch span carried the matched text")
	}
}

func TestRenderMarksTheSelectedRow(t *testing.T) {
	view := sampleView()
	view.RowCursor = 2 // the "macos" target, beneath the first heading
	frame := NewRenderer().Render(view, 80, 12)

	selected := 0
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleSelected {
				selected++
				if !strings.Contains(span.Text, "macos") {
					t.Fatalf("selected span = %q, want the second row", span.Text)
				}
			}
		}
	}
	if selected != 1 {
		t.Fatalf("%d selected spans, want exactly 1", selected)
	}
}

func TestRenderMarksTheSelectedActionOnlyWhenFocused(t *testing.T) {
	view := sampleView()
	view.ActionCursor = 1

	unfocused := NewRenderer().Render(view, 80, 12)
	for _, line := range unfocused.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleSelected && strings.Contains(span.Text, "Build") {
				t.Fatal("the action panel highlighted a row while unfocused")
			}
		}
	}

	view.Focus = FocusActions
	focused := NewRenderer().Render(view, 80, 12)
	found := false
	for _, line := range focused.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleSelected && strings.Contains(span.Text, "Build") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("the focused action panel did not highlight its row")
	}
}

func TestRenderScrollsWithRowOffset(t *testing.T) {
	view := sampleView()
	view.RowOffset = 3
	frame := NewRenderer().Render(view, 80, 12)
	body := plain(frame.Lines[1])
	if !strings.Contains(body, "//src/go/tools/basalt/cmd/basalt") {
		t.Fatalf("first body row = %q, want the second package heading", body)
	}
}

// The point of grouping: a package heading is visually distinct from the
// targets indented beneath it.
func TestRenderStylesGroupHeadingsDistinctly(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)

	// A highlighted heading splits into three spans — before, match, after — so
	// this counts lines carrying the style rather than spans.
	headings := 0
	for _, line := range frame.Lines {
		styled := false
		for _, span := range line.Spans {
			if span.Style == StyleGroup {
				styled = true
			}
		}
		if !styled {
			continue
		}
		headings++
		if !strings.Contains(plain(line), "//src/go/") {
			t.Fatalf("a group-styled line held %q, which is not a package path", plain(line))
		}
	}
	if headings != 2 {
		t.Fatalf("%d headings styled as groups, want 2", headings)
	}
}

func TestRenderStylesTargetsAndHeadingsDifferently(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleGroup && strings.Contains(span.Text, "· go_binary") {
				t.Fatalf("a target row was styled as a heading: %q", span.Text)
			}
		}
	}
}

// A heading is a label, not a choice: even if the cursor index lands on one it
// must not be drawn as the selection.
func TestRenderNeverHighlightsAHeadingAsSelected(t *testing.T) {
	view := sampleView()
	view.RowCursor = 0 // the first heading
	frame := NewRenderer().Render(view, 80, 12)

	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleSelected {
				t.Fatalf("a heading was drawn as the selection: %q", span.Text)
			}
		}
	}
}

func TestRenderHighlightsAMatchInAHeading(t *testing.T) {
	frame := NewRenderer().Render(sampleView(), 80, 12)
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleMatch && span.Text == "falcon" {
				return
			}
		}
	}
	t.Fatal("the matched run inside the package heading was not highlighted")
}

func TestRenderHandlesAnEmptyResultSet(t *testing.T) {
	view := sampleView()
	view.Rows = nil
	view.MatchCount = 0
	frame := NewRenderer().Render(view, 80, 12)
	if len(frame.Lines) != 12 {
		t.Fatalf("got %d lines", len(frame.Lines))
	}
	for i, line := range frame.Lines {
		if text.Width(plain(line)) != 80 {
			t.Fatalf("line %d is not 80 wide", i)
		}
	}
}

func TestRenderShowsAnErrorStatus(t *testing.T) {
	view := sampleView()
	view.Status = "bad regex"
	view.StatusIsError = true
	frame := NewRenderer().Render(view, 80, 12)

	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == StyleError && strings.Contains(span.Text, "bad regex") {
				return
			}
		}
	}
	t.Fatal("the error status was not rendered with StyleError")
}

func TestRenderRefusesATinyTerminal(t *testing.T) {
	for _, size := range [][2]int{{20, 24}, {80, 4}, {10, 3}, {39, 8}, {40, 7}} {
		frame := NewRenderer().Render(sampleView(), size[0], size[1])
		if len(frame.Lines) != 1 {
			t.Fatalf("at %dx%d got %d lines, want a single notice", size[0], size[1], len(frame.Lines))
		}
	}
}

// A target name carrying escape sequences must not reach the frame intact.
func TestRenderSanitizesHostileTargetNames(t *testing.T) {
	view := sampleView()
	view.Rows = []Row{
		{IsGroup: true, Text: "//src/evil\x1b[2J", MatchStart: -1, MatchEnd: -1},
		{Text: "  \x1b[?1049h\x07 · go_binary", MatchStart: -1, MatchEnd: -1},
	}
	frame := NewRenderer().Render(view, 80, 12)
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if strings.ContainsRune(span.Text, 0x1B) || strings.ContainsRune(span.Text, 0x07) {
				t.Fatalf("a control character reached the frame: %q", span.Text)
			}
		}
	}
}

func TestRenderSelectedRowIsAlsoSanitized(t *testing.T) {
	view := sampleView()
	view.RowCursor = 0
	view.Rows = []Row{{Text: "  b\x1b[2J · go_binary", MatchStart: -1, MatchEnd: -1}}
	frame := NewRenderer().Render(view, 80, 12)
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if strings.ContainsRune(span.Text, 0x1B) {
				t.Fatal("the selected row leaked an escape character")
			}
		}
	}
}

func TestComputeGeometryFillsTheWidth(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120, 200} {
		g := ComputeGeometry(width, 24, 0)
		if g.TargetsWidth+g.ActionsWidth+borderCells != width {
			t.Fatalf("at width %d: %d + %d + %d != %d", width, g.TargetsWidth, g.ActionsWidth, borderCells, width)
		}
		if g.ActionsWidth < minActionsWidth {
			t.Fatalf("at width %d the action panel collapsed to %d", width, g.ActionsWidth)
		}
	}
}

func TestTooSmall(t *testing.T) {
	cases := []struct {
		w, h int
		want bool
	}{
		{80, 24, false},
		{MinWidth, MinHeight, false},
		{MinWidth - 1, MinHeight, true},
		{MinWidth, MinHeight - 1, true},
		{0, 0, true},
	}
	for _, tc := range cases {
		if got := TooSmall(tc.w, tc.h); got != tc.want {
			t.Fatalf("TooSmall(%d, %d) = %t, want %t", tc.w, tc.h, got, tc.want)
		}
	}
}

func TestTitleBarFillsWidth(t *testing.T) {
	for _, width := range []int{5, 10, 30, 60} {
		if got := text.Width(TitleBar("SEARCH", width)); got != width {
			t.Fatalf("TitleBar(_, %d) has width %d", width, got)
		}
	}
}

func TestStyledCellIsAlwaysExactWidth(t *testing.T) {
	for _, width := range []int{1, 2, 5, 20, 27, 28, 80} {
		spans := StyledCell("//src/go/tools/basalt:basalt", width, 2, 6, StyleNormal, StyleMatch)
		total := 0
		for _, span := range spans {
			total += text.Width(span.Text)
		}
		if total != width {
			t.Fatalf("StyledCell(_, %d) produced width %d", width, total)
		}
	}
}

// Truncation invalidates the byte offsets, so the highlight must be dropped
// rather than applied to a string that has been cut and ellipsized.
func TestStyledCellDropsHighlightWhenTruncated(t *testing.T) {
	spans := StyledCell("//src/go/tools/basalt:basalt", 8, 2, 6, StyleNormal, StyleMatch)
	for _, span := range spans {
		if span.Style == StyleMatch {
			t.Fatal("a highlight was applied to a truncated cell")
		}
	}
}

func TestStyledCellDropsHighlightOnSanitizedText(t *testing.T) {
	spans := StyledCell("ab\x1bcd", 20, 0, 2, StyleNormal, StyleMatch)
	for _, span := range spans {
		if span.Style == StyleMatch {
			t.Fatal("a highlight survived on text whose bytes were rewritten")
		}
	}
}

// decoratedView is sampleView carrying everything the 0.4.0 decorations add:
// a toolchain header, per-row markers, and the double-width glyphs that suffix
// package headings and packaging targets.
func decoratedView() View {
	view := sampleView()
	view.Toolchain = []ToolchainSegment{
		{Text: MarkerC + " gcc/g++ 16.1.1"},
		{Text: MarkerGo + " go 1.25.8 → 1.26.5"},
	}
	view.Rows = []Row{
		{IsGroup: true, Marker: MarkerGroup, Text: "//src/go/apps/falcon", Suffix: MarkerGo + " " + MarkerC, MatchStart: -1, MatchEnd: -1},
		{Marker: "  " + MarkerPassed, MarkerStyle: StyleSuccess, Text: "linux · go_binary", MatchStart: -1, MatchEnd: -1},
		{Marker: "  " + MarkerFailed, MarkerStyle: StyleFailure, Text: "macos · go_binary", MatchStart: -1, MatchEnd: -1},
		{Marker: "  " + MarkerUnrun, MarkerStyle: StyleDim, Text: "falcon_rpm · pkg_rpm", Suffix: MarkerPackaging, MatchStart: -1, MatchEnd: -1},
	}
	return view
}

func TestRenderKeepsWidthWithGlyphsAndToolchain(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 137} {
		for _, height := range []int{8, 12, 24} {
			frame := NewRenderer().Render(decoratedView(), width, height)
			if len(frame.Lines) != height {
				t.Fatalf("at %dx%d got %d lines", width, height, len(frame.Lines))
			}
			for i, line := range frame.Lines {
				if got := text.Width(plain(line)); got != width {
					t.Fatalf("at %dx%d line %d has width %d: %q", width, height, i, got, plain(line))
				}
			}
		}
	}
}

func TestRenderShowsTheToolchainHeaderOnlyWhenPresent(t *testing.T) {
	with := plain(NewRenderer().Render(decoratedView(), 100, 24).Lines[1])
	if !strings.Contains(with, "gcc/g++ 16.1.1") || !strings.Contains(with, MarkerGo) {
		t.Fatalf("toolchain line = %q", with)
	}

	frame := NewRenderer().Render(sampleView(), 100, 24)
	for _, line := range frame.Lines {
		if strings.Contains(plain(line), "gcc") {
			t.Fatalf("a toolchain line was drawn for a view without one: %q", plain(line))
		}
	}
}

// Each toolchain is its own cell: a rule divides them on the header row, the
// border above carries a connector for it, and a rule beneath closes them off
// from the target list.
func TestRenderFramesEachToolchainSeparately(t *testing.T) {
	frame := NewRenderer().Render(decoratedView(), 100, 24)
	border, header, rule := plain(frame.Lines[0]), plain(frame.Lines[1]), plain(frame.Lines[2])

	divider := strings.Index(header, "│")
	if divider < 0 {
		t.Fatalf("the header cells are not divided: %q", header)
	}
	if !strings.Contains(border, "┬") {
		t.Fatalf("the border carries no connector for the divider: %q", border)
	}
	if !strings.HasPrefix(rule, "├") || !strings.Contains(rule, "┴") {
		t.Fatalf("the header is not closed off from the list: %q", rule)
	}
	if strings.Index(header, "gcc") > strings.Index(header, "go 1.25.8") {
		t.Fatalf("the cells are out of order: %q", header)
	}
}

// The header belongs to the target column, so the action panel keeps filling
// from the top rather than losing two rows to it.
func TestRenderKeepsTheActionPanelAlignedToTheTop(t *testing.T) {
	view := decoratedView()
	view.Actions = []string{"Bazel > Run", "Bazel > Build", "Nano > README.md"}
	frame := NewRenderer().Render(view, 100, 24)

	for i, want := range view.Actions {
		if got := plain(frame.Lines[i+1]); !strings.Contains(got, want) {
			t.Fatalf("line %d = %q, want it to carry %q", i+1, got, want)
		}
	}
}

// One toolchain warning must not redden the other's cell.
func TestRenderWarnsInOnlyTheUnsatisfiedCell(t *testing.T) {
	view := decoratedView()
	view.Toolchain[1].Warn = true
	frame := NewRenderer().Render(view, 100, 24)

	if !hasStyledText(frame, "go 1.25.8", StyleError) {
		t.Fatal("an unmet requirement was not drawn as an error")
	}
	if hasStyledText(frame, "gcc/g++", StyleError) {
		t.Fatal("a satisfied toolchain was reddened alongside an unsatisfied one")
	}
	if hasStyle(NewRenderer().Render(decoratedView(), 100, 24).Lines[1], StyleError) {
		t.Fatal("a satisfied header was drawn as an error")
	}
}

// A column too narrow to divide gets its rows back rather than cells too narrow
// to read.
func TestSegmentWidthsRefuseAColumnTooNarrowToDivide(t *testing.T) {
	if got := SegmentWidths(80, 0); got != nil {
		t.Fatalf("SegmentWidths(80, 0) = %v, want none", got)
	}
	if got := SegmentWidths(20, 2); got != nil {
		t.Fatalf("SegmentWidths(20, 2) = %v, want none — neither cell would be readable", got)
	}
}

// The cells and the rules between them fill the column exactly, which is what
// keeps the frame from coming out a column short.
func TestSegmentWidthsFillTheColumn(t *testing.T) {
	for _, total := range []int{31, 40, 65, 66, 120} {
		for _, count := range []int{1, 2, 3} {
			widths := SegmentWidths(total, count)
			if widths == nil {
				continue
			}
			sum := count - 1
			for _, width := range widths {
				sum += width
			}
			if sum != total {
				t.Fatalf("SegmentWidths(%d, %d) = %v, sums to %d", total, count, widths, sum)
			}
		}
	}
}

func TestRenderDrawsMarkersInTheirOwnStyle(t *testing.T) {
	frame := NewRenderer().Render(decoratedView(), 100, 24)

	cases := []struct {
		marker string
		style  Style
	}{
		{MarkerPassed, StyleSuccess},
		{MarkerFailed, StyleFailure},
		{MarkerUnrun, StyleDim},
	}
	for _, want := range cases {
		if !hasStyledText(frame, want.marker, want.style) {
			t.Errorf("marker %q was not drawn in style %v", want.marker, want.style)
		}
	}
}

func TestRenderKeepsSuffixGlyphsWhenTheNameFits(t *testing.T) {
	frame := NewRenderer().Render(decoratedView(), 100, 24)
	joined := framePlain(frame)

	for _, glyph := range []string{MarkerGroup, MarkerPackaging, MarkerGo, MarkerC} {
		if !strings.Contains(joined, glyph) {
			t.Errorf("glyph %q is missing from the frame", glyph)
		}
	}
}

// A pane too narrow to hold both a name and its glyph gives the name the room:
// an unreadable target is worse than a missing decoration.
func TestSuffixIsDroppedWhenTheNameWouldNotFit(t *testing.T) {
	entry := Row{Marker: "  " + MarkerUnrun, Suffix: MarkerPackaging}

	if got := suffixOf(entry, Geometry{TargetsWidth: minimumNameCells + 8}); got != " "+MarkerPackaging {
		t.Fatalf("suffix = %q, want it kept when the name has room", got)
	}
	if got := suffixOf(entry, Geometry{TargetsWidth: minimumNameCells + 5}); got != "" {
		t.Fatalf("suffix = %q, want it dropped when the name would be squeezed", got)
	}
	if got := suffixOf(Row{Marker: MarkerGroup}, Geometry{TargetsWidth: 80}); got != "" {
		t.Fatalf("suffix = %q, want empty for a row that carries none", got)
	}
}

func framePlain(frame Frame) string {
	var b strings.Builder
	for _, line := range frame.Lines {
		b.WriteString(plain(line))
		b.WriteByte('\n')
	}
	return b.String()
}

func hasStyle(line Line, style Style) bool {
	for _, span := range line.Spans {
		if span.Style == style {
			return true
		}
	}
	return false
}

func hasStyledText(frame Frame, want string, style Style) bool {
	for _, line := range frame.Lines {
		for _, span := range line.Spans {
			if span.Style == style && strings.Contains(span.Text, want) {
				return true
			}
		}
	}
	return false
}

func BenchmarkRender(b *testing.B) {
	renderer := NewRenderer()
	view := sampleView()
	b.ResetTimer()
	for b.Loop() {
		renderer.Render(view, 120, 40)
	}
}

func BenchmarkComputeGeometry(b *testing.B) {
	for b.Loop() {
		ComputeGeometry(120, 40, 0)
	}
}

func BenchmarkStyledCell(b *testing.B) {
	for b.Loop() {
		StyledCell("//src/go/tools/basalt:basalt", 60, 10, 16, StyleNormal, StyleMatch)
	}
}

func BenchmarkTitleBar(b *testing.B) {
	for b.Loop() {
		TitleBar("BAZEL TARGETS 3/1304", 80)
	}
}
