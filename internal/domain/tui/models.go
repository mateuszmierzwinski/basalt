// Package tui is the state machine: a keystroke goes in, a new state comes
// out. It performs no I/O and knows nothing about terminals, which is what
// makes every binding testable without a pty.
package tui

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/grouping"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Outcome tells the session loop what to do after a keystroke.
type Outcome int

const (
	// OutcomeContinue redraws and waits for the next key.
	OutcomeContinue Outcome = iota
	// OutcomeQuit ends the session.
	OutcomeQuit
	// OutcomeRunAction suspends the UI and runs State.Chosen.
	OutcomeRunAction
	// OutcomeRefresh re-indexes from bazel.
	OutcomeRefresh
	// OutcomeRedraw forces a full repaint without changing state.
	OutcomeRedraw
)

// State is the complete UI state. It is a value: every transition returns a new
// one rather than mutating in place, so a handler cannot leave it half-updated.
type State struct {
	// Targets is the full index; Matches is the filtered view of it; Display is
	// Matches arranged into the package tree that is actually drawn.
	Targets []target.Target
	Matches []filter.Match
	Display []grouping.Row

	// Statuses is how each target's last bazel command went, keyed by label.
	// The list is coloured from it, and the session replaces it after every run.
	Statuses map[string]runlog.Status

	Query string
	Mode  filter.Mode
	Focus layout.Focus

	// Cursor indexes Display and always rests on a selectable row — package
	// headings are labels, not choices. ActionCursor indexes Actions.
	Cursor       int
	ActionCursor int
	Offset       int

	Actions []action.Action

	Status        string
	StatusIsError bool

	// Chosen is the action to run when the outcome is OutcomeRunAction.
	Chosen action.Action

	// Viewport is how many target rows fit on screen, which the session
	// updates on resize so scrolling stays correct.
	Viewport int
}

// Selected returns the highlighted target, and whether there is one at all —
// an empty result set has no selection, and neither does a cursor that has
// somehow come to rest on a heading.
func (s State) Selected() (target.Target, bool) {
	if s.Cursor < 0 || s.Cursor >= len(s.Display) {
		return target.Target{}, false
	}
	row := s.Display[s.Cursor]
	if !row.IsSelectable() {
		return target.Target{}, false
	}
	return row.Target, true
}

// SelectedAction returns the highlighted action, if any.
func (s State) SelectedAction() (action.Action, bool) {
	if s.ActionCursor < 0 || s.ActionCursor >= len(s.Actions) {
		return nil, false
	}
	return s.Actions[s.ActionCursor], true
}
