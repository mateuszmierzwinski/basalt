package tui

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/keys"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolchain"
)

// ActionsProvider supplies the actions available for a target. Deciding them
// needs the filesystem — which Markdown files exist in the package — so the
// machine takes it as an interface rather than reaching for disk itself.
type ActionsProvider interface {
	Actions(selected target.Target) []action.Action
}

// ToolchainProber reports what the selection is written in and which compilers
// answer for it. It may be nil, in which case the header line is simply absent.
type ToolchainProber interface {
	SummaryFor(selected target.Target, languages []target.Language) toolchain.Summary
}

// Machine advances the UI state in response to input.
type Machine interface {
	// Apply returns the state produced by a keystroke and what the session
	// should do next.
	Apply(state State, key keys.Key) (State, Outcome)
	// Refilter recomputes the matches after the target set or query changed.
	Refilter(state State) State
	// View projects state into what the renderer draws.
	View(state State) layout.View
}

// KeyHandler is one binding. Every entry in the handler map is a named struct
// implementing this interface — never a raw func (Rule 6.3).
type KeyHandler interface {
	Handle(machine *machine, state State) (State, Outcome)
}
