package tui

import (
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/grouping"
	"github.com/mateuszmierzwinski/basalt/internal/domain/keys"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolchain"
)

type machine struct {
	matcher  filter.Matcher
	provider ActionsProvider
	prober   ToolchainProber
}

// NewMachine builds the state machine, returning the interface (Rule 7.2).
//
// prober may be nil: without it the toolchain header is simply absent, which is
// how the machine behaves in every test that does not care about compilers.
func NewMachine(matcher filter.Matcher, provider ActionsProvider, prober ToolchainProber) Machine {
	handlersOnce.Do(loadHandlers)
	return &machine{matcher: matcher, provider: provider, prober: prober}
}

// Apply advances the state by one keystroke. A printable character always goes
// to the search field regardless of which panel has focus, so narrowing the
// list never costs a keystroke; everything else is dispatched through the
// binding map.
func (m *machine) Apply(state State, key keys.Key) (State, Outcome) {
	if key.IsRune() {
		return m.typeRune(state, key.Rune)
	}
	handler, bound := handlers[key.Kind]
	if !bound {
		return state, OutcomeContinue
	}
	return handler.Handle(m, state)
}

// typeRune appends to the query, refusing once the query hits the length cap
// so a held-down key cannot grow it without bound.
func (m *machine) typeRune(state State, r rune) (State, Outcome) {
	if len(state.Query)+len(string(r)) > filter.MaxQueryLength {
		return state, OutcomeContinue
	}
	state.Query += string(r)
	return m.Refilter(state), OutcomeContinue
}

// Refilter recomputes the match set. A regex that does not compile leaves the
// previous matches on screen and reports the error in the status line, because
// a half-typed pattern is the normal state while the operator is typing.
func (m *machine) Refilter(state State) State {
	result := m.matcher.Filter(state.Targets, state.Query, state.Mode)
	if result.Err != nil {
		state.Status = result.Err.Error()
		state.StatusIsError = true
		return state
	}
	state.Matches = result.Matches
	state.Display = grouping.Build(result.Matches, state.Statuses)
	state.Status = ""
	state.StatusIsError = false
	state.Cursor = EnsureSelectable(state.Display, state.Cursor)
	return m.afterCursorMove(state)
}

// afterCursorMove keeps the viewport and the action panel in step with the
// selection.
//
// The offset is computed twice on purpose: once to guarantee the cursor is on
// screen, then again to pull in the package heading directly above it when that
// heading would otherwise have scrolled off. The cursor's visibility wins,
// because a selection you cannot see is worse than a heading you cannot see.
func (m *machine) afterCursorMove(state State) State {
	state.Offset = ScrollOffset(state.Offset, state.Cursor, state.Viewport, len(state.Display))
	if anchor := ScrollAnchor(state.Display, state.Cursor, state.Viewport); anchor < state.Offset {
		state.Offset = anchor
	}
	return m.refreshActions(state)
}

// refreshActions repopulates the action panel for the selected target, which is
// what makes the panel adapt to the target's rule kind and its files.
func (m *machine) refreshActions(state State) State {
	selected, ok := state.Selected()
	if !ok {
		state.Actions = nil
		state.ActionCursor = 0
		return state
	}
	state.Actions = m.provider.Actions(selected)
	state.ActionCursor = ClampCursor(state.ActionCursor, len(state.Actions))
	return state
}

// View projects state into the renderer's input.
func (m *machine) View(state State) layout.View {
	return layout.View{
		Rows:          rows(state),
		Toolchain:     m.toolchainHeader(state),
		RowOffset:     state.Offset,
		RowCursor:     state.Cursor,
		MatchCount:    len(state.Matches),
		TotalTargets:  len(state.Targets),
		Actions:       actionLabels(state.Actions),
		ActionCursor:  state.ActionCursor,
		Query:         state.Query,
		ModeName:      ModeName(state.Mode),
		Focus:         state.Focus,
		Status:        state.Status,
		StatusIsError: state.StatusIsError,
	}
}

// rows projects the package tree into the renderer's row list. The domain has
// already composed and indented the text, so this only carries it across.
func rows(state State) []layout.Row {
	built := make([]layout.Row, 0, len(state.Display))
	for _, row := range state.Display {
		marker, markerStyle := markerFor(row)
		built = append(built, layout.Row{
			IsGroup:     row.Kind == grouping.KindGroup,
			Marker:      marker,
			MarkerStyle: markerStyle,
			Text:        row.Text,
			Suffix:      suffixFor(row),
			Style:       styleForStatus(row.Status),
			MatchStart:  row.MatchStart,
			MatchEnd:    row.MatchEnd,
		})
	}
	return built
}

// suffixFor is the decoration trailing a row: the languages a package is
// written in, or a marker that a target produces a distributable artifact.
func suffixFor(row grouping.Row) string {
	if row.Kind == grouping.KindGroup {
		return languageGlyphs(row.Languages)
	}
	if row.IsPackaging {
		return layout.MarkerPackaging
	}
	return ""
}

// languageGlyphs renders a package's languages in the order given, which
// Languages has already sorted alphabetically.
func languageGlyphs(languages []target.Language) string {
	glyphs := make([]string, 0, len(languages))
	for _, language := range languages {
		switch language {
		case target.LanguageC:
			glyphs = append(glyphs, layout.MarkerC)
		case target.LanguageGo:
			glyphs = append(glyphs, layout.MarkerGo)
		}
	}
	return strings.Join(glyphs, " ")
}

// toolchainHeader describes the compilers for whatever is selected — one framed
// cell per language, each carrying whether the host falls short of what that
// language's project asks for.
//
// The languages come from the selected target's *package*, not from the target
// itself: a package holding both Go and C rules is a package you need both
// toolchains for, whichever of its targets the cursor happens to be on.
func (m *machine) toolchainHeader(state State) []layout.ToolchainSegment {
	selected, ok := state.Selected()
	if !ok || m.prober == nil {
		return nil
	}
	return headerSegments(m.prober.SummaryFor(selected, languagesOfEnclosingGroup(state)))
}

func headerSegments(summary toolchain.Summary) []layout.ToolchainSegment {
	described := toolchain.Segments(summary, headerGlyphs)
	built := make([]layout.ToolchainSegment, 0, len(described))
	for _, segment := range described {
		built = append(built, layout.ToolchainSegment{Text: segment.Text, Warn: !segment.Satisfied})
	}
	return built
}

// headerGlyphs hands the domain the vocabulary the renderer draws with, so the
// description stays free of presentation imports.
var headerGlyphs = toolchain.Glyphs{Go: layout.MarkerGo, C: layout.MarkerC}

// languagesOfEnclosingGroup walks back from the cursor to its package heading,
// which is the row carrying the package's languages.
func languagesOfEnclosingGroup(state State) []target.Language {
	for index := state.Cursor; index >= 0 && index < len(state.Display); index-- {
		if state.Display[index].Kind == grouping.KindGroup {
			return state.Display[index].Languages
		}
	}
	return nil
}

// targetMarkerIndent sets a target in from the package heading above it. It
// rides on the marker so the row's own text stays free of layout, which keeps
// the match offsets measured against the target name alone.
const targetMarkerIndent = "  "

// markerFor picks the glyph that leads a row and the colour it is drawn in.
//
// A target that has never been run gets a grey arrow but keeps its name in the
// ordinary colour: the marker is the signal, and greying the whole row would
// make the majority of the list — everything not yet run — harder to read than
// the handful of rows that carry news.
func markerFor(row grouping.Row) (string, layout.Style) {
	if row.Kind == grouping.KindGroup {
		return layout.MarkerGroup, layout.StyleGroup
	}
	switch row.Status {
	case runlog.StatusPassed:
		return targetMarkerIndent + layout.MarkerPassed, layout.StyleSuccess
	case runlog.StatusFailed:
		return targetMarkerIndent + layout.MarkerFailed, layout.StyleFailure
	default:
		return targetMarkerIndent + layout.MarkerUnrun, layout.StyleDim
	}
}

// styleForStatus is where a run outcome becomes a colour. A target that has
// never been run is drawn normally rather than in a third colour: "no news" and
// "passed" must not look alike.
func styleForStatus(status runlog.Status) layout.Style {
	switch status {
	case runlog.StatusPassed:
		return layout.StyleSuccess
	case runlog.StatusFailed:
		return layout.StyleFailure
	default:
		return layout.StyleNormal
	}
}

func actionLabels(actions []action.Action) []string {
	labels := make([]string, 0, len(actions))
	for _, entry := range actions {
		labels = append(labels, entry.Label())
	}
	return labels
}
