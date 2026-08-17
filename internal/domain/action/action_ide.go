package action

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// ideAction opens the target's directory in a JetBrains IDE. GoLand and CLion
// differ only in their tool and the rule kinds they suit (Rule 5.1).
type ideAction struct {
	id    string
	label string
	tool  toolset.Tool
	suits func(kind target.Kind) bool
}

func (a *ideAction) ID() string    { return a.id }
func (a *ideAction) Label() string { return a.label }
func (a *ideAction) Kind() Kind    { return KindIDE }

// Applies gates on both the rule kind and the IDE being installed. Neither
// GoLand nor CLion is present on the operator's machine, so in practice these
// actions are usually absent — which is the intended behaviour, not a fault.
func (a *ideAction) Applies(ctx Context) bool {
	return ctx.Tools.Has(a.tool) && a.suits(ctx.Target.Kind)
}

// Command opens the package directory, which is what the IDE imports as a
// Bazel project.
func (a *ideAction) Command(ctx Context) (port.CommandSpec, error) {
	return toolCommand(ctx, a.tool, ctx.Directory, ctx.Directory)
}

// newGoland builds the GoLand action, offered for Go rules.
func newGoland() Action {
	return &ideAction{
		id:    IDGoland,
		label: "Goland > Open as bazel project",
		tool:  toolset.ToolGoland,
		suits: IsGo,
	}
}

// newClion builds the CLion action, offered for C/C++ rules.
func newClion() Action {
	return &ideAction{
		id:    IDClion,
		label: "Clion > Open as bazel project",
		tool:  toolset.ToolClion,
		suits: IsNative,
	}
}
