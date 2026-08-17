package tui

import (
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/grouping"
	"github.com/mateuszmierzwinski/basalt/internal/domain/keys"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolchain"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
)

// stubProvider returns the real registry's actions for a fixed environment, so
// the machine is exercised against genuine gating rather than a fake list.
type stubProvider struct {
	registry action.Registry
	markdown []string
}

func (p stubProvider) Actions(selected target.Target) []action.Action {
	all := make(map[toolset.Tool]toolset.Availability, len(toolset.KnownTools))
	for _, tool := range toolset.KnownTools {
		all[tool] = toolset.Availability{Tool: tool, Path: "/usr/bin/" + string(tool), Found: true}
	}
	return p.registry.For(action.Context{
		Target:    selected,
		Root:      "/repo",
		Directory: "/repo/src",
		Tools:     toolset.NewSet(all),
		Markdown:  p.markdown,
	})
}

// stubProber answers with a fixed host so the header is asserted against a
// known toolchain rather than whatever compilers the test machine has.
type stubProber struct {
	goRequired string
}

func (p stubProber) SummaryFor(_ target.Target, languages []target.Language) toolchain.Summary {
	return toolchain.Summary{
		Languages:   languages,
		GoRequired:  p.goRequired,
		GoAvailable: "1.26.5",
		GCC:         "16.1.1",
		GXX:         "16.1.1",
	}
}

func testTargets(t *testing.T) []target.Target {
	t.Helper()
	labels := []struct {
		kind  target.Kind
		label string
	}{
		{target.KindGoBinary, "//src/go/apps/falcon:linux"},
		{target.KindGoBinary, "//src/go/apps/falcon:macos"},
		{target.KindGoLibrary, "//src/go/tools/basalt/internal/domain/tui:tui"},
		{target.KindGoTest, "//src/go/tools/basalt/internal/domain/tui:tui_test"},
		{target.KindCcLibrary, "//src/c/mixer:engine"},
	}
	built := make([]target.Target, 0, len(labels))
	for _, entry := range labels {
		tgt, err := target.New(entry.kind, entry.label)
		if err != nil {
			t.Fatalf("target.New: %v", err)
		}
		built = append(built, tgt)
	}
	return built
}

func newTestMachine(t *testing.T, markdown ...string) (Machine, State) {
	t.Helper()
	m := NewMachine(filter.NewMatcher(), stubProvider{registry: action.NewRegistry(), markdown: markdown}, stubProber{})
	state := State{Targets: testTargets(t), Viewport: 3}
	return m, m.Refilter(state)
}

func typeString(m Machine, state State, s string) State {
	for _, r := range s {
		state, _ = m.Apply(state, keys.Key{Kind: keys.KindRune, Rune: r})
	}
	return state
}

func press(m Machine, state State, kind keys.Kind) (State, Outcome) {
	return m.Apply(state, keys.Key{Kind: kind})
}

// selectedLabel names what the cursor is on, which is what the tests actually
// care about — raw indices shift as headings are inserted.
func selectedLabel(t *testing.T, state State) string {
	t.Helper()
	selected, ok := state.Selected()
	if !ok {
		t.Fatalf("nothing is selected (cursor %d of %d rows)", state.Cursor, len(state.Display))
	}
	return selected.Label
}

func TestInitialStateShowsEverything(t *testing.T) {
	_, state := newTestMachine(t)
	if len(state.Matches) != 5 {
		t.Fatalf("matches = %d, want 5", len(state.Matches))
	}
	if state.Focus != layout.FocusTargets {
		t.Fatalf("focus = %v", state.Focus)
	}
	if _, ok := state.Selected(); !ok {
		t.Fatal("the initial cursor is not on a selectable row")
	}
}

// The list is a tree: one heading per package, targets indented beneath it.
func TestDisplayGroupsTargetsByPackage(t *testing.T) {
	_, state := newTestMachine(t)

	headings, targets := 0, 0
	for _, row := range state.Display {
		if row.Kind == grouping.KindGroup {
			headings++
			continue
		}
		targets++
	}
	if headings != 3 {
		t.Fatalf("%d headings, want 3 packages", headings)
	}
	if targets != 5 {
		t.Fatalf("%d target rows, want 5", targets)
	}
}

// A heading is a label, not a choice.
func TestCursorNeverRestsOnAHeading(t *testing.T) {
	m, state := newTestMachine(t)
	for i := 0; i < 12; i++ {
		if !state.Display[state.Cursor].IsSelectable() {
			t.Fatalf("cursor %d landed on a heading after %d moves", state.Cursor, i)
		}
		state, _ = press(m, state, keys.KindDown)
	}
	for i := 0; i < 12; i++ {
		if !state.Display[state.Cursor].IsSelectable() {
			t.Fatalf("cursor %d landed on a heading while moving up", state.Cursor)
		}
		state, _ = press(m, state, keys.KindUp)
	}
}

func TestTypingNarrowsTheList(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "falcon")
	if len(state.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(state.Matches))
	}
	if state.Query != "falcon" {
		t.Fatalf("query = %q", state.Query)
	}
}

func TestTypingWorksWhileTheActionPanelIsFocused(t *testing.T) {
	m, state := newTestMachine(t)
	state, _ = press(m, state, keys.KindTab)
	if state.Focus != layout.FocusActions {
		t.Fatal("Tab did not move focus")
	}
	state = typeString(m, state, "falcon")
	if state.Query != "falcon" || len(state.Matches) != 2 {
		t.Fatalf("typing while the action panel was focused did not reach the search field: %+v", state)
	}
}

func TestBackspaceWidensTheList(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "falcon")
	state, _ = press(m, state, keys.KindBackspace)
	if state.Query != "falco" {
		t.Fatalf("query = %q", state.Query)
	}
}

func TestClearLineEmptiesTheQuery(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "falcon")
	state, _ = press(m, state, keys.KindClearLine)
	if state.Query != "" || len(state.Matches) != 5 {
		t.Fatalf("state = %+v", state)
	}
}

func TestQueryIsLengthCapped(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, strings.Repeat("a", filter.MaxQueryLength+50))
	if len(state.Query) > filter.MaxQueryLength {
		t.Fatalf("query grew to %d, cap is %d", len(state.Query), filter.MaxQueryLength)
	}
}

func TestToggleModeSwitchesToRegex(t *testing.T) {
	m, state := newTestMachine(t)
	state, _ = press(m, state, keys.KindToggleMode)
	if state.Mode != filter.ModeRegex {
		t.Fatal("Ctrl+R did not switch to regex")
	}
	state = typeString(m, state, "(.*)falcon(.*)")
	if len(state.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(state.Matches))
	}
}

func TestInvalidRegexKeepsPreviousMatches(t *testing.T) {
	m, state := newTestMachine(t)
	state, _ = press(m, state, keys.KindToggleMode)
	state = typeString(m, state, "falcon")
	before := len(state.Matches)

	state = typeString(m, state, "(")
	if len(state.Matches) != before {
		t.Fatalf("matches changed to %d on an invalid pattern, want %d retained", len(state.Matches), before)
	}
	if !state.StatusIsError || state.Status == "" {
		t.Fatalf("no error status was set: %+v", state)
	}
}

// Down steps to the next target, over the heading between packages.
func TestCursorMovementStepsOverHeadings(t *testing.T) {
	m, state := newTestMachine(t)
	if got := selectedLabel(t, state); got != "//src/c/mixer:engine" {
		t.Fatalf("initial selection = %q", got)
	}
	state, _ = press(m, state, keys.KindDown)
	if got := selectedLabel(t, state); got != "//src/go/apps/falcon:linux" {
		t.Fatalf("after Down = %q, want the first target of the next package", got)
	}
	state, _ = press(m, state, keys.KindDown)
	if got := selectedLabel(t, state); got != "//src/go/apps/falcon:macos" {
		t.Fatalf("after a second Down = %q", got)
	}
	state, _ = press(m, state, keys.KindUp)
	if got := selectedLabel(t, state); got != "//src/go/apps/falcon:linux" {
		t.Fatalf("after Up = %q", got)
	}
}

func TestCursorStopsAtTheEnds(t *testing.T) {
	m, state := newTestMachine(t)
	for i := 0; i < 20; i++ {
		state, _ = press(m, state, keys.KindUp)
	}
	if got := selectedLabel(t, state); got != "//src/c/mixer:engine" {
		t.Fatalf("after many Ups = %q, want the first target", got)
	}
	for i := 0; i < 20; i++ {
		state, _ = press(m, state, keys.KindDown)
	}
	if got := selectedLabel(t, state); got != "//src/go/tools/basalt/internal/domain/tui:tui_test" {
		t.Fatalf("after many Downs = %q, want the last target", got)
	}
}

func TestHomeAndEnd(t *testing.T) {
	m, state := newTestMachine(t)
	state, _ = press(m, state, keys.KindEnd)
	if got := selectedLabel(t, state); got != "//src/go/tools/basalt/internal/domain/tui:tui_test" {
		t.Fatalf("End selected %q", got)
	}
	state, _ = press(m, state, keys.KindHome)
	if got := selectedLabel(t, state); got != "//src/c/mixer:engine" {
		t.Fatalf("Home selected %q", got)
	}
}

// Paging counts drawn rows, headings included, then lands on a target.
func TestPageKeysMoveAScreenAndLandOnATarget(t *testing.T) {
	m, state := newTestMachine(t)
	before := state.Cursor
	state, _ = press(m, state, keys.KindPageDown)

	if state.Cursor <= before {
		t.Fatalf("PageDown moved from %d to %d", before, state.Cursor)
	}
	if _, ok := state.Selected(); !ok {
		t.Fatal("PageDown left the cursor on a heading")
	}
	state, _ = press(m, state, keys.KindPageUp)
	if _, ok := state.Selected(); !ok {
		t.Fatal("PageUp left the cursor on a heading")
	}
}

// The heading directly above the selection is kept on screen, so it is always
// clear which package the highlighted target belongs to.
func TestScrollKeepsTheOwningHeadingVisible(t *testing.T) {
	m, state := newTestMachine(t)
	state.Viewport = 3
	state = m.Refilter(state)

	for i := 0; i < 6; i++ {
		state, _ = press(m, state, keys.KindDown)
		if state.Display[state.Cursor-1].Kind != grouping.KindGroup {
			continue
		}
		if state.Offset > state.Cursor-1 {
			t.Fatalf("heading at %d scrolled off with the cursor at %d and offset %d",
				state.Cursor-1, state.Cursor, state.Offset)
		}
	}
}

func TestScrollingKeepsCursorVisible(t *testing.T) {
	m, state := newTestMachine(t)
	state, _ = press(m, state, keys.KindEnd)
	if state.Cursor < state.Offset || state.Cursor >= state.Offset+state.Viewport {
		t.Fatalf("cursor %d outside window [%d, %d)", state.Cursor, state.Offset, state.Offset+state.Viewport)
	}
}

func TestActionsFollowTheSelectedTarget(t *testing.T) {
	m, state := newTestMachine(t)
	// The first target is a cc_library — no Run, and CLion rather than GoLand.
	if hasLabel(state.Actions, "Bazel > Run") {
		t.Fatalf("a cc_library must not offer Run, got %v", labels(state.Actions))
	}
	if !hasLabel(state.Actions, "Clion > Open as bazel project") {
		t.Fatalf("a cc_library should offer CLion, got %v", labels(state.Actions))
	}

	// Down lands on the falcon go_binary, which does offer Run.
	state, _ = press(m, state, keys.KindDown)
	if !hasLabel(state.Actions, "Bazel > Run") {
		t.Fatalf("a go_binary should offer Run, got %v", labels(state.Actions))
	}
	if !hasLabel(state.Actions, "Goland > Open as bazel project") {
		t.Fatalf("a go_binary should offer GoLand, got %v", labels(state.Actions))
	}
	if !hasLabel(state.Actions, "Bazel > Build") {
		t.Fatalf("every target should offer Build, got %v", labels(state.Actions))
	}
}

func TestActionsIncludeMarkdownFiles(t *testing.T) {
	_, state := newTestMachine(t, "README.md", "CLAUDE.md")
	if !hasLabel(state.Actions, "Nano > README.md") {
		t.Fatalf("Markdown files should appear as Nano actions, got %v", labels(state.Actions))
	}
}

func TestEnterMovesToActionsThenRuns(t *testing.T) {
	m, state := newTestMachine(t)
	state, outcome := press(m, state, keys.KindEnter)
	if outcome != OutcomeContinue || state.Focus != layout.FocusActions {
		t.Fatalf("first Enter should focus the actions, got %v / %v", outcome, state.Focus)
	}
	state, outcome = press(m, state, keys.KindEnter)
	if outcome != OutcomeRunAction {
		t.Fatalf("second Enter should run the action, got %v", outcome)
	}
	if state.Chosen == nil {
		t.Fatal("no action was chosen")
	}
}

func TestEnterDoesNothingWithNoMatches(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "nothing-matches-this")
	state, outcome := press(m, state, keys.KindEnter)
	if outcome != OutcomeContinue {
		t.Fatalf("outcome = %v, want Continue when nothing is selected", outcome)
	}
	if state.Chosen != nil {
		t.Fatal("an action was chosen with no target selected")
	}
}

func TestEscapeBacksOutInStages(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "falcon")
	state, _ = press(m, state, keys.KindTab)

	state, outcome := press(m, state, keys.KindEscape)
	if outcome != OutcomeContinue || state.Focus != layout.FocusTargets {
		t.Fatal("Escape should first return focus to the targets")
	}
	state, outcome = press(m, state, keys.KindEscape)
	if outcome != OutcomeContinue || state.Query != "" {
		t.Fatal("Escape should then clear the query")
	}
	_, outcome = press(m, state, keys.KindEscape)
	if outcome != OutcomeQuit {
		t.Fatalf("Escape on an empty query should quit, got %v", outcome)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, kind := range []keys.Kind{keys.KindInterrupt, keys.KindEOF} {
		m, state := newTestMachine(t)
		if _, outcome := press(m, state, kind); outcome != OutcomeQuit {
			t.Fatalf("key %v gave outcome %v, want Quit", kind, outcome)
		}
	}
}

func TestRefreshKey(t *testing.T) {
	m, state := newTestMachine(t)
	state, outcome := press(m, state, keys.KindRefresh)
	if outcome != OutcomeRefresh {
		t.Fatalf("outcome = %v, want Refresh", outcome)
	}
	if state.StatusIsError {
		t.Fatal("re-indexing is not an error state")
	}
}

func TestRedrawKey(t *testing.T) {
	m, state := newTestMachine(t)
	if _, outcome := press(m, state, keys.KindRedraw); outcome != OutcomeRedraw {
		t.Fatal("Ctrl+L should force a redraw")
	}
}

func TestUnboundKeyIsIgnored(t *testing.T) {
	m, state := newTestMachine(t)
	after, outcome := press(m, state, keys.KindUnknown)
	if outcome != OutcomeContinue {
		t.Fatalf("outcome = %v", outcome)
	}
	if after.Query != state.Query || after.Cursor != state.Cursor {
		t.Fatal("an unbound key changed the state")
	}
}

func TestViewProjectsState(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "falcon")
	view := m.View(state)

	if view.MatchCount != 2 || view.TotalTargets != 5 {
		t.Fatalf("counts = %d/%d, want 2/5", view.MatchCount, view.TotalTargets)
	}
	if view.Query != "falcon" || view.ModeName != "literal" {
		t.Fatalf("view = %+v", view)
	}
	// Two matching targets under one package heading.
	if len(view.Rows) != 3 {
		t.Fatalf("rows = %d, want 3 (one heading plus two targets)", len(view.Rows))
	}
	if !view.Rows[0].IsGroup {
		t.Fatalf("first row = %+v, want a package heading", view.Rows[0])
	}
	if view.Rows[0].MatchStart < 0 {
		t.Fatal("the heading lost the highlight span for a match in the path")
	}
	if view.Rows[1].IsGroup {
		t.Fatalf("second row = %+v, want a target", view.Rows[1])
	}
}

func TestViewOnEmptyResultSet(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "nothing-matches-this")
	view := m.View(state)
	if view.MatchCount != 0 || len(view.Rows) != 0 || len(view.Actions) != 0 {
		t.Fatalf("view = %+v", view)
	}
}

func labels(actions []action.Action) []string {
	got := make([]string, 0, len(actions))
	for _, a := range actions {
		got = append(got, a.Label())
	}
	return got
}

func hasLabel(actions []action.Action, want string) bool {
	for _, a := range actions {
		if a.Label() == want {
			return true
		}
	}
	return false
}

// rowFor finds the projected row whose text names the given target.
func rowFor(t *testing.T, view layout.View, name string) layout.Row {
	t.Helper()
	for _, row := range view.Rows {
		if strings.Contains(row.Text, name) {
			return row
		}
	}
	t.Fatalf("no row for %q in %v", name, view.Rows)
	return layout.Row{}
}

func TestViewMarksRunOutcomes(t *testing.T) {
	m, state := newTestMachine(t)
	state.Statuses = map[string]runlog.Status{
		"//src/go/apps/falcon:linux": runlog.StatusPassed,
		"//src/go/apps/falcon:macos": runlog.StatusFailed,
	}
	view := m.View(m.Refilter(state))

	cases := []struct {
		name   string
		marker string
		style  layout.Style
	}{
		{"linux", layout.MarkerPassed, layout.StyleSuccess},
		{"macos", layout.MarkerFailed, layout.StyleFailure},
		{"engine", layout.MarkerUnrun, layout.StyleDim},
	}
	for _, want := range cases {
		row := rowFor(t, view, want.name)
		if !strings.HasSuffix(row.Marker, want.marker) {
			t.Errorf("%s marker = %q, want it to end in %q", want.name, row.Marker, want.marker)
		}
		if row.MarkerStyle != want.style {
			t.Errorf("%s marker style = %v, want %v", want.name, row.MarkerStyle, want.style)
		}
	}
}

func TestViewMarksPackageHeadings(t *testing.T) {
	m, state := newTestMachine(t)
	view := m.View(state)

	for _, row := range view.Rows {
		if row.IsGroup && row.Marker != layout.MarkerGroup {
			t.Fatalf("group marker = %q, want %q", row.Marker, layout.MarkerGroup)
		}
	}
}

func TestViewSuffixesHeadingWithPackageLanguages(t *testing.T) {
	m, state := newTestMachine(t)
	view := m.View(state)

	cases := map[string]string{
		"//src/c/mixer":        layout.MarkerC,
		"//src/go/apps/falcon": layout.MarkerGo,
	}
	for pkg, want := range cases {
		if got := rowFor(t, view, pkg).Suffix; got != want {
			t.Errorf("%s suffix = %q, want %q", pkg, got, want)
		}
	}
}

func TestViewSuffixesPackagingTargets(t *testing.T) {
	m, state := newTestMachine(t)
	packaging, err := target.New(target.KindGoBinary, "//src/go/tools/basalt:basalt_rpm")
	if err != nil {
		t.Fatalf("target.New: %v", err)
	}
	state.Targets = append(state.Targets, packaging)
	view := m.View(m.Refilter(state))

	if got := rowFor(t, view, "basalt_rpm").Suffix; got != layout.MarkerPackaging {
		t.Fatalf("packaging suffix = %q, want %q", got, layout.MarkerPackaging)
	}
	if got := rowFor(t, view, "linux").Suffix; got != "" {
		t.Fatalf("ordinary target suffix = %q, want empty", got)
	}
}

// toolchainText joins the header cells, for assertions about what is said
// rather than about how it is divided.
func toolchainText(view layout.View) string {
	parts := make([]string, 0, len(view.Toolchain))
	for _, segment := range view.Toolchain {
		parts = append(parts, segment.Text)
	}
	return strings.Join(parts, " | ")
}

func TestViewDescribesTheSelectedPackagesToolchain(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "mixer")
	view := m.View(state)

	if len(view.Toolchain) != 1 {
		t.Fatalf("toolchain = %+v, want one cell for a C-only package", view.Toolchain)
	}
	if !strings.Contains(toolchainText(view), "gcc/g++ 16.1.1") {
		t.Fatalf("toolchain = %q, want the C compilers", toolchainText(view))
	}
	if view.Toolchain[0].Warn {
		t.Fatal("warned about a toolchain with no unmet requirement")
	}
}

// A package written in both languages gets a cell each, C first.
func TestViewFramesEachLanguageSeparately(t *testing.T) {
	m := NewMachine(filter.NewMatcher(), stubProvider{registry: action.NewRegistry()}, stubProber{})
	mixed, err := target.New(target.KindGoTest, "//src/c/mixer:bridge_test")
	if err != nil {
		t.Fatalf("target.New: %v", err)
	}
	state := m.Refilter(State{Targets: append(testTargets(t), mixed), Viewport: 8})
	state = typeString(m, state, "mixer")
	view := m.View(state)

	if len(view.Toolchain) != 2 {
		t.Fatalf("toolchain = %+v, want a cell per language", view.Toolchain)
	}
	if !strings.Contains(view.Toolchain[0].Text, "gcc") || !strings.Contains(view.Toolchain[1].Text, "go ") {
		t.Fatalf("toolchain = %+v, want C then Go", view.Toolchain)
	}
}

func TestViewWarnsWhenTheHostGoIsTooOld(t *testing.T) {
	m := NewMachine(filter.NewMatcher(), stubProvider{registry: action.NewRegistry()}, stubProber{goRequired: "1.99.0"})
	state := m.Refilter(State{Targets: testTargets(t), Viewport: 8})
	state = typeString(m, state, "falcon")
	view := m.View(state)

	if !strings.Contains(toolchainText(view), "go 1.99.0 → 1.26.5") {
		t.Fatalf("toolchain = %q, want required → available", toolchainText(view))
	}
	if !view.Toolchain[0].Warn {
		t.Fatal("did not warn although the host Go is older than required")
	}
}

func TestViewOmitsToolchainWithoutAProber(t *testing.T) {
	m := NewMachine(filter.NewMatcher(), stubProvider{registry: action.NewRegistry()}, nil)
	state := m.Refilter(State{Targets: testTargets(t), Viewport: 8})

	if view := m.View(state); len(view.Toolchain) != 0 {
		t.Fatalf("toolchain = %+v, want none without a prober", view.Toolchain)
	}
}

func TestViewOmitsToolchainWithoutASelection(t *testing.T) {
	m, state := newTestMachine(t)
	state = typeString(m, state, "nothing-matches-this")

	if view := m.View(state); len(view.Toolchain) != 0 {
		t.Fatalf("toolchain = %+v, want none with nothing selected", view.Toolchain)
	}
}

func BenchmarkApplyRune(b *testing.B) {
	m := NewMachine(filter.NewMatcher(), stubProvider{registry: action.NewRegistry()}, stubProber{})
	targets := make([]target.Target, 0, 1300)
	for i := 0; i < 1300; i++ {
		tgt, _ := target.New(target.KindGoLibrary, "//src/go/tools/basalt/internal/domain/tui:tui")
		targets = append(targets, tgt)
	}
	state := m.Refilter(State{Targets: targets, Viewport: 30})
	key := keys.Key{Kind: keys.KindRune, Rune: 'b'}
	b.ResetTimer()
	for b.Loop() {
		m.Apply(state, key)
	}
}
