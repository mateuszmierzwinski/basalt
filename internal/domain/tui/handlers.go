package tui

import "github.com/mateuszmierzwinski/basalt/internal/domain/layout"

// quitHandler ends the session.
type quitHandler struct{}

func (h *quitHandler) Handle(_ *machine, state State) (State, Outcome) {
	return state, OutcomeQuit
}

// redrawHandler forces a repaint without touching state.
type redrawHandler struct{}

func (h *redrawHandler) Handle(_ *machine, state State) (State, Outcome) {
	return state, OutcomeRedraw
}

// refreshHandler asks the session to re-index from bazel.
type refreshHandler struct{}

func (h *refreshHandler) Handle(_ *machine, state State) (State, Outcome) {
	state.Status = "re-indexing…"
	state.StatusIsError = false
	return state, OutcomeRefresh
}

// focusHandler moves the focus between the two panels.
type focusHandler struct {
	forward bool
}

func (h *focusHandler) Handle(_ *machine, state State) (State, Outcome) {
	if h.forward {
		state.Focus = layout.Focus(NextFocus(int(state.Focus)))
	} else {
		state.Focus = layout.Focus(PreviousFocus(int(state.Focus)))
	}
	return state, OutcomeContinue
}

// moveHandler steps the focused panel's cursor by a fixed delta. Page keys use
// the same handler with a delta resolved from the viewport.
type moveHandler struct {
	delta int
	page  bool
}

func (h *moveHandler) Handle(machine *machine, state State) (State, Outcome) {
	if state.Focus == layout.FocusActions {
		state.ActionCursor = ClampCursor(state.ActionCursor+h.delta, len(state.Actions))
		return state, OutcomeContinue
	}
	state.Cursor = h.moveTargets(state)
	return machine.afterCursorMove(state), OutcomeContinue
}

// moveTargets steps over package headings, so a press of Down always lands on
// the next target rather than on the heading between them.
func (h *moveHandler) moveTargets(state State) int {
	if h.page && state.Viewport > 1 {
		return PageSelectable(state.Display, state.Cursor, state.Viewport, h.delta)
	}
	return StepSelectable(state.Display, state.Cursor, h.delta)
}

// edgeHandler jumps the focused panel's cursor to the first or last entry.
type edgeHandler struct {
	toEnd bool
}

func (h *edgeHandler) Handle(machine *machine, state State) (State, Outcome) {
	if state.Focus == layout.FocusActions {
		state.ActionCursor = h.position(len(state.Actions))
		return state, OutcomeContinue
	}
	state.Cursor = h.edgeTarget(state)
	return machine.afterCursorMove(state), OutcomeContinue
}

// edgeTarget jumps to the first or last target, never to a heading.
func (h *edgeHandler) edgeTarget(state State) int {
	if h.toEnd {
		return LastSelectable(state.Display)
	}
	return FirstSelectable(state.Display)
}

func (h *edgeHandler) position(length int) int {
	if h.toEnd {
		return ClampCursor(length-1, length)
	}
	return 0
}

// selectHandler runs the highlighted action. From the target panel it moves
// focus to the actions instead, so Enter always means "go forward".
type selectHandler struct{}

func (h *selectHandler) Handle(_ *machine, state State) (State, Outcome) {
	if state.Focus == layout.FocusTargets {
		if len(state.Actions) == 0 {
			return state, OutcomeContinue
		}
		state.Focus = layout.FocusActions
		return state, OutcomeContinue
	}
	chosen, ok := state.SelectedAction()
	if !ok {
		return state, OutcomeContinue
	}
	state.Chosen = chosen
	return state, OutcomeRunAction
}

// escapeHandler backs out one step: from the actions to the targets, from a
// non-empty query to an empty one, and otherwise it quits.
type escapeHandler struct{}

func (h *escapeHandler) Handle(machine *machine, state State) (State, Outcome) {
	if state.Focus == layout.FocusActions {
		state.Focus = layout.FocusTargets
		return state, OutcomeContinue
	}
	if state.Query != "" {
		state.Query = ""
		return machine.Refilter(state), OutcomeContinue
	}
	return state, OutcomeQuit
}

// toggleModeHandler switches literal and regex search.
type toggleModeHandler struct{}

func (h *toggleModeHandler) Handle(machine *machine, state State) (State, Outcome) {
	state.Mode = ToggleMode(state.Mode)
	return machine.Refilter(state), OutcomeContinue
}

// editHandler mutates the query. All three edits — backspace, word delete,
// clear — differ only in the transform they apply (Rule 5.1).
type editHandler struct {
	transform func(query string) string
}

func (h *editHandler) Handle(machine *machine, state State) (State, Outcome) {
	state.Query = h.transform(state.Query)
	return machine.Refilter(state), OutcomeContinue
}
