package action

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// dangerousSkipFlag is Claude Code's permission bypass. It is a separate,
// explicitly-labelled action rather than a default, so choosing it is always a
// deliberate act.
const dangerousSkipFlag = "--dangerously-skip-permissions"

// claudeAction starts Claude Code in the target's directory. The two variants
// differ only in their arguments, so they share one struct (Rule 5.1).
type claudeAction struct {
	id    string
	label string
	args  []string
}

func (a *claudeAction) ID() string    { return a.id }
func (a *claudeAction) Label() string { return a.label }
func (a *claudeAction) Kind() Kind    { return KindClaude }

func (a *claudeAction) Applies(ctx Context) bool {
	return ctx.Tools.Has(toolset.ToolClaude)
}

func (a *claudeAction) Command(ctx Context) (port.CommandSpec, error) {
	return toolCommand(ctx, toolset.ToolClaude, ctx.Directory, a.args...)
}

// newClaude builds the ordinary "Claude > Run here" action.
func newClaude() Action {
	return &claudeAction{id: IDClaudeRun, label: "Claude > Run here"}
}

// newClaudeDSP builds the permission-bypassing variant the brief calls DSP.
func newClaudeDSP() Action {
	return &claudeAction{
		id:    IDClaudeDSP,
		label: "Claude > Run here (DSP)",
		args:  []string{dangerousSkipFlag},
	}
}
