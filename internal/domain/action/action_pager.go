package action

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// lessAction opens the output kept from the target's last bazel command.
//
// The acknowledgement prompt catches the result while it is on screen; this is
// for afterwards — coming back to a failure once you have read the code, or
// checking what a build actually said an hour later.
type lessAction struct{}

func (a *lessAction) ID() string    { return IDLessLog }
func (a *lessAction) Label() string { return "Less > Last run log" }
func (a *lessAction) Kind() Kind    { return KindEditor }

// Applies only when there is a log to read. A target that has never been run,
// or whose log has since been deleted, simply does not offer the action rather
// than offering one that opens an empty pager.
func (a *lessAction) Applies(ctx Context) bool {
	return ctx.Tools.Has(toolset.ToolLess) && ctx.LastLog != ""
}

// Command opens the log from the monorepo root. The path comes from the run
// history, which the history service has already proven lies inside the cache
// directory — nothing here constructs a path from a label.
func (a *lessAction) Command(ctx Context) (port.CommandSpec, error) {
	return toolCommand(ctx, toolset.ToolLess, ctx.Root, ctx.LastLog)
}

// newLessAction builds the log-viewing action.
func newLessAction() Action {
	return &lessAction{}
}
