package action

import "github.com/mateuszmierzwinski/basalt/internal/port"

// Action is one operation offered for a target. Every action is a named struct
// implementing this interface and held in a package-scoped map — never a raw
// func reference (Rule 6.3).
type Action interface {
	// ID is the stable identifier used as the registry key.
	ID() string
	// Label is what the action panel displays.
	Label() string
	// Kind groups the action for ordering.
	Kind() Kind
	// Applies reports whether the action makes sense for this target with
	// these tools available.
	Applies(ctx Context) bool
	// Command builds the invocation. It is only called for an action that
	// Applies.
	Command(ctx Context) (port.CommandSpec, error)
}

// Registry produces the actions available for a target.
type Registry interface {
	For(ctx Context) []Action
}
