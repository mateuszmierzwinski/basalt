package session

import (
	"context"

	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/tui"
)

// Session runs the interactive loop until the operator quits.
type Session interface {
	Run(ctx context.Context) error
}

// Actions both lists the actions for a target — which is what the state machine
// needs — and rebuilds the context one needs to run.
type Actions interface {
	tui.ActionsProvider
	Context(selected target.Target) (action.Context, error)
}

// Renderer is layout.Renderer, aliased so the session's dependency list reads
// as one set of names.
type Renderer = layout.Renderer
